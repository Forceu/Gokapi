package csrftoken

import (
	"net/http"
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

var tokens = make(map[string]csrfToken)
var mutex sync.Mutex
var cleanupOnce sync.Once

const ttl = 5 * time.Minute
const cookieName = "gokapi_csrf"

const (
	TypeLogin    = iota
	TypeApiToken // currently not used
)

type csrfToken struct {
	Type   int
	Expiry int64
	Nonce  string // value of the companion cookie
}

func Generate(w http.ResponseWriter, tokenType int) string {
	token := helper.GenerateRandomString(20)
	nonce := helper.GenerateRandomString(20)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    nonce,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	mutex.Lock()
	tokens[token] = csrfToken{
		Type:   tokenType,
		Expiry: time.Now().Add(ttl).Unix(),
		Nonce:  nonce,
	}
	mutex.Unlock()

	cleanupOnce.Do(func() {
		go cleanup(true)
	})
	return token
}

func IsValid(r *http.Request, tokenType int, tokenId string) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	mutex.Lock()
	defer mutex.Unlock()
	token, ok := tokens[tokenId]
	if !ok {
		return false
	}
	delete(tokens, tokenId) //always deletes token, even if it was wrong type or wrong nonce
	return token.Type == tokenType &&
		token.Expiry > time.Now().Unix() &&
		token.Nonce == c.Value
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
