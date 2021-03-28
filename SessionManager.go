package main

/**
Manages the sessions for the admin user or to access password protected files
*/

import (
	"net/http"
	"time"
)


//If no login occurred during this time, the admin session will be deleted. Default 30 days
const COOKIE_LIFE_ADMIN = 30 * 24 * time.Hour

// Structure for cookies
type Session struct {
	RenewAt    int64
	ValidUntil int64
}

// Checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func isValidSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			_, ok := globalConfig.Sessions[sessionString]
			if ok {
				return useSession(w, sessionString)
			}
		}
	}
	return false
}

// Checks if a session is still valid. Changes the session string if it has
// been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, sessionString string) bool {
	session := globalConfig.Sessions[sessionString]
	if session.ValidUntil < time.Now().Unix() {
		delete(globalConfig.Sessions, sessionString)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		createSession(w)
		delete(globalConfig.Sessions, sessionString)
		saveConfig()
	}
	return true
}

//Creates a new session - called after login with correct username / password
func createSession(w http.ResponseWriter) {
	sessionString, err := generateRandomString(60)
	if err != nil {
		sessionString = unsafeId(60)
	}
	globalConfig.Sessions[sessionString] = Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(COOKIE_LIFE_ADMIN).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(COOKIE_LIFE_ADMIN))
	saveConfig()
}

// Logs out user and deletes session
func logoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(globalConfig.Sessions, cookie.Value)
		saveConfig()
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
