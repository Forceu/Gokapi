package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixSessions = "se:"
)

// GetSession returns the session with the given ID or false if not a valid ID
func (p DatabaseProvider) GetSession(id string) (models.Session, bool) {
	hashmapEntry, ok := getHashMap(prefixSessions + id)
	if !ok {
		return models.Session{}, false
	}
	var result models.Session
	err := redigo.ScanStruct(hashmapEntry, &result)
	helper.Check(err)
	return result, true
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func (p DatabaseProvider) SaveSession(id string, session models.Session) {
	setHashMap(buildArgs(prefixSessions + id).AddFlat(session))
	setExpiryAt(prefixSessions+id, session.ValidUntil)
}

// DeleteSession deletes a session with the given ID
func (p DatabaseProvider) DeleteSession(id string) {
	deleteKey(prefixSessions + id)
}

// DeleteAllSessions logs all users out
func (p DatabaseProvider) DeleteAllSessions() {
	deleteAllWithPrefix(prefixSessions)
}
