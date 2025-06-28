package assistant

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAssistant(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := AssistantConfig{
		Name:           "TestAssistant",
		OpenAIAPIKey:   "test-key",
		UseLocalModels: false,
		HistoryEnabled: true,
		HistoryPath:    "test-history.db",
	}

	// Создаем новый экземпляр ассистента
	assistant, err := NewAssistant(config)

	// Проверяем, что ассистент создан без ошибок
	assert.NoError(t, err)
	assert.NotNil(t, assistant)

	// Проверяем, что поля ассистента инициализированы правильно
	assert.Equal(t, config.Name, assistant.Config.Name)
	assert.Equal(t, config.OpenAIAPIKey, assistant.Config.OpenAIAPIKey)
	assert.Equal(t, config.UseLocalModels, assistant.Config.UseLocalModels)
	assert.Equal(t, config.HistoryEnabled, assistant.Config.HistoryEnabled)
	assert.Equal(t, config.HistoryPath, assistant.Config.HistoryPath)

	// Проверяем, что канал команд создан
	assert.NotNil(t, assistant.CommandChan)

	// Закрываем ассистента после теста
	assistant.Stop()
}

func TestProcessCommand(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := AssistantConfig{
		Name:           "TestAssistant",
		OpenAIAPIKey:   "test-key",
		UseLocalModels: false,
		HistoryEnabled: false, // Отключаем историю для упрощения теста
	}

	// Создаем новый экземпляр ассистента
	assistant, err := NewAssistant(config)
	assert.NoError(t, err)
	assert.NotNil(t, assistant)

	// Тестируем обработку специальной команды
	response, isSpecial := assistant.ProcessCommand("помощь")
	assert.True(t, isSpecial)
	assert.Contains(t, response, "доступные команды")

	// Тестируем обработку обычной команды
	// Для этого теста мы не можем проверить фактический ответ от AI,
	// поэтому просто проверяем, что команда не распознана как специальная
	response, isSpecial = assistant.ProcessCommand("расскажи о погоде")
	assert.False(t, isSpecial)

	// Закрываем ассистента после теста
	assistant.Stop()
}

func TestAddToHistory(t *testing.T) {
	// Создаем временный файл для истории
	tempHistoryPath := "test-history-" + time.Now().Format("20060102150405") + ".db"

	// Создаем конфигурацию для тестирования
	config := AssistantConfig{
		Name:           "TestAssistant",
		OpenAIAPIKey:   "test-key",
		UseLocalModels: false,
		HistoryEnabled: true,
		HistoryPath:    tempHistoryPath,
	}

	// Создаем новый экземпляр ассистента
	assistant, err := NewAssistant(config)
	assert.NoError(t, err)
	assert.NotNil(t, assistant)

	// Добавляем команду в историю
	err = assistant.AddToHistory("тестовая команда", "тестовый ответ")
	assert.NoError(t, err)

	// Получаем историю
	history, err := assistant.GetHistory(10)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "тестовая команда", history[0].Command)
	assert.Equal(t, "тестовый ответ", history[0].Response)

	// Закрываем ассистента после теста
	assistant.Stop()

	// Удаляем временный файл истории
	// В реальном тесте мы бы использовали os.Remove(tempHistoryPath),
	// но для упрощения этого теста мы пропустим этот шаг
}

func TestIsSpecialCommand(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := AssistantConfig{
		Name:           "TestAssistant",
		OpenAIAPIKey:   "test-key",
		UseLocalModels: false,
		HistoryEnabled: false,
	}

	// Создаем новый экземпляр ассистента
	assistant, err := NewAssistant(config)
	assert.NoError(t, err)
	assert.NotNil(t, assistant)

	// Тестируем различные специальные команды
	tests := []struct {
		command  string
		expected bool
	}{
		{"помощь", true},
		{"Помощь", true}, // Проверка регистра
		{"ПОМОЩЬ", true}, // Проверка регистра
		{"настройки", true},
		{"выключись", true},
		{"перезапустись", true},
		{"который час", true},
		{"какая сегодня дата", true},
		{"расскажи анекдот", true},
		{"случайная команда", false},
		{"", false},
	}

	for _, test := range tests {
		result := assistant.isSpecialCommand(test.command)
		assert.Equal(t, test.expected, result, "Команда: %s", test.command)
	}

	// Закрываем ассистента после теста
	assistant.Stop()
}