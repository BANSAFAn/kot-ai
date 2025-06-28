package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config содержит все настройки приложения
type Config struct {
	AssistantConfig AssistantConfig `json:"assistant"`
	VoiceConfig     VoiceConfig     `json:"voice"`
	UIConfig        UIConfig        `json:"ui"`
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

// VoiceConfig содержит настройки голосового модуля
type VoiceConfig struct {
	Enabled           bool    `json:"enabled"`
	WakeWord          string  `json:"wake_word"`
	Language          string  `json:"language"`
	VoiceRecognition  string  `json:"voice_recognition"` // google, local, whisper
	TTSProvider       string  `json:"tts_provider"`      // google, local
	VoiceThreshold    float64 `json:"voice_threshold"`
	SilenceThreshold  float64 `json:"silence_threshold"`
	InputDevice       string  `json:"input_device"`
	OutputDevice      string  `json:"output_device"`
}

// UIConfig содержит настройки пользовательского интерфейса
type UIConfig struct {
	Enabled      bool   `json:"enabled"`
	UIType       string `json:"ui_type"` // web, tray, console
	WebPort      int    `json:"web_port"`
	Theme        string `json:"theme"`
	StartMinimized bool   `json:"start_minimized"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	historyPath := filepath.Join(homeDir, ".kot.ai", "history.db")

	return &Config{
		AssistantConfig: AssistantConfig{
			Name:            "KOT.AI",
			OpenAIAPIKey:    "",
			GoogleAPIKey:    "",
			UseLocalModels:  false,
			LocalModelPath:  "",
			HistoryEnabled:  true,
			HistoryFilePath: historyPath,
		},
		VoiceConfig: VoiceConfig{
			Enabled:          true,
			WakeWord:         "кот",
			Language:         "ru-RU",
			VoiceRecognition: "google",
			TTSProvider:      "google",
			VoiceThreshold:   0.5,
			SilenceThreshold: 0.1,
			InputDevice:      "",
			OutputDevice:     "",
		},
		UIConfig: UIConfig{
			Enabled:        true,
			UIType:         "web",
			WebPort:        8080,
			Theme:          "dark",
			StartMinimized: false,
		},
	}
}

// Load загружает конфигурацию из файла
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".kot.ai")
	configPath := filepath.Join(configDir, "config.json")

	// Создаем директорию, если она не существует
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, err
		}
	}

	// Проверяем существование файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Создаем конфигурацию по умолчанию
		defaultCfg := DefaultConfig()

		// Сохраняем конфигурацию по умолчанию
		data, err := json.MarshalIndent(defaultCfg, "", "  ")
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, err
		}

		return defaultCfg, nil
	}

	// Загружаем существующую конфигурацию
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save сохраняет конфигурацию в файл
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".kot.ai")
	configPath := filepath.Join(configDir, "config.json")

	// Создаем директорию, если она не существует
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
	}

	// Сохраняем конфигурацию
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}