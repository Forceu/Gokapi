package downloadstatus

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"sync"
	"time"
)

var statusMap = make(map[string]models.DownloadStatus)
var statusMutex sync.RWMutex

// SetDownload creates a new DownloadStatus struct and returns its Id
func SetDownload(file models.File) string {
	newStatus := newDownloadStatus(file)
	statusMutex.Lock()
	statusMap[newStatus.Id] = newStatus
	statusMutex.Unlock()
	return newStatus.Id
}

// SetComplete removes the download object
func SetComplete(downloadStatusId string) {
	statusMutex.Lock()
	delete(statusMap, downloadStatusId)
	statusMutex.Unlock()
}

// Clean removes all expires status objects
func Clean() {
	now := time.Now().Unix()
	for _, item := range statusMap {
		if item.ExpireAt < now {
			SetComplete(item.Id)
		}
	}
}

// newDownloadStatus initialises a new DownloadStatus item
func newDownloadStatus(file models.File) models.DownloadStatus {
	s := models.DownloadStatus{
		Id:       helper.GenerateRandomString(30),
		FileId:   file.Id,
		ExpireAt: time.Now().Add(24 * time.Hour).Unix(),
	}
	return s
}

// IsCurrentlyDownloading returns true if file is currently being downloaded
func IsCurrentlyDownloading(file models.File) bool {
	isDownloading := false
	statusMutex.RLock()
	for _, status := range statusMap {
		if status.FileId == file.Id {
			if status.ExpireAt > time.Now().Unix() {
				isDownloading = true
				break
			}
		}
	}
	statusMutex.RUnlock()
	return isDownloading
}

// SetAllComplete removes all download status associated with this file
func SetAllComplete(fileId string) {
	statusMutex.Lock()
	for _, status := range statusMap {
		if status.FileId == fileId {
			delete(statusMap, status.Id)
		}
	}
	statusMutex.Unlock()
}

// DeleteAll removes all download status
func DeleteAll() {
	statusMutex.Lock()
	statusMap = make(map[string]models.DownloadStatus)
	statusMutex.Unlock()
}
