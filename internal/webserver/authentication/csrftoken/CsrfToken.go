package csrftoken

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

var tokens = make(map[string]int64)
var mutex sync.Mutex
var cleanupOnce sync.Once

const ttl = 5 * time.Minute

func Generate() string {
	token := helper.GenerateRandomString(20)
	mutex.Lock()
	tokens[token] = time.Now().Add(ttl).Unix()
	mutex.Unlock()

	cleanupOnce.Do(func() {
		go cleanup(true)
	})
	return token
}

func IsValid(token string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	expireTime, ok := tokens[token]
	if !ok {
		return false
	}
	delete(tokens, token)
	return expireTime > time.Now().Unix()
}

func cleanup(periodic bool) {
	mutex.Lock()
	for token, expireTime := range tokens {
		if expireTime < time.Now().Unix() {
			delete(tokens, token)
		}
	}
	mutex.Unlock()
	if periodic {
		time.Sleep(time.Hour)
		go cleanup(true)
	}

}
