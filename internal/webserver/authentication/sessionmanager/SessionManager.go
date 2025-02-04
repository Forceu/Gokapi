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

// If no login occurred during this time, the admin session will be deleted. Default 30 days
const cookieLifeAdmin = 30 * 24 * time.Hour
const lengthSessionId = 60

// IsValidSession checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func IsValidSession(w http.ResponseWriter, r *http.Request, isOauth bool, OAuthRecheckInterval int) (models.User, bool) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			session, ok := database.GetSession(sessionString)
			if ok {
				user, userExists := database.GetUser(session.UserId)
				if !userExists {
					return user, false
				}
				return user, useSession(w, sessionString, session, isOauth, OAuthRecheckInterval)
			}
		}
	}
	return models.User{}, false
}

// useSession checks if a session is still valid. It Changes the session string
// if it has // been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, id string, session models.Session, isOauth bool, OAuthRecheckInterval int) bool {
	if session.ValidUntil < time.Now().Unix() {
		database.DeleteSession(id)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w, isOauth, OAuthRecheckInterval, session.UserId)
		database.DeleteSession(id)
	}
	go database.UpdateUserLastOnline(session.UserId)
	return true
}

// CreateSession creates a new session - called after login with correct username / password
// If sessions parameter is nil, it will be loaded from config
func CreateSession(w http.ResponseWriter, isOauth bool, OAuthRecheckInterval int, userId int) {
	timeExpiry := time.Now().Add(cookieLifeAdmin)
	if isOauth {
		timeExpiry = time.Now().Add(time.Duration(OAuthRecheckInterval) * time.Hour)
	}

	sessionString := helper.GenerateRandomString(lengthSessionId)
	database.SaveSession(sessionString, models.Session{
		RenewAt:    time.Now().Add(12 * time.Hour).Unix(),
		ValidUntil: timeExpiry.Unix(),
		UserId:     userId,
	})
	writeSessionCookie(w, sessionString, timeExpiry)
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
