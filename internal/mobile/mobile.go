package mobile

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ztrue/tracerr"
)

// MobileManager управляет подключением и взаимодействием с мобильными устройствами
type MobileManager struct {
	config       MobileConfig
	isRunning    bool
	adbPath      string
	connectedUSB bool
	deviceID     string
	mutex        sync.Mutex
}

// MobileConfig содержит настройки для мобильного подключения
type MobileConfig struct {
	Enabled       bool   `json:"enabled"`       // Включено ли мобильное подключение
	USBEnabled    bool   `json:"usb_enabled"`    // Включено ли USB-подключение
	ADBPath       string `json:"adb_path"`       // Путь к ADB (Android Debug Bridge)
	WebUIEnabled  bool   `json:"webui_enabled"`  // Включен ли веб-интерфейс для мобильных устройств
	WebUIPort     int    `json:"webui_port"`     // Порт для веб-интерфейса мобильных устройств
	AutoConnect   bool   `json:"auto_connect"`   // Автоматически подключаться к устройству при запуске
}

// NewMobileManager создает новый экземпляр MobileManager
func NewMobileManager(config MobileConfig) *MobileManager {
	return &MobileManager{
		config:       config,
		isRunning:    false,
		adbPath:      config.ADBPath,
		connectedUSB: false,
		mutex:        sync.Mutex{},
	}
}

// Start запускает менеджер мобильных устройств
func (mm *MobileManager) Start() error {
	if !mm.config.Enabled {
		log.Println("Мобильное подключение отключено в настройках")
		return nil
	}

	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Проверяем и инициализируем ADB, если USB-подключение включено
	if mm.config.USBEnabled {
		if err := mm.initADB(); err != nil {
			log.Printf("Предупреждение: не удалось инициализировать ADB: %v", err)
		}

		// Если включено автоподключение, пытаемся подключиться к устройству
		if mm.config.AutoConnect {
			go mm.autoConnectUSB()
		}
	}

	mm.isRunning = true
	return nil
}

// Stop останавливает менеджер мобильных устройств
func (mm *MobileManager) Stop() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.config.USBEnabled && mm.connectedUSB {
		// Отключаем устройство
		mm.disconnectUSB()
	}

	mm.isRunning = false
}

// initADB инициализирует ADB и проверяет его доступность
func (mm *MobileManager) initADB() error {
	// Если путь к ADB не указан, пытаемся найти его
	if mm.adbPath == "" {
		adbPath, err := mm.findADB()
		if err != nil {
			return tracerr.Wrap(err)
		}
		mm.adbPath = adbPath
	}

	// Проверяем, что ADB работает
	cmd := exec.Command(mm.adbPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return tracerr.Wrap(fmt.Errorf("ошибка при проверке ADB: %v, вывод: %s", err, output))
	}

	log.Printf("ADB инициализирован: %s", strings.TrimSpace(string(output)))
	return nil
}

// findADB пытается найти ADB в системе
func (mm *MobileManager) findADB() (string, error) {
	// Стандартные пути для разных ОС
	var possiblePaths []string

	switch runtime.GOOS {
	case "windows":
		// Стандартные пути для Windows
		possiblePaths = []string{
			"adb.exe",
			"C:\\Program Files\\Android\\android-sdk\\platform-tools\\adb.exe",
			"C:\\Program Files (x86)\\Android\\android-sdk\\platform-tools\\adb.exe",
			"C:\\Android\\android-sdk\\platform-tools\\adb.exe",
			"C:\\Android\\sdk\\platform-tools\\adb.exe",
		}
	case "darwin":
		// Стандартные пути для macOS
		possiblePaths = []string{
			"adb",
			"/usr/local/bin/adb",
			"/usr/bin/adb",
			"/opt/homebrew/bin/adb",
			"/Users/" + mm.getCurrentUser() + "/Library/Android/sdk/platform-tools/adb",
		}
	case "linux":
		// Стандартные пути для Linux
		possiblePaths = []string{
			"adb",
			"/usr/local/bin/adb",
			"/usr/bin/adb",
			"/home/" + mm.getCurrentUser() + "/Android/Sdk/platform-tools/adb",
		}
	}

	// Проверяем каждый путь
	for _, path := range possiblePaths {
		cmd := exec.Command(path, "version")
		if err := cmd.Run(); err == nil {
			return path, nil
		}
	}

	return "", tracerr.New("ADB не найден в системе")
}

// getCurrentUser возвращает имя текущего пользователя
func (mm *MobileManager) getCurrentUser() string {
	// Для простоты используем переменную окружения
	var username string
	switch runtime.GOOS {
	case "windows":
		username = mm.runCommand("whoami")
		// Извлекаем только имя пользователя из формата DOMAIN\USER
		parts := strings.Split(username, "\\")
		if len(parts) > 1 {
			username = parts[1]
		}
	case "darwin", "linux":
		username = mm.runCommand("whoami")
	}
	return strings.TrimSpace(username)
}

// runCommand запускает команду и возвращает её вывод
func (mm *MobileManager) runCommand(command string, args ...string) string {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Ошибка при выполнении команды %s: %v", command, err)
		return ""
	}
	return string(output)
}

