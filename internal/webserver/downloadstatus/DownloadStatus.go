package downloadstatus

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"time"
)

var status map[string]models.DownloadStatus

func Init() {
	status = make(map[string]models.DownloadStatus)
}

// SetDownload creates a new DownloadStatus struct and returns its Id
func SetDownload(file models.File) string {
	newStatus := newDownloadStatus(file)
	status[newStatus.Id] = newStatus
	return newStatus.Id
}

// SetComplete removes the download object
func SetComplete(id string) {
	delete(status, id)
}

// Clean removes all expires status objects
func Clean() {
	now := time.Now().Unix()
	for _, item := range status {
		if item.ExpireAt < now {
			delete(status, item.Id)
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
	for _, statusField := range status {
		if statusField.FileId == file.Id {
			if statusField.ExpireAt > time.Now().Unix() {
				return true
			}
		}
	}
	return false
}

func GetAll() map[string]models.DownloadStatus {
	return status
}
