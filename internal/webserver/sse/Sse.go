package sse

import (
	"encoding/json"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/processingstatus/pstatusdb"
	"io"
	"net/http"
	"sync"
	"time"
)

var listeners = make(map[string]listener)
var mutex = sync.RWMutex{}

var maxConnection = 2 * time.Hour
var pingInterval = 15 * time.Second

type listener struct {
	Reply    func(reply string)
	Shutdown func()
}

func addListener(id string, channel listener) {
	mutex.Lock()
	listeners[id] = channel
	mutex.Unlock()
}

func removeListener(id string) {
	mutex.Lock()
	delete(listeners, id)
	mutex.Unlock()
}

type eventFileDownload struct {
	Event              string `json:"event"`
	FileId             string `json:"file_id"`
	DownloadCount      int    `json:"download_count"`
	DownloadsRemaining int    `json:"downloads_remaining"`
}

type eventUploadStatus struct {
	Event        string `json:"event"`
	ChunkId      string `json:"chunk_id"`
	FileId       string `json:"file_id"`
	ErrorMessage string `json:"error_message"`
	UploadStatus int    `json:"upload_status"`
}

type eventData interface {
	eventUploadStatus | eventFileDownload
}

// PublishNewStatus sends a new upload status to all listeners
func PublishNewStatus(uploadStatus models.UploadStatus) {
	event := eventUploadStatus{
		Event:        "uploadStatus",
		ChunkId:      uploadStatus.ChunkId,
		UploadStatus: uploadStatus.CurrentStatus,
		FileId:       uploadStatus.FileId,
		ErrorMessage: uploadStatus.ErrorMessage,
	}
	publishMessage(event)
}

func publishMessage[d eventData](data d) {
	message, err := json.Marshal(data)
	helper.Check(err)

	mutex.RLock()
	for _, channel := range listeners {
		go channel.Reply("event: message\ndata: " + string(message) + "\n\n")
	}
	mutex.RUnlock()
}

// PublishDownloadCount sends a new download count to all listeners
func PublishDownloadCount(file models.File) {
	event := eventFileDownload{
		Event:              "download",
		FileId:             file.Id,
		DownloadCount:      file.DownloadCount,
		DownloadsRemaining: file.DownloadsRemaining,
	}
	if file.UnlimitedDownloads {
		event.DownloadsRemaining = -1
	}
	publishMessage(event)
}

// Shutdown stops the SSE and closes the connection to all listeners
func Shutdown() {
	mutex.RLock()
	for _, channel := range listeners {
		channel.Shutdown()
	}
	mutex.RUnlock()
}

// GetStatusSSE sends all existing upload status and new updates to a new listener
func GetStatusSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Keep-Alive", "timeout=20, max=20")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	creationTime := time.Now()

	replyChannel := make(chan string)
	shutdownChannel := make(chan bool)
	channel := listener{Reply: func(reply string) { replyChannel <- reply }, Shutdown: func() {
		shutdownChannel <- true
	}}
	channelId := helper.GenerateRandomString(20)
	addListener(channelId, channel)

	allStatus := pstatusdb.GetAll()
	for _, status := range allStatus {
		PublishNewStatus(status)
	}
	w.(http.Flusher).Flush()
	for {
		if time.Now().After(creationTime.Add(maxConnection)) {
			removeListener(channelId)
			w.(http.Flusher).Flush()
			return
		}
		select {
		case reply := <-replyChannel:
			_, _ = io.WriteString(w, reply)
		case <-time.After(pingInterval):
			_, _ = io.WriteString(w, "event: ping\n\n")
		case <-ctx.Done():
			removeListener(channelId)
			return
		case <-shutdownChannel:
			removeListener(channelId)
			return
		}
		w.(http.Flusher).Flush()
	}
}
