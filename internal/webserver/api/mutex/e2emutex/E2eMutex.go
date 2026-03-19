package e2emutex

import (
	"sync"
	"time"
)

const autoUnlockDuration = 30 * time.Second

var mutexMap = make(map[int]*timedMutex)
var globalMutex sync.Mutex

type timedMutex struct {
	mutex      sync.Mutex
	timer      *time.Timer
	isLocked   bool
	stateMutex sync.Mutex // protects isLocked and timer
}

// Lock locks the mutex for the given user
// Automatically unlocks after 30 seconds
func Lock(userId int) {
	getMutex(userId).lock()
}

// Unlock unlocks the mutex for the given user
// Does nothing if the mutex is not locked
func Unlock(userId int) {
	getMutex(userId).unlock()
}

// IsLocked returns true if the mutex for the given user is locked
func IsLocked(userId int) bool {
	return getMutex(userId).isLocked
}

func getMutex(userId int) *timedMutex {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	m, ok := mutexMap[userId]
	if !ok {
		m = &timedMutex{}
		mutexMap[userId] = m
	}
	return m
}

func (t *timedMutex) lock() {
	t.mutex.Lock()

	t.stateMutex.Lock()
	t.isLocked = true
	t.timer = time.AfterFunc(autoUnlockDuration, func() {
		t.unlock()
	})
	t.stateMutex.Unlock()
}

func (t *timedMutex) unlock() {
	t.stateMutex.Lock()
	defer t.stateMutex.Unlock()

	if !t.isLocked {
		return // already unlocked, nothing to do
	}

	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}

	t.isLocked = false
	t.mutex.Unlock()
}
