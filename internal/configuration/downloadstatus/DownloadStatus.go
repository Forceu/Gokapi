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
	settings := configuration.GetServerSettings()
	settings.DownloadStatus[status.Id] = status
	configuration.ReleaseAndSave()
	return status.Id
}

// SetComplete removes the download object
func SetComplete(id string) {
	settings := configuration.GetServerSettings()
	delete(settings.DownloadStatus, id)
	configuration.ReleaseAndSave()
}

// Clean removes all expires status objects
func Clean() {
	settings := configuration.GetServerSettings()
	now := time.Now().Unix()
	for _, item := range settings.DownloadStatus {
		if item.ExpireAt < now {
			delete(settings.DownloadStatus, item.Id)
		}
	}
	configuration.Release()
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
func IsCurrentlyDownloading(file models.File, settings *configuration.Configuration) bool {
	for _, status := range settings.DownloadStatus {
		if status.FileId == file.Id {
			if status.ExpireAt > time.Now().Unix() {
				return true
			}
		}
	}
	return false
}
