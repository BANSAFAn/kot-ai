package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Создаем временный файл для конфигурации
	tempDir, err := os.MkdirTemp("", "kot-ai-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Создаем тестовую конфигурацию
	testConfig := Config{
		Assistant: AssistantConfig{
			Name:           "TestAssistant",
			OpenAIAPIKey:   "test-key",
			GoogleAPIKey:   "test-google-key",
			UseLocalModels: false,
			LocalModelPath: "",
			HistoryEnabled: true,
			HistoryPath:    filepath.Join(tempDir, "history.db"),
		},
		Voice: VoiceConfig{
			Enabled:          true,
			WakeWord:         "тест",
			Language:         "ru-RU",
			VoiceRecognition: "google",
			TTSProvider:      "google",
			VoiceThreshold:   0.5,
			SilenceThreshold: 0.1,
			InputDevice:      "",
			OutputDevice:     "",
		},
		UI: UIConfig{
			Enabled:        true,
			UIType:         "web",
			WebPort:        8080,
			Theme:          "dark",
			StartMinimized: false,
		},
	}

	// Сохраняем тестовую конфигурацию
	err = SaveConfig(testConfig, configPath)
	assert.NoError(t, err)

	// Загружаем конфигурацию
	loadedConfig, err := LoadConfig(configPath)
	assert.NoError(t, err)

	// Проверяем, что загруженная конфигурация соответствует сохраненной
	assert.Equal(t, testConfig.Assistant.Name, loadedConfig.Assistant.Name)
	assert.Equal(t, testConfig.Assistant.OpenAIAPIKey, loadedConfig.Assistant.OpenAIAPIKey)
	assert.Equal(t, testConfig.Assistant.GoogleAPIKey, loadedConfig.Assistant.GoogleAPIKey)
	assert.Equal(t, testConfig.Assistant.UseLocalModels, loadedConfig.Assistant.UseLocalModels)
	assert.Equal(t, testConfig.Assistant.LocalModelPath, loadedConfig.Assistant.LocalModelPath)
	assert.Equal(t, testConfig.Assistant.HistoryEnabled, loadedConfig.Assistant.HistoryEnabled)
	assert.Equal(t, testConfig.Assistant.HistoryPath, loadedConfig.Assistant.HistoryPath)

	assert.Equal(t, testConfig.Voice.Enabled, loadedConfig.Voice.Enabled)
	assert.Equal(t, testConfig.Voice.WakeWord, loadedConfig.Voice.WakeWord)
	assert.Equal(t, testConfig.Voice.Language, loadedConfig.Voice.Language)
	assert.Equal(t, testConfig.Voice.VoiceRecognition, loadedConfig.Voice.VoiceRecognition)
	assert.Equal(t, testConfig.Voice.TTSProvider, loadedConfig.Voice.TTSProvider)
	assert.Equal(t, testConfig.Voice.VoiceThreshold, loadedConfig.Voice.VoiceThreshold)
	assert.Equal(t, testConfig.Voice.SilenceThreshold, loadedConfig.Voice.SilenceThreshold)
	assert.Equal(t, testConfig.Voice.InputDevice, loadedConfig.Voice.InputDevice)
	assert.Equal(t, testConfig.Voice.OutputDevice, loadedConfig.Voice.OutputDevice)

	assert.Equal(t, testConfig.UI.Enabled, loadedConfig.UI.Enabled)
	assert.Equal(t, testConfig.UI.UIType, loadedConfig.UI.UIType)
	assert.Equal(t, testConfig.UI.WebPort, loadedConfig.UI.WebPort)
	assert.Equal(t, testConfig.UI.Theme, loadedConfig.UI.Theme)
	assert.Equal(t, testConfig.UI.StartMinimized, loadedConfig.UI.StartMinimized)
}

