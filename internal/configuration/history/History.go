package history

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"time"
	"net/http"
	"fmt"

)

// newDownloadHistory initialises the a new DownloadHistory item
func newDownloadHistory(file models.File, r *http.Request) models.DownloadHistory {
	s := models.DownloadHistory{
		Id:       helper.GenerateRandomString(30),
		FileId:   file.Id,
		DownloaderIP: r.RemoteAddr,
		DownloaderUA: r.UserAgent(),
		DownloadDate: time.Now().Add(24 * time.Hour).Unix(),
	}
	return s
}


// LogHistory creates a new DownloadHistory struct and returns its Id
func LogHistory(file models.File, r *http.Request) string {
	status := newDownloadHistory(file, r)
	settings := configuration.GetServerSettings()

	fmt.Println(status.Id)

	
	settings.DownloadHistory[status.Id] = status
	configuration.ReleaseAndSave()
	return status.Id
}

