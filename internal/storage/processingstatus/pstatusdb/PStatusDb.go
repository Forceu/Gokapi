package pstatusdb

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/models"
)

var statusMap = make(map[string]models.UploadStatus)
var statusMutex sync.RWMutex
var startCleanupOnce sync.Once

// getAll returns all UploadStatus that were created in the last 24 hours
func getAll() []models.UploadStatus {
	statusMutex.RLock()
	result := make([]models.UploadStatus, 0)
	for _, status := range statusMap {
		result = append(result, status)
	}
	statusMutex.RUnlock()
	return result
}

// GetAllForUser returns all UploadStatus that were created in the last 24 hours for a user
func GetAllForUser(userId int) []models.UploadStatus {
	statusMutex.RLock()
	result := make([]models.UploadStatus, 0)
	for _, status := range statusMap {
		if status.IsForUser(userId) {
			result = append(result, status)
		}
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
	startCleanupOnce.Do(func() { go doGarbageCollection(true) })
}

func deleteAllExpiredStatus() {
	allStatus := getAll()
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
	time.Sleep(1 * time.Hour)
	go doGarbageCollection(true)
}
