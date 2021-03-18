package main

import (
	"net/http"
	"time"
)

//If no login occurred during this time, the session will be deleted. Default 30 days
const COOKIE_LIFE = 30 * 24 * time.Hour

type Session struct {
	RenewAt    int64
	ValidUntil int64
}

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

func createSession(w http.ResponseWriter) {
	sessionString, err := generateRandomString(60)
	if err != nil {
		sessionString = unsafeId(60)
	}
	globalConfig.Sessions[sessionString] = Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(COOKIE_LIFE).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(COOKIE_LIFE))
	saveConfig()
}

func logoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(globalConfig.Sessions, cookie.Value)
		saveConfig()
	}
	writeSessionCookie(w, "", time.Now())
}

func writeSessionCookie(w http.ResponseWriter, sessionString string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionString,
		Expires: expiry,
	})
}

