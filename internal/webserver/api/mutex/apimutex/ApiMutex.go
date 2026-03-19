package apimutex

import (
	"hash/fnv"
	"sync"
)

const numStripes = 256

var stripes [numStripes]sync.Mutex

func getStripe(objectType int, key string) *sync.Mutex {
	switch objectType {
	case TypeUser, TypeApiKey, TypeMetaData:
		// valid
	default:
		panic("invalid object type")
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte{byte(objectType)})
	_, _ = h.Write([]byte(key))
	return &stripes[h.Sum32()%numStripes]
}

const (
	TypeUser = iota
	TypeApiKey
	TypeMetaData
)

func Lock(objectType int, key string) {
	getStripe(objectType, key).Lock()
}

func Unlock(objectType int, key string) {
	getStripe(objectType, key).Unlock()
}
