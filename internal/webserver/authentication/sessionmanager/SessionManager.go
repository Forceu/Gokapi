package sessionmanager

/**
Manages the sessions for the admin user or to access password-protected files
*/

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"net/http"
	"time"
)

// TODO add username to check for revocation

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
			session, ok := database.GetSession(sessionString)
			if ok {
				return useSession(w, sessionString, session)
			}
		}
	}
	return false
}

// useSession checks if a session is still valid. It Changes the session string
// if it has // been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, id string, session models.Session) bool {
	if session.ValidUntil < time.Now().Unix() {
		database.DeleteSession(id)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w)
		database.DeleteSession(id)
	}
	return true
}

// CreateSession creates a new session - called after login with correct username / password
// If sessions parameter is nil, it will be loaded from config
func CreateSession(w http.ResponseWriter) {
	sessionString := helper.GenerateRandomString(60)
	database.SaveSession(sessionString, models.Session{
		RenewAt:    time.Now().Add(12 * time.Hour).Unix(),
		ValidUntil: time.Now().Add(cookieLifeAdmin).Unix(),
	}, cookieLifeAdmin)
	writeSessionCookie(w, sessionString, time.Now().Add(cookieLifeAdmin))
}

// LogoutSession logs out user and deletes session
func LogoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		database.DeleteSession(cookie.Value)
	}
	writeSessionCookie(w, "", time.Now())
}

// Writes session cookie to browser
func writeSessionCookie(w http.ResponseWriter, sessionString string, expiry time.Time) {
	c := &http.Cookie{
		Name:    "session_token",
		Value:   sessionString,
		Expires: expiry,
	}
	http.SetCookie(w, c)
}
