package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/ztrue/tracerr"
)

// SystemManager управляет системными операциями
type SystemManager struct {
	initializedOLE bool
}

// NewSystemManager создает новый экземпляр SystemManager
func NewSystemManager() *SystemManager {
	return &SystemManager{}
}

// RunCommand запускает команду в системе
func (sm *SystemManager) RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), tracerr.Wrap(err)
	}
	return string(output), nil
}

// OpenApplication открывает приложение
func (sm *SystemManager) OpenApplication(appPath string) error {
	// Проверяем, существует ли файл
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		// Пробуем найти приложение в PATH
		pathEnv := os.Getenv("PATH")
		paths := strings.Split(pathEnv, ";")

		found := false
		for _, path := range paths {
			possiblePath := filepath.Join(path, appPath)
			if _, err := os.Stat(possiblePath); !os.IsNotExist(err) {
				appPath = possiblePath
				found = true
				break
			}

			// Проверяем с расширением .exe
			if !strings.HasSuffix(appPath, ".exe") {
				possiblePath = filepath.Join(path, appPath+".exe")
				if _, err := os.Stat(possiblePath); !os.IsNotExist(err) {
					appPath = possiblePath
					found = true
					break
				}
			}
		}

		if !found {
			return tracerr.New(fmt.Sprintf("Приложение %s не найдено", appPath))
		}
	}

	cmd := exec.Command(appPath)
	err := cmd.Start()
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// OpenURL открывает URL в браузере по умолчанию
func (sm *SystemManager) OpenURL(url string) error {
	var cmd *exec.Cmd

	cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	err := cmd.Start()
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// GetSystemInfo возвращает информацию о системе
func (sm *SystemManager) GetSystemInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Информация о хосте
	hostInfo, err := host.Info()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	info["hostname"] = hostInfo.Hostname
	info["os"] = hostInfo.OS
	info["platform"] = hostInfo.Platform
	info["platform_version"] = hostInfo.PlatformVersion
	info["uptime"] = hostInfo.Uptime

	// Информация о CPU
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	if len(cpuInfo) > 0 {
		info["cpu_model"] = cpuInfo[0].ModelName
		info["cpu_cores"] = cpuInfo[0].Cores
	}

	// Информация о памяти
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	info["total_memory"] = memInfo.Total
	info["free_memory"] = memInfo.Free
	info["used_memory"] = memInfo.Used
	info["memory_percent"] = memInfo.UsedPercent

	// Информация о дисках
	diskInfo, err := disk.Partitions(false)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	disks := make([]map[string]interface{}, 0, len(diskInfo))
	for _, partition := range diskInfo {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		diskMap := make(map[string]interface{})
		diskMap["device"] = partition.Device
		diskMap["mountpoint"] = partition.Mountpoint
		diskMap["fstype"] = partition.Fstype
		diskMap["total"] = usage.Total
		diskMap["free"] = usage.Free
		diskMap["used"] = usage.Used
		diskMap["percent"] = usage.UsedPercent

		disks = append(disks, diskMap)
	}
	info["disks"] = disks

	return info, nil
}

// GetRunningProcesses возвращает список запущенных процессов
func (sm *SystemManager) GetRunningProcesses() ([]map[string]interface{}, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	result := make([]map[string]interface{}, 0, len(processes))
	for _, p := range processes {
		processInfo, err := sm.getProcessInfo(p)
		if err != nil {
			continue // Пропускаем процессы, для которых не удалось получить информацию
		}
		result = append(result, processInfo)
	}

	return result, nil
}

func (sm *SystemManager) getProcessInfo(p *process.Process) (map[string]interface{}, error) {
	name, err := p.Name()
	if err != nil {
		return nil, err
	}

	createTime, err := p.CreateTime()
	if err != nil {
		return nil, err
	}

	memPercent, err := p.MemoryPercent()
	if err != nil {
		return nil, err
	}

	cpuPercent, err := p.CPUPercent()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pid":         p.Pid,
		"name":        name,
		"create_time": time.Unix(0, createTime*int64(time.Millisecond)),
		"mem_percent": memPercent,
		"cpu_percent": cpuPercent,
	}, nil
}

// KillProcess завершает процесс по PID
func (sm *SystemManager) KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return p.Kill()
}

// SetVolume устанавливает громкость системы (0-100)
func (sm *SystemManager) SetVolume(level int) error {
	if level < 0 {
		level = 0
	} else if level > 100 {
		level = 100
	}

	return sm.setMasterVolume(float32(level) / 100.0)
}

// GetVolume возвращает текущий уровень громкости (0-100)
func (sm *SystemManager) GetVolume() int {
	volume, err := sm.getMasterVolume()
	if err != nil {
		return 0 // Возвращаем 0 в случае ошибки
	}
	return int(volume * 100)
}

// ToggleMute включает/выключает звук
func (sm *SystemManager) ToggleMute() error {
	return sm.toggleMasterMute()
}

// --- Вспомогательные функции для управления звуком --- 

func (sm *SystemManager) getAudioEndpointVolume() (*wca.IAudioEndpointVolume, error) {
	if !sm.initializedOLE {
		if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
			return nil, err
		}
		sm.initializedOLE = true
	}

	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return nil, err
	}
	defer mmde.Release()

	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		return nil, err
	}
	defer mmd.Release()

	var aev *wca.IAudioEndpointVolume
	if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return nil, err
	}

	return aev, nil
}

func (sm *SystemManager) setMasterVolume(level float32) error {
	aev, err := sm.getAudioEndpointVolume()
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer aev.Release()

	return tracerr.Wrap(aev.SetMasterVolumeLevelScalar(level, nil))
}

func (sm *SystemManager) getMasterVolume() (float32, error) {
	aev, err := sm.getAudioEndpointVolume()
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	defer aev.Release()

	var level float32
	if err := aev.GetMasterVolumeLevelScalar(&level); err != nil {
		return 0, tracerr.Wrap(err)
	}
	return level, nil
}

func (sm *SystemManager) toggleMasterMute() error {
	aev, err := sm.getAudioEndpointVolume()
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer aev.Release()

	var muted bool
	if err := aev.GetMute(&muted); err != nil {
		return tracerr.Wrap(err)
	}

	return tracerr.Wrap(aev.SetMute(!muted, nil))
}

// TakeScreenshot делает снимок экрана и сохраняет его в указанный файл
func (sm *SystemManager) TakeScreenshot(filePath string) error {


	// Создаем команду для использования утилиты screencapture
	cmd := exec.Command("powershell", "-command", fmt.Sprintf(
		"Add-Type -AssemblyName System.Windows.Forms; "+
		"[System.Windows.Forms.SendKeys]::SendWait('{PRTSC}'); "+
		"$img = [System.Windows.Forms.Clipboard]::GetImage(); "+
		"$img.Save('%s');", filePath))

	err := cmd.Run()
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// Cleanup освобождает ресурсы
func (sm *SystemManager) Cleanup() {
	if sm.initializedOLE {
		ole.CoUninitialize()
		sm.initializedOLE = false
	}
}

// PrintStatus выводит информацию о статусе приложения в консоль
func (sm *SystemManager) PrintStatus() {
	status := sm.CheckStatus()
	fmt.Println(status.GetStatusSummary())
}