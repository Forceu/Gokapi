package pStatusDb

import (
	"github.com/forceu/gokapi/internal/models"
	"sync"
	"time"
)

var statusMap = make(map[string]models.UploadStatus)
var statusMutex sync.RWMutex
var isGbStarted = false

func GetAll() []models.UploadStatus {
	statusMutex.RLock()
	result := make([]models.UploadStatus, len(statusMap))
	i := 0
	for _, status := range statusMap {
		result[i] = status
		i++
	}
	statusMutex.RUnlock()
	return result
}

func Set(status models.UploadStatus) {
	statusMutex.Lock()
	oldStatus, ok := statusMap[status.ChunkId]
	if ok && oldStatus.CurrentStatus > status.CurrentStatus {
		statusMutex.Unlock()
		return
	}
	status.Creation = time.Now().Unix()
	statusMap[status.ChunkId] = status
	statusMutex.Unlock()
	if !isGbStarted {
		isGbStarted = true
		go doGarbageCollection(true)
	}
}

func deleteAllExpiredStatus() {
	allStatus := GetAll()
	cutOff := time.Now().Add(-24 * time.Hour).Unix()
	for _, status := range allStatus {
		if status.Creation < cutOff {
			statusMutex.Lock()
			delete(statusMap, status.ChunkId)
			statusMutex.Unlock()
		}
	}
}

func doGarbageCollection(runPeriodically bool) {
	deleteAllExpiredStatus()
	if !runPeriodically {
		return
	}
	select {
	case <-time.After(1 * time.Hour):
		doGarbageCollection(true)
	}
}
