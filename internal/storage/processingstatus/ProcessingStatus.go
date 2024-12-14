package processingstatus

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/processingstatus/pstatusdb"
	"github.com/forceu/gokapi/internal/webserver/sse"
)

// StatusHashingOrEncrypting indicates that the file has been completely uploaded, but is now processed by Gokapi
const StatusHashingOrEncrypting = 0

// StatusUploading indicates that the file has been processed, but is now moved to the data filesystem
const StatusUploading = 1

// StatusFinished indicates that the file has been fully processed and uploaded
const StatusFinished = 2

// StatusError indicates that there was an error during the upload
const StatusError = 3

// Set sets the status for an id
func Set(id string, status int, file models.File, err error) {
	newStatus := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: status,
		FileId:        file.Id,
	}
	if err != nil {
		newStatus.ErrorMessage = err.Error()
	}
	pstatusdb.Set(newStatus)
	go sse.PublishNewStatus(newStatus)
}
