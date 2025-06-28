package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// StatusInfo represents the status information of the application
type StatusInfo struct {
	ExecutableExists   bool
	ConfigDirExists    bool
	ConfigFileExists   bool
	HistoryDBExists    bool
	IsRunning          bool
	GoInstalled        bool
	GoVersion          string
	LastChecked        time.Time
	StatusMessages     []StatusMessage
}

// StatusMessage represents a status message with a level
type StatusMessage struct {
	Level   string // "OK", "INFO", "WARNING", "ERROR"
	Message string
}

// CheckStatus checks the status of the application and returns a StatusInfo
func (sm *SystemManager) CheckStatus() StatusInfo {
	info := StatusInfo{
		LastChecked:    time.Now(),
		StatusMessages: []StatusMessage{},
	}

	// Get executable path
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		exeName := filepath.Base(exePath)
		info.ExecutableExists = true
		info.StatusMessages = append(info.StatusMessages, StatusMessage{
			Level:   "OK",
			Message: fmt.Sprintf("Executable found: %s", exeName),
		})
	} else {
		info.StatusMessages = append(info.StatusMessages, StatusMessage{
			Level:   "ERROR",
			Message: "Could not determine executable path",
		})
	}

	// Check if application is running (this is always true when called from the app)
	info.IsRunning = true
	info.StatusMessages = append(info.StatusMessages, StatusMessage{
		Level:   "OK",
		Message: "Application is running",
	})

	// Check configuration directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".kot.ai")
		if _, err := os.Stat(configDir); err == nil {
			info.ConfigDirExists = true
			info.StatusMessages = append(info.StatusMessages, StatusMessage{
				Level:   "OK",
				Message: "Configuration directory exists",
			})

			// Check configuration file
			configFile := filepath.Join(configDir, "config.json")
			if _, err := os.Stat(configFile); err == nil {
				info.ConfigFileExists = true
				info.StatusMessages = append(info.StatusMessages, StatusMessage{
					Level:   "OK",
					Message: "Configuration file exists",
				})
			} else {
				info.StatusMessages = append(info.StatusMessages, StatusMessage{
					Level:   "WARNING",
					Message: "Configuration file not found",
				})
			}

			// Check history database
			historyDB := filepath.Join(configDir, "history.db")
			if _, err := os.Stat(historyDB); err == nil {
				info.HistoryDBExists = true
				info.StatusMessages = append(info.StatusMessages, StatusMessage{
					Level:   "OK",
					Message: "History database exists",
				})
			} else {
				info.StatusMessages = append(info.StatusMessages, StatusMessage{
					Level:   "INFO",
					Message: "History database not found",
				})
			}
		} else {
			info.StatusMessages = append(info.StatusMessages, StatusMessage{
				Level:   "WARNING",
				Message: "Configuration directory not found",
			})
		}
	} else {
		info.StatusMessages = append(info.StatusMessages, StatusMessage{
			Level:   "ERROR",
			Message: "Could not determine user home directory",
		})
	}

	// Check Go version (only relevant for development)
	info.GoVersion = runtime.Version()
	if info.GoVersion != "" {
		info.GoInstalled = true
		info.StatusMessages = append(info.StatusMessages, StatusMessage{
			Level:   "OK",
			Message: fmt.Sprintf("Go is installed (Version: %s)", info.GoVersion),
		})
	}

	return info
}

// GetStatusSummary returns a string summary of the application status
func (info StatusInfo) GetStatusSummary() string {
	var sb strings.Builder

	sb.WriteString("KOT.AI Status Summary:\n")
	sb.WriteString(fmt.Sprintf("Last checked: %s\n\n", info.LastChecked.Format(time.RFC1123)))

	// Count status levels
	okCount := 0
	warningCount := 0
	errorCount := 0

	for _, msg := range info.StatusMessages {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", msg.Level, msg.Message))

		switch msg.Level {
		case "OK":
			okCount++
		case "WARNING":
			warningCount++
		case "ERROR":
			errorCount++
		}
	}

	sb.WriteString("\nSummary:\n")
	sb.WriteString(fmt.Sprintf("- %d OK\n", okCount))
	sb.WriteString(fmt.Sprintf("- %d Warnings\n", warningCount))
	sb.WriteString(fmt.Sprintf("- %d Errors\n", errorCount))

	if errorCount > 0 {
		sb.WriteString("\nStatus: CRITICAL - Application may not function correctly\n")
	} else if warningCount > 0 {
		sb.WriteString("\nStatus: WARNING - Application may have limited functionality\n")
	} else {
		sb.WriteString("\nStatus: HEALTHY - All systems operational\n")
	}

	return sb.String()
}