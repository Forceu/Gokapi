package sse

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
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

func PublishNewStatus(reply string) {
	mutex.RLock()
	for _, channel := range listeners {
		go channel.Reply(reply)
	}
	mutex.RUnlock()
}

func Shutdown() {
	mutex.RLock()
	for _, channel := range listeners {
		channel.Shutdown()
	}
	mutex.RUnlock()
}

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

	allStatus := database.GetAllUploadStatus()
	for _, status := range allStatus {
		jsonOutput, err := status.ToJson()
		helper.Check(err)
		_, _ = io.WriteString(w, string(jsonOutput)+"\n")
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
			_, _ = io.WriteString(w, "{\"type\":\"ping\"}\n")
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