func TestLoadConfigNotExist(t *testing.T) {
	// Создаем временный путь для несуществующего файла конфигурации
	tempDir, err := os.MkdirTemp("", "kot-ai-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "non-existent-config.json")

	// Загружаем конфигурацию из несуществующего файла
	config, err := LoadConfig(configPath)

	// Проверяем, что возвращена ошибка
	assert.Error(t, err)
	// Проверяем, что возвращена пустая конфигурация
	assert.Equal(t, Config{}, config)
}

func TestSaveConfig(t *testing.T) {
	// Создаем временный файл для конфигурации
	tempDir, err := os.MkdirTemp("", "kot-ai-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Создаем тестовую конфигурацию
	testConfig := Config{
		Assistant: AssistantConfig{
			Name:           "TestAssistant",
			OpenAIAPIKey:   "test-key",
			GoogleAPIKey:   "test-google-key",
			UseLocalModels: false,
			LocalModelPath: "",
			HistoryEnabled: true,
			HistoryPath:    filepath.Join(tempDir, "history.db"),
		},
		Voice: VoiceConfig{
			Enabled:          true,
			WakeWord:         "тест",
			Language:         "ru-RU",
			VoiceRecognition: "google",
			TTSProvider:      "google",
			VoiceThreshold:   0.5,
			SilenceThreshold: 0.1,
			InputDevice:      "",
			OutputDevice:     "",
		},
		UI: UIConfig{
			Enabled:        true,
			UIType:         "web",
			WebPort:        8080,
			Theme:          "dark",
			StartMinimized: false,
		},
	}

	// Сохраняем тестовую конфигурацию
	err = SaveConfig(testConfig, configPath)
	assert.NoError(t, err)

	// Проверяем, что файл конфигурации создан
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Проверяем содержимое файла
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "TestAssistant")
	assert.Contains(t, string(content), "test-key")
	assert.Contains(t, string(content), "тест")
	assert.Contains(t, string(content), "ru-RU")
	assert.Contains(t, string(content), "web")
	assert.Contains(t, string(content), "8080")
}

func TestGetDefaultConfig(t *testing.T) {
	// Получаем конфигурацию по умолчанию
	config := GetDefaultConfig()

	// Проверяем, что конфигурация содержит ожидаемые значения
	assert.Equal(t, "KOT.AI", config.Assistant.Name)
	assert.Equal(t, "", config.Assistant.OpenAIAPIKey) // Ключ API должен быть пустым по умолчанию
	assert.Equal(t, false, config.Assistant.UseLocalModels)
	assert.Equal(t, true, config.Assistant.HistoryEnabled)

	assert.Equal(t, true, config.Voice.Enabled)
	assert.Equal(t, "кот", config.Voice.WakeWord)
	assert.Equal(t, "ru-RU", config.Voice.Language)

	assert.Equal(t, true, config.UI.Enabled)
	assert.Equal(t, "web", config.UI.UIType)
	assert.Equal(t, 8080, config.UI.WebPort)
	assert.Equal(t, "dark", config.UI.Theme)
	assert.Equal(t, false, config.UI.StartMinimized)
}

func TestGetConfigPath(t *testing.T) {
	// Получаем путь к конфигурации
	configPath := GetConfigPath()

	// Проверяем, что путь не пустой
	assert.NotEmpty(t, configPath)

	// Проверяем, что путь содержит ожидаемые компоненты
	assert.Contains(t, configPath, ".kot.ai")
	assert.Contains(t, configPath, "config.json")
}

func TestEnsureConfigDir(t *testing.T) {
	// Создаем временный путь для директории конфигурации
	tempDir, err := os.MkdirTemp("", "kot-ai-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, ".kot.ai")

	// Проверяем, что директория не существует
	_, err = os.Stat(configDir)
	assert.True(t, os.IsNotExist(err))

	// Создаем директорию конфигурации
	err = ensureConfigDir(configDir)
	assert.NoError(t, err)

	// Проверяем, что директория создана
	_, err = os.Stat(configDir)
	assert.NoError(t, err)
}

func TestInitConfig(t *testing.T) {
	// Создаем временный путь для директории конфигурации
	tempDir, err := os.MkdirTemp("", "kot-ai-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Устанавливаем временный путь для конфигурации
	oldConfigDir := configDir
	configDir = filepath.Join(tempDir, ".kot.ai")
	defer func() { configDir = oldConfigDir }()

	// Инициализируем конфигурацию
	config, err := InitConfig()
	assert.NoError(t, err)

	// Проверяем, что конфигурация инициализирована
	assert.NotEmpty(t, config.Assistant.Name)
	assert.NotEmpty(t, config.Voice.WakeWord)
	assert.NotEmpty(t, config.UI.UIType)

	// Проверяем, что файл конфигурации создан
	configPath := filepath.Join(configDir, "config.json")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}