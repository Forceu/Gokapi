package downloadstatus

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"time"
)

// SetDownload creates a new DownloadStatus struct and returns its Id
func SetDownload(file models.File) string {
	status := newDownloadStatus(file)
	configuration.ServerSettings.DownloadStatus[status.Id] = status
	return status.Id
}

// SetComplete removes the download object
func SetComplete(id string) {
	delete(configuration.ServerSettings.DownloadStatus, id)
}

// Clean removes all expires status objects
func Clean() {
	now := time.Now().Unix()
	for _, item := range configuration.ServerSettings.DownloadStatus {
		if item.ExpireAt < now {
			delete(configuration.ServerSettings.DownloadStatus, item.Id)
		}
	}
}

// newDownloadStatus initialises the a new DownloadStatus item
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
	for _, status := range configuration.ServerSettings.DownloadStatus {
		if status.FileId == file.Id {
			if status.ExpireAt > time.Now().Unix() {
				return true
			}
		}
	}
	return false
}
