package serverStats

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

const trafficSaveInterval = 5 * time.Minute

var startTime time.Time
var currentTraffic trafficInfo

type trafficInfo struct {
	Total      uint64
	Mutex      sync.RWMutex
	LastUpdate time.Time
}

func Init() {
	startTime = time.Now()
	currentTraffic = trafficInfo{LastUpdate: startTime}
	AddTraffic(database.GetStatTraffic())
	monitorCpuUsage()
}

func monitorCpuUsage() {
	go func() {
		_ = GetCpuUsage()
		select {
		// run every minute, as GetCpuUsage only reports the
		// percentage since the last call
		case <-time.After(time.Minute * 1):
			monitorCpuUsage()
		}
	}()
}

func Shutdown() {
	saveTraffic()
}

func saveTraffic() {
	database.SaveStatTraffic(GetCurrentTraffic())
}

func GetUptime() int64 {
	return time.Since(startTime).Milliseconds() / 1000
}

func GetTotalFiles() int {
	return len(database.GetAllMetadata())
}

func GetCurrentTraffic() uint64 {
	currentTraffic.Mutex.RLock()
	defer currentTraffic.Mutex.RUnlock()
	return currentTraffic.Total
}

func GetMemoryInfo() (uint64, uint64, uint64) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println(err)
		return 0, 0, 0
	}
	return memInfo.Free, memInfo.Used, memInfo.Total
}

func GetDiskInfo() (uint64, uint64, uint64) {
	diskInfo, err := disk.Usage(configuration.Get().DataDir)
	if err != nil {
		fmt.Println(err)
		return 0, 0, 0
	}
	return diskInfo.Free, diskInfo.Used, diskInfo.Total
}

func GetCpuUsage() int {
	usage, err := cpu.Percent(0, false)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return int(math.Round(usage[0]))
}

func AddTraffic(bytes uint64) {
	currentTraffic.Mutex.Lock()
	currentTraffic.Total = currentTraffic.Total + bytes
	requireSave := time.Since(currentTraffic.LastUpdate) > trafficSaveInterval
	currentTraffic.LastUpdate = time.Now()
	currentTraffic.Mutex.Unlock()

	if requireSave {
		saveTraffic()
	}
}
