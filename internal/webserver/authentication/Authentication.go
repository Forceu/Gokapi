package authentication

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/models"
	"Gokapi/internal/webserver/sessionmanager"
	"crypto/subtle"
	"github.com/coreos/go-oidc/v3/oidc"
	"io"
	"net/http"
	"strings"
)

const CookieOauth = "state"


const AuthenticationInternal = 0
const AuthenticationOAuth2 = 1
const AuthenticationHeader = 2
const AuthenticationDisabled = 3

var authSettings models.AuthenticationConfig

func Init(config models.AuthenticationConfig) {
	authSettings = config
}

func IsAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	switch authSettings.Method {
	case AuthenticationInternal:
		return isGrantedSession(w, r)
	case AuthenticationOAuth2:
		return isGrantedSession(w, r)
	case AuthenticationHeader:
		return isGrantedHeader(r)
	case AuthenticationDisabled:
		return true
	}
	return false
}

// isGrantedHeader returns true if the user was authenticated by a proxy header if enabled
func isGrantedHeader(r *http.Request) bool {

	if authSettings.HeaderKey == "" {
		return false
	}
	value := r.Header.Get(authSettings.HeaderKey)
	if value == "" {
		return false
	}
	if len(authSettings.HeaderUsers) == 0 {
		return true
	}
	return isUserInArray(value, authSettings.HeaderUsers)
}

func isUserInArray(userEntered string, strArray []string) bool {
	for _, user := range strArray {
		if strings.ToLower(user) == strings.ToLower(userEntered) {
			return true
		}
	}
	return false
}

func CheckOauthUser(userInfo *oidc.UserInfo, w http.ResponseWriter) {
	if isValidOauthUser(userInfo.Email) {
		// TODO revoke session if oauth is not valid any more
		sessionmanager.CreateSession(w, nil)
		redirect(w, "admin")
		return
	}
	redirect(w, "error-auth")
}

func isValidOauthUser(name string) bool {
	if name == "" {
		return false
	}
	if len(authSettings.OauthUsers) == 0 {
		return true
	}
	return isUserInArray(name, authSettings.OauthUsers)
}

// isGrantedSession returns true if the user holds a valid internal session cookie
func isGrantedSession(w http.ResponseWriter, r *http.Request) bool {
	return sessionmanager.IsValidSession(w, r)
}

// IsCorrectUsernameAndPassword checks if a provided username and password is correct
func IsCorrectUsernameAndPassword(username, password string) bool {
	return isEqualStringConstantTime(username, authSettings.Username) &&
		isEqualStringConstantTime(configuration.HashPassword(password, false), authSettings.Password)
}

// Use ConstantTimeCompare to prevent timing attack.
func isEqualStringConstantTime(s1, s2 string) bool {
	return subtle.ConstantTimeCompare(
		[]byte(strings.ToLower(s1)),
		[]byte(strings.ToLower(s2))) == 1
}

// Sends a redirect HTTP output to the client. Variable url is used to redirect to ./url
func redirect(w http.ResponseWriter, url string) {
	_, _ = io.WriteString(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head></html>")
}

func GetMethod() int {
	return authSettings.Method
}

func Logout(w http.ResponseWriter, r *http.Request) {
	switch authSettings.Method {
	case AuthenticationInternal:
		sessionmanager.LogoutSession(w, r)
	case AuthenticationOAuth2:
		sessionmanager.LogoutSession(w, r)
	case AuthenticationHeader:
		// TODO
	}
	redirect(w, "login")
}

func IsLogoutAvailable() bool {
	return authSettings.Method == AuthenticationInternal || authSettings.Method == AuthenticationOAuth2
}