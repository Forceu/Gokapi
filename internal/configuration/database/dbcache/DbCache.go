package dbcache

import (
	"sync"
	"time"
)

var lastOnlineTimeUpdate map[int]int64
var lastOnlineTimeMutex sync.Mutex

// Init starts the DB Cache
func Init() {
	lastOnlineTimeUpdate = make(map[int]int64)
}

// LastOnlineRequiresSave returns true if the last update time of the user is older than 60 seconds.
func LastOnlineRequiresSave(userId int) bool {
	lastOnlineTimeMutex.Lock()
	timestamp := time.Now().Unix()
	defer lastOnlineTimeMutex.Unlock()
	if lastOnlineTimeUpdate[userId] < (timestamp - 60) {
		lastOnlineTimeUpdate[userId] = timestamp
		return true
	}
	return false
}
