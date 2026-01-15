package chunkreservation

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

var reservedChunks = make(map[string]map[string]reservation)
var reservationMutex sync.RWMutex
var gcIsRunning = false

const timeReservationWithoutUpload = 4 * 60
const timeReservationWithUpload = 23 * 60 * 60

type reservation struct {
	Uuid   string
	Expiry int64
}

func GetCount(id string) int {
	reservationMutex.RLock()
	defer reservationMutex.RUnlock()
	length := len(reservedChunks[id])
	return length
}

func New(id string) string {
	reservationMutex.Lock()
	defer reservationMutex.Unlock()

	uuid := helper.GenerateRandomString(32)
	if reservedChunks[id] == nil {
		reservedChunks[id] = make(map[string]reservation)
	}
	reservedChunks[id][uuid] = reservation{
		Uuid:   uuid,
		Expiry: time.Now().Unix() + timeReservationWithoutUpload,
	}

	if !gcIsRunning {
		gcIsRunning = true
		go cleanUp(true)
	}
	return uuid
}

func SetUploading(id string, uuid string) bool {
	reservationMutex.Lock()
	defer reservationMutex.Unlock()

	if reservedChunks[id] == nil {
		return false
	}
	chunk, ok := reservedChunks[id][uuid]
	if !ok {
		return false
	}
	if chunk.Expiry < time.Now().Unix() {
		return false
	}
	chunk.Expiry = time.Now().Unix() + timeReservationWithUpload
	reservedChunks[id][uuid] = chunk
	return true
}

func SetComplete(id string, uuid string) {
	reservationMutex.Lock()
	delete(reservedChunks[id], uuid)
	reservationMutex.Unlock()
}

func cleanUp(isPeriodic bool) {
	reservationMutex.Lock()
	for id, chunks := range reservedChunks {
		now := time.Now().Unix()
		for uuid, reservedChunk := range chunks {
			if reservedChunk.Expiry < now {
				delete(reservedChunks[id], uuid)
			}
		}
	}
	reservationMutex.Unlock()

	if isPeriodic {
		go func() {
			time.Sleep(time.Minute * 5)
			cleanUp(true)
		}()
	}
}
