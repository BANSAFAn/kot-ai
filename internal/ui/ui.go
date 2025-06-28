package ui

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zserge/lorca"
	"github.com/ztrue/tracerr"

	"kot.ai/internal/assistant"
)

//go:embed web
var webContent embed.FS

// UIManager управляет пользовательским интерфейсом
type UIManager struct {
	config      UIConfig
	assistant   *assistant.Assistant
	ui          lorca.UI
	server      *http.Server
	clients     map[*websocket.Conn]bool
	clientMutex sync.Mutex
	upgrader    websocket.Upgrader
	isRunning   bool
}

// UIConfig содержит настройки пользовательского интерфейса
type UIConfig struct {
	Enabled        bool   `json:"enabled"`
	UIType         string `json:"ui_type"` // web, tray, console
	WebPort        int    `json:"web_port"`
	Theme          string `json:"theme"`
	StartMinimized bool   `json:"start_minimized"`
}

// NewUIManager создает новый экземпляр UIManager
func NewUIManager(config UIConfig, assistant *assistant.Assistant) *UIManager {
	return &UIManager{
		config:    config,
		assistant: assistant,
		clients:   make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Разрешаем все источники для простоты
			},
		},
	}
}

// Start запускает пользовательский интерфейс
func (um *UIManager) Start() error {
	if !um.config.Enabled {
		log.Println("Пользовательский интерфейс отключен в настройках")
		return nil
	}

	switch um.config.UIType {
	case "web":
		return um.startWebUI()
	case "tray":
		return um.startTrayUI()
	case "console":
		return um.startConsoleUI()
	default:
		return um.startWebUI()
	}
}

// Stop останавливает пользовательский интерфейс
func (um *UIManager) Stop() {
	um.isRunning = false

	// Закрываем все WebSocket соединения
	um.clientMutex.Lock()
	for client := range um.clients {
		client.Close()
		delete(um.clients, client)
	}
	um.clientMutex.Unlock()

	// Останавливаем HTTP сервер
	if um.server != nil {
		um.server.Close()
		um.server = nil
	}

	// Закрываем UI
	if um.ui != nil {
		um.ui.Close()
		um.ui = nil
	}
}

// SendMessage отправляет сообщение всем подключенным клиентам
func (um *UIManager) SendMessage(message map[string]interface{}) {
	um.clientMutex.Lock()
	defer um.clientMutex.Unlock()

	for client := range um.clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			client.Close()
			delete(um.clients, client)
		}
	}
}

// startWebUI запускает веб-интерфейс
func (um *UIManager) startWebUI() error {
	// Создаем временную директорию для веб-файлов
	tmpDir, err := os.MkdirTemp("", "kot_web")
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Извлекаем веб-файлы из встроенного FS
	if err := um.extractWebFiles(tmpDir); err != nil {
		return tracerr.Wrap(err)
	}

	// Настраиваем HTTP сервер
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(tmpDir)))
	mux.HandleFunc("/ws", um.handleWebSocket)

	// Запускаем HTTP сервер
	um.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", um.config.WebPort),
		Handler: mux,
	}

	go func() {
		log.Printf("Веб-интерфейс запущен на http://localhost:%d", um.config.WebPort)
		if err := um.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка HTTP сервера: %v", err)
		}
	}()

	// Запускаем браузер, если не в минимизированном режиме
	if !um.config.StartMinimized {
		go func() {
			// Даем серверу время на запуск
			time.Sleep(500 * time.Millisecond)

			var err error
			um.ui, err = lorca.New(fmt.Sprintf("http://localhost:%d", um.config.WebPort), "", 800, 600)
			if err != nil {
				log.Printf("Не удалось запустить браузер: %v", err)
				// Пробуем открыть в системном браузере
				um.openInSystemBrowser(fmt.Sprintf("http://localhost:%d", um.config.WebPort))
			}
		}()
	}

	um.isRunning = true
	return nil
}

// startTrayUI запускает интерфейс в системном трее
func (um *UIManager) startTrayUI() error {
	// Для простоты используем веб-интерфейс
	return um.startWebUI()
}

// startConsoleUI запускает консольный интерфейс
func (um *UIManager) startConsoleUI() error {
	// Консольный интерфейс уже запущен в main.go
	um.isRunning = true
	return nil
}

// handleWebSocket обрабатывает WebSocket соединения
func (um *UIManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := um.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка WebSocket: %v", err)
		return
	}

	// Регистрируем нового клиента
	um.clientMutex.Lock()
	um.clients[conn] = true
	um.clientMutex.Unlock()

	// Обрабатываем сообщения от клиента
	go um.handleMessages(conn)
}

// handleMessages обрабатывает сообщения от клиента
func (um *UIManager) handleMessages(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		um.clientMutex.Lock()
		delete(um.clients, conn)
		um.clientMutex.Unlock()
	}()

	for {
		// Читаем сообщение
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Ошибка WebSocket: %v", err)
			}
			break
		}

		// Обрабатываем сообщение
		um.processMessage(message, conn)
	}
}

