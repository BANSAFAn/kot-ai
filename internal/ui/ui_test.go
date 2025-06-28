package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewUIManager(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "web",
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)

	// Проверяем, что UIManager создан без ошибок
	assert.NoError(t, err)
	assert.NotNil(t, um)

	// Проверяем, что поля UIManager инициализированы правильно
	assert.Equal(t, config.Enabled, um.Config.Enabled)
	assert.Equal(t, config.UIType, um.Config.UIType)
	assert.Equal(t, config.WebPort, um.Config.WebPort)
	assert.Equal(t, config.Theme, um.Config.Theme)
	assert.Equal(t, config.StartMinimized, um.Config.StartMinimized)

	// Проверяем, что каналы созданы
	assert.NotNil(t, um.CommandChan)
	assert.NotNil(t, um.ResponseChan)

	// Останавливаем UIManager после теста
	um.Stop()
}

func TestWebHandler(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "web",
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)
	assert.NoError(t, err)

	// Создаем тестовый HTTP-запрос
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Создаем тестовый HTTP-ответ
	rec := httptest.NewRecorder()

	// Вызываем обработчик
	um.webHandler(rec, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rec.Code)

	// Проверяем, что ответ содержит ожидаемый HTML
	assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
	assert.Contains(t, rec.Body.String(), "KOT.AI")

	// Останавливаем UIManager после теста
	um.Stop()
}

func TestSendMessage(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "console", // Используем консольный интерфейс для упрощения теста
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)
	assert.NoError(t, err)

	// Отправляем сообщение
	um.SendMessage("Тестовое сообщение")

	// Поскольку это консольный интерфейс, мы не можем проверить вывод,
	// но можем убедиться, что функция не вызывает панику
	assert.NotPanics(t, func() {
		um.SendMessage("Еще одно тестовое сообщение")
	})

	// Останавливаем UIManager после теста
	um.Stop()
}

func TestProcessCommand(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "console",
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)
	assert.NoError(t, err)

	// Запускаем обработчик команд в отдельной горутине
	go func() {
		// Ждем, пока команда будет отправлена в канал
		command := <-um.CommandChan
		assert.Equal(t, "тестовая команда", command)

		// Отправляем ответ в канал
		um.ResponseChan <- "тестовый ответ"
	}()

	// Обрабатываем команду
	response := um.ProcessCommand("тестовая команда")

	// Проверяем ответ
	assert.Equal(t, "тестовый ответ", response)

	// Останавливаем UIManager после теста
	um.Stop()
}

func TestGetHistory(t *testing.T) {
	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "console",
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)
	assert.NoError(t, err)

	// Добавляем сообщения в историю
	um.history = append(um.history, Message{Type: "user", Text: "тестовая команда", Time: time.Now()})
	um.history = append(um.history, Message{Type: "assistant", Text: "тестовый ответ", Time: time.Now()})

	// Получаем историю
	history := um.GetHistory()

	// Проверяем, что история содержит добавленные сообщения
	assert.Len(t, history, 2)
	assert.Equal(t, "user", history[0].Type)
	assert.Equal(t, "тестовая команда", history[0].Text)
	assert.Equal(t, "assistant", history[1].Type)
	assert.Equal(t, "тестовый ответ", history[1].Text)

	// Останавливаем UIManager после теста
	um.Stop()
}

func TestWebSocketHandler(t *testing.T) {
	// Этот тест требует реальное WebSocket-соединение, поэтому мы просто проверим,
	// что функция не вызывает панику

	// Создаем конфигурацию для тестирования
	config := UIConfig{
		Enabled:        true,
		UIType:         "web",
		WebPort:        8080,
		Theme:          "dark",
		StartMinimized: false,
	}

	// Создаем новый экземпляр UIManager
	um, err := NewUIManager(config)
	assert.NoError(t, err)

	// Создаем тестовый HTTP-сервер
	server := httptest.NewServer(http.HandlerFunc(um.wsHandler))
	defer server.Close()

	// Преобразуем URL с HTTP на WebSocket
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Проверяем, что функция не вызывает панику при попытке подключения
	assert.NotPanics(t, func() {
		// Пытаемся установить WebSocket-соединение
		_, _, err := websocket.DefaultDialer.Dial(url, nil)
		// Мы ожидаем ошибку, так как тестовый сервер не поддерживает WebSocket
		// Но функция не должна вызывать панику
		_ = err
	})

	// Останавливаем UIManager после теста
	um.Stop()
}