package downloadstatus

import (
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
)

// SetDownload creates a new DownloadStatus struct and returns its Id
func SetDownload(file models.File) string {
	downloadId := helper.GenerateRandomString(30)
	dataStorage.SaveDownloadStatus(downloadId, file.Id)
	return downloadId
}

// SetComplete removes the download object
func SetComplete(id string) {
	dataStorage.DeleteDownloadStatus(id)
}

// IsCurrentlyDownloading returns true if file is currently being downloaded
func IsCurrentlyDownloading(file models.File) bool {
	for _, status := range dataStorage.GetAllDownloadStatus() {
		if status == file.Id {
			return true
		}
	}
	return false
}
