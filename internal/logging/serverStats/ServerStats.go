package serverStats

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
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
}

func Shutdown() {
	saveTraffic()
}

func saveTraffic() {
	currentTraffic.Mutex.RLock()
	database.SaveStatTraffic(currentTraffic.Total)
	currentTraffic.Mutex.RUnlock()
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

func AddTraffic(file models.File) {
	currentTraffic.Mutex.Lock()
	currentTraffic.Total = currentTraffic.Total + uint64(file.SizeBytes)
	requireSave := time.Since(currentTraffic.LastUpdate) > trafficSaveInterval
	currentTraffic.LastUpdate = time.Now()
	currentTraffic.Mutex.Unlock()

	if requireSave {
		saveTraffic()
	}
}
