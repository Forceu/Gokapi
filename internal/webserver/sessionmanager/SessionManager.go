package sessionmanager

/**
Manages the sessions for the admin user or to access password protected files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"net/http"
	"time"
)

// If no login occurred during this time, the admin session will be deleted. Default 30 days
const cookieLifeAdmin = 30 * 24 * time.Hour

// IsValidSession checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func IsValidSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			configuration.LockSessions()
			defer func() { configuration.UnlockSessionsAndSave() }()
			_, ok := configuration.ServerSettings.Sessions[sessionString]
			if ok {
				return useSession(w, sessionString)
			}
		}
	}
	return false
}

// useSession checks if a session is still valid. It Changes the session string
// if it has // been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, sessionString string) bool {
	session := configuration.ServerSettings.Sessions[sessionString]
	if session.ValidUntil < time.Now().Unix() {
		delete(configuration.ServerSettings.Sessions, sessionString)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w, true)
		delete(configuration.ServerSettings.Sessions, sessionString)
	}
	return true
}

// CreateSession creates a new session - called after login with correct username / password
func CreateSession(w http.ResponseWriter, isLocked bool) {
	if !isLocked {
		configuration.LockSessions()
		defer func() { configuration.UnlockSessionsAndSave() }()
	}
	sessionString := helper.GenerateRandomString(60)
	configuration.ServerSettings.Sessions[sessionString] = models.Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(cookieLifeAdmin).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(cookieLifeAdmin))
}

// LogoutSession logs out user and deletes session
func LogoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		configuration.LockSessions()
		delete(configuration.ServerSettings.Sessions, cookie.Value)
		configuration.UnlockSessionsAndSave()
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
