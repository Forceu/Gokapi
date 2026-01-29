package dbcache

import (
	"sync"
	"time"
)

const (
	TypeUserLastOnline = iota
	TypeApiLastUsed
)

var cacheStore []cacheEntry
var globalMutex sync.RWMutex

type cacheEntry struct {
	mapString map[string]int64
	mapInt    map[int]int64
	mutex     sync.Mutex
}

func (c *cacheEntry) Init() {
	c.mapString = make(map[string]int64)
	c.mapInt = make(map[int]int64)
}

func Init() {
	globalMutex.Lock()
	cacheStore = make([]cacheEntry, 2)
	cacheStore[TypeUserLastOnline].Init()
	cacheStore[TypeApiLastUsed].Init()
	globalMutex.Unlock()
}

func (c *cacheEntry) IsUpdateRequiredString(key string, maxDiffSeconds int64) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	timeNow := time.Now().Unix()
	if c.mapString[key] < timeNow-maxDiffSeconds {
		c.mapString[key] = timeNow
		return true
	}
	return false
}
func (c *cacheEntry) IsUpdateRequiredInt(key int, maxDiffSeconds int64) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	timeNow := time.Now().Unix()
	if c.mapInt[key] < timeNow-maxDiffSeconds {
		c.mapInt[key] = timeNow
		return true
	}
	return false
}

// RequireSaveUserOnline returns false if no write is necessary.
// To reduce database writes, the entry is only updated if the last timestamp is more than 30 seconds old
func RequireSaveUserOnline(userId int) bool {
	return cacheStore[TypeUserLastOnline].IsUpdateRequiredInt(userId, 30)
}

// RequireSaveApiKeyUsage returns false if no write is necessary.
// To reduce database writes, the entry is only updated if the last timestamp is more than 30 seconds old
func RequireSaveApiKeyUsage(id string) bool {
	return cacheStore[TypeUserLastOnline].IsUpdateRequiredString(id, 30)
}
