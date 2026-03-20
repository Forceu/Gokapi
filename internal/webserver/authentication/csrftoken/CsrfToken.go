package csrftoken

import (
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

var tokens = make(map[string]csrfToken)
var mutex sync.Mutex
var cleanupOnce sync.Once

const ttl = 5 * time.Minute

const (
	TypeLogin = iota
	TypeApiToken
)

type csrfToken struct {
	Type   int
	Expiry int64
}

func Generate(tokenType int) string {
	token := helper.GenerateRandomString(20)
	mutex.Lock()
	tokens[token] = csrfToken{
		Type:   tokenType,
		Expiry: time.Now().Add(ttl).Unix(),
	}
	mutex.Unlock()

	cleanupOnce.Do(func() {
		go cleanup(true)
	})
	return token
}

func IsValid(tokenType int, tokenId string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	token, ok := tokens[tokenId]
	if !ok {
		return false
	}
	delete(tokens, tokenId)
	if token.Type != tokenType {
		return false
	}
	return token.Expiry > time.Now().Unix()
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
