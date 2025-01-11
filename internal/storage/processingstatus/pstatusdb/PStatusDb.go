package pstatusdb

import (
	"github.com/forceu/gokapi/internal/models"
	"sync"
	"time"
)

var statusMap = make(map[string]models.UploadStatus)
var statusMutex sync.RWMutex
var isGbStarted = false

// GetAll returns all UploadStatus that were created in the last 24 hours
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

// Set saves the upload status for 24 hours
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
	statusMutex.Lock()
	newStatusMap := make(map[string]models.UploadStatus)
	for _, status := range allStatus {
		if status.Creation > cutOff {
			newStatusMap[status.ChunkId] = status
		}
	}
	statusMap = newStatusMap
	statusMutex.Unlock()
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
