package voice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewVoiceManager(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := VoiceConfig{
		Enabled:          true,
		WakeWord:         "тест",
		Language:         "ru-RU",
		VoiceRecognition: "google",
		TTSProvider:      "google",
		VoiceThreshold:   0.5,
		SilenceThreshold: 0.1,
		InputDevice:      "",
		OutputDevice:     "",
	}

	// Создаем новый экземпляр VoiceManager
	vm, err := NewVoiceManager(config)

	// Проверяем, что VoiceManager создан без ошибок
	assert.NoError(t, err)
	assert.NotNil(t, vm)

	// Проверяем, что поля VoiceManager инициализированы правильно
	assert.Equal(t, config.Enabled, vm.Config.Enabled)
	assert.Equal(t, config.WakeWord, vm.Config.WakeWord)
	assert.Equal(t, config.Language, vm.Config.Language)
	assert.Equal(t, config.VoiceRecognition, vm.Config.VoiceRecognition)
	assert.Equal(t, config.TTSProvider, vm.Config.TTSProvider)
	assert.Equal(t, config.VoiceThreshold, vm.Config.VoiceThreshold)
	assert.Equal(t, config.SilenceThreshold, vm.Config.SilenceThreshold)

	// Проверяем, что каналы созданы
	assert.NotNil(t, vm.CommandChan)
	assert.NotNil(t, vm.ResponseChan)

	// Останавливаем VoiceManager после теста
	vm.Stop()
}

func TestDetectWakeWord(t *testing.T) {
	// Этот тест требует реальный аудиовход, поэтому мы просто проверим,
	// что функция не вызывает панику и возвращает ожидаемый тип

	// Создаем конфигурацию для тестирования
	config := VoiceConfig{
		Enabled:          true,
		WakeWord:         "тест",
		Language:         "ru-RU",
		VoiceRecognition: "google",
		TTSProvider:      "google",
		VoiceThreshold:   0.5,
		SilenceThreshold: 0.1,
	}

	// Создаем новый экземпляр VoiceManager
	vm, err := NewVoiceManager(config)
	assert.NoError(t, err)

	// Устанавливаем короткий таймаут для теста
	vm.wakeWordTimeout = 100 * time.Millisecond

	// Запускаем детектор ключевого слова в отдельной горутине
	go func() {
		// Останавливаем детектор через короткое время
		time.Sleep(200 * time.Millisecond)
		vm.Stop()
	}()

	// Проверяем, что функция не вызывает панику
	assert.NotPanics(t, func() {
		vm.detectWakeWord()
	})
}

func TestTextToSpeech(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := VoiceConfig{
		Enabled:          true,
		WakeWord:         "тест",
		Language:         "ru-RU",
		VoiceRecognition: "google",
		TTSProvider:      "local", // Используем локальный провайдер для теста
		VoiceThreshold:   0.5,
		SilenceThreshold: 0.1,
	}

	// Создаем новый экземпляр VoiceManager
	vm, err := NewVoiceManager(config)
	assert.NoError(t, err)

	// Тестируем преобразование текста в речь
	// Поскольку это требует реальный аудиовыход, мы просто проверим,
	// что функция не вызывает панику
	assert.NotPanics(t, func() {
		vm.TextToSpeech("Это тестовое сообщение")
	})

	// Останавливаем VoiceManager после теста
	vm.Stop()
}

func TestListenForCommand(t *testing.T) {
	// Этот тест требует реальный аудиовход, поэтому мы просто проверим,
	// что функция не вызывает панику и возвращает ожидаемый тип

	// Создаем конфигурацию для тестирования
	config := VoiceConfig{
		Enabled:          true,
		WakeWord:         "тест",
		Language:         "ru-RU",
		VoiceRecognition: "local", // Используем локальный провайдер для теста
		TTSProvider:      "local",
		VoiceThreshold:   0.5,
		SilenceThreshold: 0.1,
	}

	// Создаем новый экземпляр VoiceManager
	vm, err := NewVoiceManager(config)
	assert.NoError(t, err)

	// Устанавливаем короткий таймаут для теста
	vm.commandTimeout = 100 * time.Millisecond

	// Запускаем слушатель команд в отдельной горутине
	go func() {
		// Останавливаем слушатель через короткое время
		time.Sleep(200 * time.Millisecond)
		vm.Stop()
	}()

	// Проверяем, что функция не вызывает панику
	assert.NotPanics(t, func() {
		vm.listenForCommand()
	})
}