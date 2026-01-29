package serverstats

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
var lastCpuCheck time.Time
var currentTraffic trafficInfo

type trafficInfo struct {
	Total          uint64
	Mutex          sync.RWMutex
	LastUpdate     time.Time
	RecordingSince int64
}

// Init initializes the server stats
func Init() {
	startTime = time.Now()
	currentTraffic = trafficInfo{LastUpdate: startTime, RecordingSince: getInitTrafficSince()}
	AddTraffic(database.GetStatTraffic())
	monitorCpuUsage()
}

func getInitTrafficSince() int64 {
	since, ok := database.GetTrafficSince()
	if !ok {
		since = time.Now().Unix()
		database.SaveTrafficSince(since)
	}
	return since
}

func monitorCpuUsage() {
	// continuously run, as GetCpuUsage only reports the
	// percentage since the last call
	go func() {
		if time.Since(lastCpuCheck) > 2*time.Minute {
			_ = GetCpuUsage()
			lastCpuCheck = time.Now()
		}
		select {
		case <-time.After(time.Minute * 1):
			monitorCpuUsage()
		}
	}()
}

// Shutdown saves statistics to the database
func Shutdown() {
	saveTraffic()
}

func saveTraffic() {
	totalTraffic, _ := GetCurrentTraffic()
	database.SaveStatTraffic(totalTraffic)
}

// ClearTraffic resets the traffic counter
func ClearTraffic() {
	timeNow := time.Now().Unix()
	currentTraffic = trafficInfo{LastUpdate: time.Now(), RecordingSince: timeNow}
	database.SaveStatTraffic(0)
	database.SaveTrafficSince(timeNow)
}

// GetUptime returns the uptime of the server in seconds
func GetUptime() int64 {
	return time.Since(startTime).Milliseconds() / 1000
}

// GetTotalFiles returns the total number of files stored in the database
func GetTotalFiles() int {
	return len(database.GetAllMetadata())
}

// GetCurrentTraffic returns the current traffic in bytes and the time since the last recording
func GetCurrentTraffic() (uint64, int64) {
	currentTraffic.Mutex.RLock()
	defer currentTraffic.Mutex.RUnlock()
	return currentTraffic.Total, currentTraffic.RecordingSince
}

// GetMemoryInfo returns information about the memory usage
func GetMemoryInfo() (uint64, uint64, uint64, int) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println(err)
		return 0, 0, 0, 0
	}
	return memInfo.Free, memInfo.Used, memInfo.Total, int((float64(memInfo.Used) / float64(memInfo.Total)) * 100)
}

// GetDiskInfo returns information about the disk usage
func GetDiskInfo() (uint64, uint64, uint64, int) {
	diskInfo, err := disk.Usage(configuration.Get().DataDir)
	if err != nil {
		fmt.Println(err)
		return 0, 0, 0, 0
	}
	return diskInfo.Free, diskInfo.Used, diskInfo.Total, int((float64(diskInfo.Used) / float64(diskInfo.Total)) * 100)
}

// GetCpuUsage returns the current CPU usage in percent
func GetCpuUsage() int {
	usage, err := cpu.Percent(0, false)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return int(math.Round(usage[0]))
}

// AddTraffic adds traffic to the current traffic counter
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
