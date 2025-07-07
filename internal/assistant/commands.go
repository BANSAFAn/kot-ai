package assistant

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baneronetwo/kot-ai/internal/bank"
	"github.com/baneronetwo/kot-ai/internal/drawing"
	"github.com/baneronetwo/kot-ai/internal/steam"
)

// CommandHandler определяет функцию-обработчик для команды
type CommandHandler func(a *Assistant, args []string) (string, bool)

// Command представляет специальную команду
type Command struct {
	Keywords []string
	Handler  CommandHandler
}

// commandRegistry содержит все зарегистрированные специальные команды
var commandRegistry = []Command{
	{
		Keywords: []string{"открой", "запусти"},
		Handler:  handleOpenApplication,
	},
	{
		Keywords: []string{"увеличь громкость", "громче"},
		Handler:  handleIncreaseVolume,
	},
	{
		Keywords: []string{"уменьши громкость", "тише"},
		Handler:  handleDecreaseVolume,
	},
	{
		Keywords: []string{"выключи звук", "без звука"},
		Handler:  handleMuteVolume,
	},
	{
		Keywords: []string{"скриншот", "снимок экрана"},
		Handler:  handleScreenshot,
	},
	{
		Keywords: []string{"информация о системе", "системная информация"},
		Handler:  handleSystemInfo,
	},
	{
		Keywords: []string{"список процессов", "запущенные программы"},
		Handler:  handleProcessList,
	},
	{
		Keywords: []string{"завершить процесс", "убить процесс"},
		Handler:  handleKillProcess,
	},
	{
		Keywords: []string{"открой сайт", "открой страницу"},
		Handler:  handleOpenURL,
	},
	{
		Keywords: []string{"очистить историю", "удалить историю"},
		Handler:  handleClearHistory,
	},
	{
		Keywords: []string{"нарисуй"},
		Handler:  handleDraw,
	},
	{
		Keywords: []string{"мои игры"},
		Handler:  handleSteamGames,
	},
	{
		Keywords: []string{"мой баланс"},
		Handler:  handleBankBalance,
	},
	{
		Keywords: []string{"отправь деньги"},
		Handler:  handleSendMoney,
	},
	{
		Keywords: []string{"выход", "закрыть", "завершить работу"},
		Handler:  handleExit,
	},
}

func handleOpenApplication(a *Assistant, args []string) (string, bool) {
	if len(args) == 0 {
		return "Пожалуйста, укажите, что открыть", true
	}
	appName := strings.Join(args, " ")
	err := a.system.OpenApplication(appName)
	if err != nil {
		return fmt.Sprintf("Не удалось открыть %s: %v", appName, err), true
	}
	return fmt.Sprintf("Открываю %s", appName), true
}

func handleIncreaseVolume(a *Assistant, args []string) (string, bool) {
	a.system.SetVolume(a.system.GetVolume() + 10)
	return "Громкость увеличена", true
}

func handleDecreaseVolume(a *Assistant, args []string) (string, bool) {
	a.system.SetVolume(a.system.GetVolume() - 10)
	return "Громкость уменьшена", true
}

func handleMuteVolume(a *Assistant, args []string) (string, bool) {
	a.system.SetVolume(0)
	return "Выключаю звук", true
}

func handleScreenshot(a *Assistant, args []string) (string, bool) {
	homeDir, _ := os.UserHomeDir()
	screenshotPath := filepath.Join(homeDir, "Pictures", fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))
	err := a.system.TakeScreenshot(screenshotPath)
	if err != nil {
		return fmt.Sprintf("Не удалось сделать скриншот: %v", err), true
	}
	return fmt.Sprintf("Скриншот сохранен в %s", screenshotPath), true
}

func handleSystemInfo(a *Assistant, args []string) (string, bool) {
	info, err := a.system.GetSystemInfo()
	if err != nil {
		return fmt.Sprintf("Не удалось получить информацию о системе: %v", err), true
	}
	response := "Информация о системе:\n"
	response += fmt.Sprintf("Хост: %s\n", info["hostname"])
	response += fmt.Sprintf("ОС: %s %s\n", info["platform"], info["platform_version"])
	response += fmt.Sprintf("Процессор: %s (%d ядер)\n", info["cpu_model"], info["cpu_cores"])
	response += fmt.Sprintf("Память: %.2f ГБ / %.2f ГБ (%.1f%%)\n",
		float64(info["used_memory"].(uint64))/(1024*1024*1024),
		float64(info["total_memory"].(uint64))/(1024*1024*1024),
		info["memory_percent"].(float64))
	return response, true
}

