package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"flag"

	"kot.ai/internal/assistant"
	"kot.ai/internal/config"
	"kot.ai/internal/mobile"
	"kot.ai/internal/ui"
	"kot.ai/internal/voice"
	"kot.ai/internal/system"
)

func main() {
	// Parse command line arguments
	checkStatus := flag.Bool("status", false, "Check application status")
	flag.Parse()

	// Initialize system manager for both normal operation and status check
	sys := system.NewSystemManager()

	// If status check is requested, print status and exit
	if *checkStatus {
		sys.PrintStatus()
		return
	}

	// Настройка логирования
	logFile, err := os.OpenFile("kot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Ошибка при открытии файла лога: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("Запуск KOT.AI...")

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	// Инициализация компонентов
	voiceManager := voice.NewVoiceManager(cfg.VoiceConfig)
	mobileManager := mobile.NewMobileManager(cfg.MobileConfig)
	assistant := assistant.NewAssistant(cfg.AssistantConfig, sys, voiceManager)
	uiManager := ui.NewUIManager(cfg.UIConfig, assistant, mobileManager)

	// Запуск компонентов
	if err := voiceManager.Start(); err != nil {
		log.Printf("Предупреждение: не удалось запустить голосовой модуль: %v", err)
	}

	if err := mobileManager.Start(); err != nil {
		log.Printf("Предупреждение: не удалось запустить модуль мобильного управления: %v", err)
	}

	if err := uiManager.Start(); err != nil {
		log.Fatalf("Ошибка при запуске UI: %v", err)
	}

	if err := assistant.Start(); err != nil {
		log.Fatalf("Ошибка при запуске ассистента: %v", err)
	}

	fmt.Println("KOT.AI запущен и готов к работе!")
	fmt.Println("Нажмите Ctrl+C для завершения работы.")

	// Ожидание сигнала завершения
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Корректное завершение работы
	fmt.Println("\nЗавершение работы KOT.AI...")
	assistant.Stop()
	uiManager.Stop()
	voiceManager.Stop()
	mobileManager.Stop()
	log.Println("KOT.AI успешно завершил работу.")
}