package webserver

/**
Manages the sessions for the admin user or to access password protected files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/webserver/sessionstructure"
	"net/http"
	"time"
)

// If no login occurred during this time, the admin session will be deleted. Default 30 days
const cookieLifeAdmin = 30 * 24 * time.Hour

// Checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func isValidSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			configuration.Lock()
			defer func() { configuration.UnlockAndSave() }()
			_, ok := configuration.ServerSettings.Sessions[sessionString]
			if ok {
				return useSession(w, sessionString, true)
			}
		}
	}
	return false
}

// Checks if a session is still valid. Changes the session string if it has
// been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, sessionString string, isLocked bool) bool {
	if !isLocked {
		configuration.Lock()
		defer func() { configuration.UnlockAndSave() }()
	}
	session := configuration.ServerSettings.Sessions[sessionString]
	if session.ValidUntil < time.Now().Unix() {
		delete(configuration.ServerSettings.Sessions, sessionString)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		createSession(w, true)
		delete(configuration.ServerSettings.Sessions, sessionString)
	}
	return true
}

// Creates a new session - called after login with correct username / password
func createSession(w http.ResponseWriter, isLocked bool) {
	if !isLocked {
		configuration.Lock()
		defer func() { configuration.UnlockAndSave() }()
	}
	sessionString := helper.GenerateRandomString(60)
	configuration.ServerSettings.Sessions[sessionString] = sessionstructure.Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(cookieLifeAdmin).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(cookieLifeAdmin))
}

// Logs out user and deletes session
func logoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		configuration.Lock()
		delete(configuration.ServerSettings.Sessions, cookie.Value)
		configuration.UnlockAndSave()
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
