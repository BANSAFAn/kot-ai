package system

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSystemManager(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Проверяем, что SystemManager создан
	assert.NotNil(t, sm)
}

func TestGetSystemInfo(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Получаем информацию о системе
	info, err := sm.GetSystemInfo()

	// Проверяем, что информация получена без ошибок
	assert.NoError(t, err)
	// Проверяем, что поля информации не пустые
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.CPUModel)
	assert.NotEmpty(t, info.CPUCores)
	assert.NotEmpty(t, info.MemoryTotal)
	assert.NotEmpty(t, info.MemoryFree)
	assert.NotEmpty(t, info.DiskTotal)
	assert.NotEmpty(t, info.DiskFree)
}

func TestListProcesses(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Получаем список процессов
	processes, err := sm.ListProcesses()

	// Проверяем, что список получен без ошибок
	assert.NoError(t, err)
	// Проверяем, что список не пустой
	assert.NotEmpty(t, processes)

	// Проверяем, что у каждого процесса есть PID и имя
	for _, proc := range processes {
		assert.NotZero(t, proc.PID)
		assert.NotEmpty(t, proc.Name)
	}
}

func TestRunCommand(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Запускаем простую команду (echo для Windows)
	output, err := sm.RunCommand("cmd", "/c", "echo", "test")

	// Проверяем, что команда выполнена без ошибок
	assert.NoError(t, err)
	// Проверяем, что вывод содержит ожидаемый текст
	assert.Contains(t, output, "test")
}

func TestGetVolume(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Получаем текущую громкость
	volume, err := sm.GetVolume()

	// Проверяем, что громкость получена без ошибок
	assert.NoError(t, err)
	// Проверяем, что громкость в допустимом диапазоне (0-100)
	assert.GreaterOrEqual(t, volume, 0)
	assert.LessOrEqual(t, volume, 100)
}

func TestIsMuted(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Получаем статус отключения звука
	muted, err := sm.IsMuted()

	// Проверяем, что статус получен без ошибок
	assert.NoError(t, err)
	// Проверяем, что статус имеет булево значение
	assert.IsType(t, bool(false), muted)
}

func TestTakeScreenshot(t *testing.T) {
	// Создаем новый экземпляр SystemManager
	sm := NewSystemManager()

	// Создаем временный файл для скриншота
	tempFile, err := os.CreateTemp("", "screenshot-*.png")
	assert.NoError(t, err)
	tempFile.Close()
	screenshotPath := tempFile.Name()
	defer os.Remove(screenshotPath) // Удаляем файл после теста

	// Делаем скриншот
	err = sm.TakeScreenshot(screenshotPath)

	// Проверяем, что скриншот сделан без ошибок
	assert.NoError(t, err)

	// Проверяем, что файл скриншота существует и не пустой
	fileInfo, err := os.Stat(screenshotPath)
	assert.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}