package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/ztrue/tracerr"

	"kot.ai/internal/drawing"
	"kot.ai/internal/bank"
	"kot.ai/internal/steam"
	"kot.ai/internal/swift"
	"kot.ai/internal/system"
	"kot.ai/internal/voice"
)

// Assistant представляет основную логику ассистента
type Assistant struct {
	config       AssistantConfig
	system       *system.SystemManager
	voice        *voice.VoiceManager
	openAIClient *openai.Client
	db           *leveldb.DB
	isRunning    bool
	mutex        sync.Mutex
}

// AssistantConfig содержит настройки ассистента
type AssistantConfig struct {
	Name            string `json:"name"`
	OpenAIAPIKey    string `json:"openai_api_key"`
	GoogleAPIKey    string `json:"google_api_key"`
	UseLocalModels  bool   `json:"use_local_models"`
	LocalModelPath  string `json:"local_model_path"`
	HistoryEnabled  bool   `json:"history_enabled"`
	HistoryFilePath string `json:"history_file_path"`
}

// HistoryEntry представляет запись в истории команд
type HistoryEntry struct {
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
	Response  string `json:"response"`
}

// NewAssistant создает новый экземпляр Assistant
func NewAssistant(config AssistantConfig, system *system.SystemManager, voice *voice.VoiceManager) *Assistant {
	return &Assistant{
		config: config,
		system: system,
		voice:  voice,
	}
}

// Start запускает ассистента
func (a *Assistant) Start() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.isRunning {
		return nil
	}

	// Инициализация OpenAI клиента
	if a.config.OpenAIAPIKey != "" {
		a.openAIClient = openai.NewClient(a.config.OpenAIAPIKey)
	}

	// Инициализация базы данных истории
	if a.config.HistoryEnabled && a.config.HistoryFilePath != "" {
		// Создаем директорию, если она не существует
		dir := filepath.Dir(a.config.HistoryFilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return tracerr.Wrap(err)
			}
		}

		// Открываем базу данных
		db, err := leveldb.OpenFile(a.config.HistoryFilePath, nil)
		if err != nil {
			return tracerr.Wrap(err)
		}
		a.db = db
	}

	// Устанавливаем обработчик голосовых команд
	a.voice.SetCommandCallback(func(command string) {
		response, err := a.ProcessCommand(command)
		if err != nil {
			log.Printf("Ошибка обработки голосовой команды: %v", err)
			a.voice.Speak("Извините, произошла ошибка при обработке команды")
			return
		}

		// Произносим ответ
		a.voice.Speak(response)
	})

	a.isRunning = true
	return nil
}

// Stop останавливает ассистента
func (a *Assistant) Stop() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.isRunning {
		return
	}

	// Закрываем базу данных
	if a.db != nil {
		a.db.Close()
		a.db = nil
	}

	a.isRunning = false
}

// ProcessCommand обрабатывает команду пользователя
func (a *Assistant) ProcessCommand(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "Пожалуйста, укажите команду", nil
	}

	// Проверяем специальные команды
	response, handled := a.handleSpecialCommands(command)
	if handled {
		// Сохраняем в историю
		a.saveToHistory(command, response)
		return response, nil
	}

	// Обрабатываем команду с помощью AI
	response, err := a.processWithAI(command)
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	// Сохраняем в историю
	a.saveToHistory(command, response)

	return response, nil
}

// handleSpecialCommands обрабатывает специальные команды
func (a *Assistant) handleSpecialCommands(command string) (string, bool) {
	lowerCmd := strings.ToLower(command)

	for _, cmd := range commandRegistry {
		for _, keyword := range cmd.Keywords {
			if strings.HasPrefix(lowerCmd, keyword) {
				// Извлекаем аргументы, удаляя ключевое слово и лишние пробелы
				argsStr := strings.TrimSpace(strings.TrimPrefix(lowerCmd, keyword))
				var args []string
				if argsStr != "" {
					args = strings.Split(argsStr, " ")
				}
				return cmd.Handler(a, args)
			}
		}
	}

	// Команда не распознана как специальная
	return "", false
}

// processWithAI обрабатывает команду с помощью AI
func (a *Assistant) processWithAI(command string) (string, error) {
	if a.openAIClient == nil {
		return "Для обработки команд необходим API ключ OpenAI", nil
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Формируем системное сообщение
	systemMessage := fmt.Sprintf(
		"Ты - %s, персональный голосовой ассистент для Windows. "+
		"Ты можешь выполнять команды для управления компьютером, "+
		"отвечать на вопросы и помогать пользователю. "+
		"Отвечай кратко и по существу. "+
		"Текущее время: %s.",
		a.config.Name,
		time.Now().Format("15:04 02.01.2006"),
	)

	// Отправляем запрос в OpenAI API
	resp, err := a.openAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemMessage,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: command,
				},
			},
			Temperature: 0.7,
		},
	)
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	if len(resp.Choices) == 0 {
		return "Извините, я не смог обработать ваш запрос", nil
	}

	return resp.Choices[0].Message.Content, nil
}

// saveToHistory сохраняет команду и ответ в историю
func (a *Assistant) saveToHistory(command, response string) {
	if a.db == nil || !a.config.HistoryEnabled {
		return
	}

	// Создаем запись
	entry := HistoryEntry{
		Timestamp: time.Now().Unix(),
		Command:   command,
		Response:  response,
	}

	// Сериализуем запись
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Ошибка сериализации записи истории: %v", err)
		return
	}

	// Создаем ключ на основе временной метки
	key := fmt.Sprintf("%d", entry.Timestamp)

	// Сохраняем запись
	err = a.db.Put([]byte(key), data, nil)
	if err != nil {
		log.Printf("Ошибка сохранения записи истории: %v", err)
	}
}

// GetHistory возвращает историю команд
func (a *Assistant) GetHistory() ([]HistoryEntry, error) {
	if a.db == nil || !a.config.HistoryEnabled {
		return nil, nil
	}

	var history []HistoryEntry

	// Итерируемся по всем записям
	iter := a.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		// Десериализуем запись
		var entry HistoryEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}

		history = append(history, entry)
	}

	if err := iter.Error(); err != nil {
		return nil, tracerr.Wrap(err)
	}

	return history, nil
}