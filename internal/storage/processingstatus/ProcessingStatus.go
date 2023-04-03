package processingstatus

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/r3labs/sse/v2"
	"time"
)

// StatusHashingOrEncrypting indicates that the file has been completely uploaded, but is now processed by Gokapi
const StatusHashingOrEncrypting = 0

// StatusUploading indicates that the file has been processed, but is now moved to the data filesystem
const StatusUploading = 1

var sseServer *sse.Server

// Init passes the SSE server, so that notifications can be sent
func Init(srv *sse.Server) {
	sseServer = srv
}

func passNewStatus(newStatus models.UploadStatus) {
	if sseServer == nil {
		panic("sseServer not initialised")
	}
	status, err := newStatus.ToJson()
	helper.Check(err)
	sseServer.Publish("changes", &sse.Event{
		Data: status,
	})
}

// Set sets the status for an id
func Set(id string, status int) {
	newStatus := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: status,
		LastUpdate:    time.Now().Unix(),
	}
	oldStatus, ok := database.GetUploadStatus(newStatus.ChunkId)
	if ok && oldStatus.LastUpdate > newStatus.LastUpdate {
		return
	}
	passNewStatus(newStatus)
	database.SaveUploadStatus(newStatus)
}