// autoConnectUSB пытается автоматически подключиться к USB-устройству
func (mm *MobileManager) autoConnectUSB() {
	// Ждем некоторое время, чтобы система успела инициализироваться
	time.Sleep(2 * time.Second)

	// Пытаемся найти подключенные устройства
	devices, err := mm.getConnectedDevices()
	if err != nil {
		log.Printf("Ошибка при поиске USB-устройств: %v", err)
		return
	}

	if len(devices) == 0 {
		log.Println("Не найдено подключенных USB-устройств")
		return
	}

	// Подключаемся к первому найденному устройству
	deviceID := devices[0]
	if err := mm.connectUSB(deviceID); err != nil {
		log.Printf("Ошибка при подключении к USB-устройству %s: %v", deviceID, err)
		return
	}

	log.Printf("Успешно подключено к USB-устройству: %s", deviceID)
}

// getConnectedDevices возвращает список подключенных USB-устройств
func (mm *MobileManager) getConnectedDevices() ([]string, error) {
	if mm.adbPath == "" {
		return nil, tracerr.New("ADB не инициализирован")
	}

	cmd := exec.Command(mm.adbPath, "devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	// Парсим вывод ADB devices
	lines := strings.Split(string(output), "\n")
	var devices []string

	// Пропускаем первую строку (заголовок)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Формат строки: "deviceID    device"
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			devices = append(devices, parts[0])
		}
	}

	return devices, nil
}

// connectUSB подключается к указанному USB-устройству
func (mm *MobileManager) connectUSB(deviceID string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.adbPath == "" {
		return tracerr.New("ADB не инициализирован")
	}

	// Проверяем, что устройство доступно
	devices, err := mm.getConnectedDevices()
	if err != nil {
		return tracerr.Wrap(err)
	}

	found := false
	for _, device := range devices {
		if device == deviceID {
			found = true
			break
		}
	}

	if !found {
		return tracerr.New(fmt.Sprintf("Устройство %s не найдено", deviceID))
	}

	// Устанавливаем устройство как активное
	mm.deviceID = deviceID
	mm.connectedUSB = true

	return nil
}

// disconnectUSB отключает текущее USB-устройство
func (mm *MobileManager) disconnectUSB() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if !mm.connectedUSB {
		return
	}

	// Сбрасываем состояние подключения
	mm.deviceID = ""
	mm.connectedUSB = false
}

// IsConnectedUSB проверяет, подключено ли USB-устройство
func (mm *MobileManager) IsConnectedUSB() bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	return mm.connectedUSB
}

// GetConnectedDeviceID возвращает ID подключенного устройства
func (mm *MobileManager) GetConnectedDeviceID() string {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	return mm.deviceID
}

// ExecuteCommand выполняет команду на подключенном устройстве
func (mm *MobileManager) ExecuteCommand(command string, args ...string) (string, error) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if !mm.connectedUSB || mm.deviceID == "" {
		return "", tracerr.New("Нет подключенного USB-устройства")
	}

	if mm.adbPath == "" {
		return "", tracerr.New("ADB не инициализирован")
	}

	// Формируем команду для ADB
	adbArgs := append([]string{"-s", mm.deviceID, "shell", command}, args...)
	cmd := exec.Command(mm.adbPath, adbArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), tracerr.Wrap(err)
	}

	return string(output), nil
}

// PushFile отправляет файл на устройство
func (mm *MobileManager) PushFile(localPath, remotePath string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if !mm.connectedUSB || mm.deviceID == "" {
		return tracerr.New("Нет подключенного USB-устройства")
	}

	if mm.adbPath == "" {
		return tracerr.New("ADB не инициализирован")
	}

	// Проверяем, существует ли локальный файл
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return tracerr.Wrap(fmt.Errorf("локальный файл не существует: %s", localPath))
	}

	// Отправляем файл на устройство
	cmd := exec.Command(mm.adbPath, "-s", mm.deviceID, "push", localPath, remotePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return tracerr.Wrap(fmt.Errorf("ошибка при отправке файла: %v, вывод: %s", err, output))
	}

	return nil
}

// PullFile получает файл с устройства
func (mm *MobileManager) PullFile(remotePath, localPath string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if !mm.connectedUSB || mm.deviceID == "" {
		return tracerr.New("Нет подключенного USB-устройства")
	}

	if mm.adbPath == "" {
		return tracerr.New("ADB не инициализирован")
	}

	// Создаем директорию для локального файла, если она не существует
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return tracerr.Wrap(err)
	}

	// Получаем файл с устройства
	cmd := exec.Command(mm.adbPath, "-s", mm.deviceID, "pull", remotePath, localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return tracerr.Wrap(fmt.Errorf("ошибка при получении файла: %v, вывод: %s", err, output))
	}

	return nil
}

// TakeScreenshot делает снимок экрана устройства
func (mm *MobileManager) TakeScreenshot(localPath string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if !mm.connectedUSB || mm.deviceID == "" {
		return tracerr.New("Нет подключенного USB-устройства")
	}

	if mm.adbPath == "" {
		return tracerr.New("ADB не инициализирован")
	}

	// Временный путь на устройстве
	remotePath := "/sdcard/screenshot.png"

	// Делаем снимок экрана
	cmd := exec.Command(mm.adbPath, "-s", mm.deviceID, "shell", "screencap", "-p", remotePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return tracerr.Wrap(fmt.Errorf("ошибка при создании снимка экрана: %v, вывод: %s", err, output))
	}

	// Получаем файл с устройства
	if err := mm.PullFile(remotePath, localPath); err != nil {
		return tracerr.Wrap(err)
	}

	// Удаляем временный файл на устройстве
	cmd = exec.Command(mm.adbPath, "-s", mm.deviceID, "shell", "rm", remotePath)
	if err := cmd.Run(); err != nil {
		log.Printf("Предупреждение: не удалось удалить временный файл на устройстве: %v", err)
	}

	return nil
}