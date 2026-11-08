package sessionmanager

/**
Manages the sessions for the admin user or to access password-protected files
*/

import (
	"net/http"
	"time"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

// After that time a session ID will be rotated
const renewalTime = 12 * time.Hour

// If no login occurred during this time, the admin session will be deleted. Default 7 days
const sessionLifetime = 7 * 24 * time.Hour

// After that time the user is forced to log in again, even if the session was active the whole time
// If the user has not been using the session for at least 4 hours and the expiry would be in the next
// 48 hours: The session will expire to have a graceful logout experience
const forceLoginAfter = 32 * 24 * time.Hour

const lengthSessionId = 60

// IsValidSession checks if the user is submitting a valid session token
// If a valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func IsValidSession(w http.ResponseWriter, r *http.Request, isOauth bool, OAuthRecheckInterval int) (models.User, bool) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			// Hash session string so it is not stored in plain text in the database
			sessionId := helper.Sha256Sum(sessionString)
			session, ok := database.GetSession(sessionId)
			if ok {
				user, userExists := database.GetUser(session.UserId)
				if !userExists {
					database.DeleteSession(sessionId)
					return models.User{}, false
				}
				return user, useSession(w, sessionId, session, isOauth, OAuthRecheckInterval, user)
			}
		}
	}
	return models.User{}, false
}

// useSession checks if a session is still valid. It Changes the session string
// if it has been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, id string, session models.Session, isOauth bool, OAuthRecheckInterval int, user models.User) bool {
	if hasSessionExpired(session, user) {
		database.DeleteSession(id)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w, isOauth, OAuthRecheckInterval, session.UserId, session.Expiry)
		database.DeleteSession(id)
	}
	go database.UpdateUserLastOnline(user.Id)
	return true
}

func hasSessionExpired(session models.Session, user models.User) bool {
	now := time.Now()
	if session.ValidUntil < now.Unix() {
		return true
	}
	if session.Expiry < now.Unix() {
		return true
	}
	// Graceful logout: if the user has been away for a while and the hard limit
	// is near, end the session now instead of cutting them off mid-work later
	if session.Expiry < now.Add(48*time.Hour).Unix() &&
		user.LastOnline < now.Add(-4*time.Hour).Unix() {
		return true
	}
	return false
}

// CreateSession creates a new session - called after login with correct username / password or other successful authentication method
func CreateSession(w http.ResponseWriter, isOauth bool, OAuthRecheckInterval, userId int, forcedExpiry int64) {
	renewAt := time.Now().Add(renewalTime)
	timeExpiry := time.Now().Add(sessionLifetime)
	timeMaxLife := time.Now().Add(forceLoginAfter)
	if isOauth {
		timeExpiry = time.Now().Add(time.Duration(OAuthRecheckInterval) * time.Hour)
		timeMaxLife = time.Now().Add(time.Duration(OAuthRecheckInterval) * time.Hour)
	}
	if forcedExpiry > 0 {
		forcedExpiryTime := time.Unix(forcedExpiry, 0)
		if timeExpiry.After(forcedExpiryTime) {
			timeExpiry = forcedExpiryTime
		}
		if timeMaxLife.After(forcedExpiryTime) {
			timeMaxLife = forcedExpiryTime
		}
	}

	sessionString := helper.GenerateRandomString(lengthSessionId)
	database.SaveSession(helper.Sha256Sum(sessionString), models.Session{
		RenewAt:    renewAt.Unix(),
		ValidUntil: timeExpiry.Unix(),
		Expiry:     timeMaxLife.Unix(),
		UserId:     userId,
	})
	writeSessionCookie(w, sessionString, timeExpiry)
}

// LogoutSession logs out user and deletes session
func LogoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		database.DeleteSession(helper.Sha256Sum(cookie.Value))
	}
	writeSessionCookie(w, "", time.Now())
}

// Writes session cookie to browser
func writeSessionCookie(w http.ResponseWriter, sessionString string, expiry time.Time) {
	c := &http.Cookie{
		Name:     "session_token",
		Value:    sessionString,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, c)
}
