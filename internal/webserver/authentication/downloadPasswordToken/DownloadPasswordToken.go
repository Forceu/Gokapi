package downloadPasswordToken

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

var tokens = make(map[string]pwToken)
var mutex sync.Mutex
var cleanupOnce sync.Once

type pwToken struct {
	FileId string
	Expiry int64
}

const ttl = 5 * time.Minute

func Generate(fileId string) string {
	token := helper.GenerateRandomString(60)
	mutex.Lock()
	tokens[token] = pwToken{
		FileId: fileId,
		Expiry: time.Now().Add(ttl).Unix(),
	}
	mutex.Unlock()

	cleanupOnce.Do(func() {
		go cleanup(true)
	})
	return token
}

func IsValid(tokenId, fileId string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	token, ok := tokens[tokenId]
	if !ok {
		return false
	}
	if token.FileId != fileId {
		return false
	}
	if token.Expiry < time.Now().Unix() {
		delete(tokens, tokenId)
		return false
	}
	return true
}

func cleanup(periodic bool) {
	mutex.Lock()
	for tokenId, token := range tokens {
		if token.Expiry < time.Now().Unix() {
			delete(tokens, tokenId)
		}
	}
	mutex.Unlock()
	if periodic {
		time.Sleep(time.Hour)
		go cleanup(true)
	}

}
