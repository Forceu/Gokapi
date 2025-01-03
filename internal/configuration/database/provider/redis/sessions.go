package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strings"
)

const (
	prefixSessions = "se:"
)

// GetSession returns the session with the given ID or false if not a valid ID
func (p DatabaseProvider) GetSession(id string) (models.Session, bool) {
	hashmapEntry, ok := p.getHashMap(prefixSessions + id)
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
	p.setHashMap(p.buildArgs(prefixSessions + id).AddFlat(session))
	p.setExpiryAt(prefixSessions+id, session.ValidUntil)
}

// DeleteSession deletes a session with the given ID
func (p DatabaseProvider) DeleteSession(id string) {
	p.deleteKey(prefixSessions + id)
}

// DeleteAllSessions logs all users out
func (p DatabaseProvider) DeleteAllSessions() {
	p.deleteAllWithPrefix(prefixSessions)
}

// DeleteAllSessionsByUser logs the specific users out
func (p DatabaseProvider) DeleteAllSessionsByUser(userId int) {
	maps := p.getAllHashesWithPrefix(prefixSessions)
	for k, v := range maps {
		var result models.Session
		err := redigo.ScanStruct(v, &result)
		helper.Check(err)
		if result.UserId == userId {
			p.DeleteSession(strings.Replace(k, prefixSessions, "", 1))
		}
	}
}
