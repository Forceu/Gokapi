package presign

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/models"
)

var presignedUrls = make(map[string]models.Presign)
var mutex sync.RWMutex
var cleanupStarted = false

// Save saves the presigned url
func Save(presign models.Presign) {
	mutex.Lock()
	presignedUrls[presign.Id] = presign
	mutex.Unlock()
	go cleanUp(true)
}

// Get returns the presigned url with the given ID or false if not a valid ID
func Get(id string) (models.Presign, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	result, ok := presignedUrls[id]
	if !ok {
		return models.Presign{}, false
	}
	if result.Expiry < time.Now().Unix() {
		return models.Presign{}, false
	}
	return result, true
}

// Delete deletes the presigned url with the given ID
func Delete(id string) {
	mutex.Lock()
	delete(presignedUrls, id)
	mutex.Unlock()
}

func cleanUp(periodic bool) {
	if cleanupStarted {
		return
	}
	cleanupStarted = true
	mutex.Lock()
	for k, v := range presignedUrls {
		if v.Expiry < time.Now().Unix() {
			delete(presignedUrls, k)
		}
	}
	mutex.Unlock()
	if periodic {
		select {
		case <-time.After(20 * time.Minute):
			go cleanUp(true)
		}
	}
}
