package processingstatus

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/sse"
)

// StatusHashingOrEncrypting indicates that the file has been completely uploaded, but is now processed by Gokapi
const StatusHashingOrEncrypting = 0

// StatusUploading indicates that the file has been processed, but is now moved to the data filesystem
const StatusUploading = 1

// Set sets the status for an id
func Set(id string, status int) {
	newStatus := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: status,
	}
	oldStatus, ok := database.GetUploadStatus(newStatus.ChunkId)
	if ok && oldStatus.CurrentStatus > newStatus.CurrentStatus {
		return
	}
	database.SaveUploadStatus(newStatus)
	go sse.PublishNewStatus(newStatus)
}
