package sessionmanager

/**
Manages the sessions for the admin user or to access password protected files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"net/http"
	"sync"
	"time"
)

// If no login occurred during this time, the admin session will be deleted. Default 30 days
const cookieLifeAdmin = 30 * 24 * time.Hour

var mutex sync.Mutex

// IsValidSession checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func IsValidSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			mutex.Lock()
			sessions := configuration.GetSessions()
			defer func() { unlockAndSave() }()
			_, ok := (*sessions)[sessionString]
			if ok {
				return useSession(w, sessionString)
			}
		}
	}
	return false
}

func unlockAndSave() {
	configuration.Save()
	mutex.Unlock()
}

// useSession checks if a session is still valid. It Changes the session string
// if it has // been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, sessionString string) bool {
	sessions := configuration.GetSessions()
	session := (*sessions)[sessionString]
	if session.ValidUntil < time.Now().Unix() {
		delete(*sessions, sessionString)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w, true)
		delete(*sessions, sessionString)
	}
	return true
}

// CreateSession creates a new session - called after login with correct username / password
func CreateSession(w http.ResponseWriter, isLocked bool) {
	if !isLocked {
		mutex.Lock()
		defer func() { unlockAndSave() }()
	}
	sessionString := helper.GenerateRandomString(60)
	sessions := configuration.GetSessions()
	(*sessions)[sessionString] = models.Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(cookieLifeAdmin).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(cookieLifeAdmin))
}

// LogoutSession logs out user and deletes session
func LogoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		mutex.Lock()
		sessions := configuration.GetSessions()
		delete(*sessions, cookie.Value)
		unlockAndSave()
	}
	writeSessionCookie(w, "", time.Now())
}

// Writes session cookie to browser
func writeSessionCookie(w http.ResponseWriter, sessionString string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionString,
		Expires: expiry,
	})
}
