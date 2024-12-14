package pStatusDb

import (
	"github.com/forceu/gokapi/internal/models"
	"sync"
)

var statusMap = make(map[string]models.UploadStatus)
var statusMutex sync.RWMutex

// TODO limit to 24h

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
	defer statusMutex.Unlock()
	oldStatus, ok := statusMap[status.ChunkId]
	if ok && oldStatus.CurrentStatus > status.CurrentStatus {
		return
	}
	statusMap[status.ChunkId] = status
}