func handleProcessList(a *Assistant, args []string) (string, bool) {
	processes, err := a.system.GetRunningProcesses()
	if err != nil {
		return fmt.Sprintf("Не удалось получить список процессов: %v", err), true
	}
	response := "Топ процессов по использованию памяти:\n"
	for i, p := range processes {
		if i >= 10 {
			break
		}
		response += fmt.Sprintf("%s (PID: %d) - %.1f%% памяти, %.1f%% CPU\n",
			p["name"].(string),
			p["pid"].(int32),
			p["mem_percent"].(float32),
			p["cpu_percent"].(float64))
	}
	return response, true
}

func handleKillProcess(a *Assistant, args []string) (string, bool) {
	if len(args) == 0 {
		return "Пожалуйста, укажите имя процесса для завершения", true
	}
	processName := strings.Join(args, " ")
	processes, err := a.system.GetRunningProcesses()
	if err != nil {
		return fmt.Sprintf("Не удалось получить список процессов: %v", err), true
	}
	var foundPID int32
	found := false
	for _, p := range processes {
		if strings.Contains(strings.ToLower(p["name"].(string)), strings.ToLower(processName)) {
			foundPID = p["pid"].(int32)
			found = true
			break
		}
	}
	if !found {
		return fmt.Sprintf("Процесс %s не найден", processName), true
	}
	err = a.system.KillProcess(foundPID)
	if err != nil {
		return fmt.Sprintf("Не удалось завершить процесс %s: %v", processName, err), true
	}
	return fmt.Sprintf("Процесс %s успешно завершен", processName), true
}

func handleOpenURL(a *Assistant, args []string) (string, bool) {
	if len(args) == 0 {
		return "Пожалуйста, укажите URL для открытия", true
	}
	url := args[len(args)-1]
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	err := a.system.OpenURL(url)
	if err != nil {
		return fmt.Sprintf("Не удалось открыть URL %s: %v", url, err), true
	}
	return fmt.Sprintf("Открываю %s", url), true
}

func handleClearHistory(a *Assistant, args []string) (string, bool) {
	if a.db != nil {
		a.db.Close()
		os.RemoveAll(a.config.HistoryFilePath)
		db, err := leveldb.OpenFile(a.config.HistoryFilePath, nil)
		if err != nil {
			return fmt.Sprintf("Не удалось очистить историю: %v", err), true
		}
		a.db = db
		return "История успешно очищена", true
	}
	return "История отключена в настройках", true
}

func handleDraw(a *Assistant, args []string) (string, bool) {
	homeDir, _ := os.UserHomeDir()
	drawingPath := filepath.Join(homeDir, "Pictures", fmt.Sprintf("drawing_%d.png", time.Now().Unix()))
	err := drawing.Draw(drawingPath)
	if err != nil {
		return fmt.Sprintf("Не удалось нарисовать: %v", err), true
	}
	return fmt.Sprintf("Рисунок сохранен в %s", drawingPath), true
}

func handleSteamGames(a *Assistant, args []string) (string, bool) {
	games, err := steam.GetGames()
	if err != nil {
		return fmt.Sprintf("Не удалось получить список игр: %v", err), true
	}
	response := "Ваши игры в Steam:\n"
	for _, game := range games {
		response += fmt.Sprintf("- %s\n", game)
	}
	return response, true
}

func handleBankBalance(a *Assistant, args []string) (string, bool) {
	balance, err := bank.GetBalance()
	if err != nil {
		return fmt.Sprintf("Не удалось получить баланс: %v", err), true
	}
	return fmt.Sprintf("Ваш баланс: %.2f", balance), true
}

func handleSendMoney(a *Assistant, args []string) (string, bool) {
	return "Функция отправки денег еще не реализована", true
}

func handleExit(a *Assistant, args []string) (string, bool) {
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	return "Завершаю работу. До свидания!", true
}