package redis

import (
	"github.com/forceu/gokapi/internal/models"
	"strconv"
)

const (
	prefixSessions           = "se:"
	hashmapSessionRenew      = "renew"
	hashmapSessionValidUntil = "valid"
)

func dbToSession(input map[string]string) (models.Session, error) {
	renew, err := strconv.ParseInt(input[hashmapSessionRenew], 10, 64)
	if err != nil {
		return models.Session{}, err
	}
	valid, err := strconv.ParseInt(input[hashmapSessionValidUntil], 10, 64)
	if err != nil {
		return models.Session{}, err
	}
	return models.Session{
		RenewAt:    renew,
		ValidUntil: valid,
	}, nil
}

func sessionToDb(input models.Session) map[string]string {
	return map[string]string{
		hashmapSessionRenew:      strconv.FormatInt(input.RenewAt, 10),
		hashmapSessionValidUntil: strconv.FormatInt(input.ValidUntil, 10),
	}
}

// GetSession returns the session with the given ID or false if not a valid ID
func (p DatabaseProvider) GetSession(id string) (models.Session, bool) {

	hashmapEntry, ok := getHashMap(prefixSessions + id)
	if !ok {
		return models.Session{}, false
	}
	result, err := dbToSession(hashmapEntry)
	if err != nil {
		return models.Session{}, false
	}
	return result, true
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func (p DatabaseProvider) SaveSession(id string, session models.Session) {
	setHashMap(prefixSessions+id, sessionToDb(session))
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