// processMessage обрабатывает сообщение от клиента
func (um *UIManager) processMessage(message map[string]interface{}, conn *websocket.Conn) {
	// Проверяем тип сообщения
	msgType, ok := message["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "command":
		// Получаем текст команды
		text, ok := message["text"].(string)
		if !ok {
			return
		}

		// Отправляем команду ассистенту
		response, err := um.assistant.ProcessCommand(text)
		if err != nil {
			log.Printf("Ошибка обработки команды: %v", err)
			conn.WriteJSON(map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			})
			return
		}

		// Отправляем ответ клиенту
		conn.WriteJSON(map[string]interface{}{
			"type":     "response",
			"response": response,
		})

	case "get_history":
		// Получаем историю команд
		history, err := um.assistant.GetHistory()
		if err != nil {
			log.Printf("Ошибка получения истории: %v", err)
			conn.WriteJSON(map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			})
			return
		}

		// Отправляем историю клиенту
		conn.WriteJSON(map[string]interface{}{
			"type":    "history",
			"history": history,
		})

	case "get_config":
		// Отправляем конфигурацию клиенту
		conn.WriteJSON(map[string]interface{}{
			"type":   "config",
			"config": um.config,
		})

	default:
		log.Printf("Неизвестный тип сообщения: %s", msgType)
	}
}

// extractWebFiles извлекает веб-файлы из встроенного FS
func (um *UIManager) extractWebFiles(destDir string) error {
	// Здесь должен быть код для извлечения файлов из встроенного FS
	// Для простоты создадим базовый HTML файл
	indexHTML := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KOT.AI</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 0;
            display: flex;
            flex-direction: column;
            height: 100vh;
            background-color: #f5f5f5;
        }
        .header {
            background-color: #4a86e8;
            color: white;
            padding: 10px 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
        }
        .chat-container {
            flex: 1;
            display: flex;
            flex-direction: column;
            padding: 20px;
            overflow-y: auto;
        }
        .message {
            margin-bottom: 10px;
            padding: 10px 15px;
            border-radius: 5px;
            max-width: 70%;
        }
        .user-message {
            align-self: flex-end;
            background-color: #4a86e8;
            color: white;
        }
        .bot-message {
            align-self: flex-start;
            background-color: #e9e9e9;
            color: #333;
        }
        .input-container {
            display: flex;
            padding: 10px 20px;
            background-color: white;
            border-top: 1px solid #ddd;
        }
        .input-container input {
            flex: 1;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-right: 10px;
        }
        .input-container button {
            padding: 10px 20px;
            background-color: #4a86e8;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .input-container button:hover {
            background-color: #3a76d8;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>KOT.AI</h1>
        <div>
            <button id="settings-btn">Настройки</button>
        </div>
    </div>
    <div class="chat-container" id="chat-container">
        <div class="message bot-message">
            Привет! Я KOT.AI, ваш персональный ассистент. Чем я могу помочь?
        </div>
    </div>
    <div class="input-container">
        <input type="text" id="message-input" placeholder="Введите сообщение...">
        <button id="send-btn">Отправить</button>
    </div>

    <script>
        const chatContainer = document.getElementById('chat-container');
        const messageInput = document.getElementById('message-input');
        const sendBtn = document.getElementById('send-btn');
        const settingsBtn = document.getElementById('settings-btn');
        
        // Устанавливаем WebSocket соединение
        const ws = new WebSocket('ws://' + window.location.host + '/ws');
        
        ws.onopen = function() {
            console.log('WebSocket соединение установлено');
        };
        
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            
            switch(message.type) {
                case 'response':
                    addMessage(message.response, 'bot');
                    break;
                case 'error':
                    addMessage('Ошибка: ' + message.error, 'bot');
                    break;
                case 'history':
                    // Обработка истории
                    break;
                case 'config':
                    // Обработка конфигурации
                    break;
            }
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket ошибка:', error);
        };
        
        ws.onclose = function() {
            console.log('WebSocket соединение закрыто');
            addMessage('Соединение с сервером потеряно. Пожалуйста, обновите страницу.', 'bot');
        };
        
        // Отправка сообщения
        function sendMessage() {
            const text = messageInput.value.trim();
            if (text === '') return;
            
            addMessage(text, 'user');
            
            ws.send(JSON.stringify({
                type: 'command',
                text: text
            }));
            
            messageInput.value = '';
        }
        
        // Добавление сообщения в чат
        function addMessage(text, sender) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + (sender === 'user' ? 'user-message' : 'bot-message');
            messageDiv.textContent = text;
            
            chatContainer.appendChild(messageDiv);
            chatContainer.scrollTop = chatContainer.scrollHeight;
        }
        
        // Обработчики событий
        sendBtn.addEventListener('click', sendMessage);
        
        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
        
        settingsBtn.addEventListener('click', function() {
            // Открытие настроек
            alert('Настройки пока не реализованы');
        });
    </script>
</body>
</html>`

	indexPath := filepath.Join(destDir, "index.html")
	err := os.WriteFile(indexPath, []byte(indexHTML), 0644)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// openInSystemBrowser открывает URL в системном браузере
func (um *UIManager) openInSystemBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return tracerr.New(fmt.Sprintf("Неподдерживаемая ОС: %s", runtime.GOOS))
	}

	return cmd.Start()
}