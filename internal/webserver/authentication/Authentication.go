package authentication

import (
	"crypto/subtle"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"io"
	"net/http"
	"strings"
)

// CookieOauth is the cookie name used for login
const CookieOauth = "state"

// Internal authentication method uses a user / password combination handled by Gokapi
const Internal = 0

// OAuth2 authentication retrieves the users email with Open Connect ID
const OAuth2 = 1

// Header authentication relies on a header from a reverse proxy to parse the user name
const Header = 2

// Disabled authentication ignores all internal authentication procedures. A reverse proxy needs to restrict access
const Disabled = 3

var authSettings models.AuthenticationConfig

// Init needs to be called first to process the authentication configuration
func Init(config models.AuthenticationConfig) {
	authSettings = config
}

// IsAuthenticated returns true if the user provides a valid authentication
func IsAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	switch authSettings.Method {
	case Internal:
		return isGrantedSession(w, r)
	case OAuth2:
		return isGrantedSession(w, r)
	case Header:
		return isGrantedHeader(r)
	case Disabled:
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

// CheckOauthUser checks if the user is allowed to use the Gokapi instance
func CheckOauthUser(userInfo *oidc.UserInfo, w http.ResponseWriter) {
	if isValidOauthUser(userInfo.Email) {
		// TODO revoke session if oauth is not valid any more
		sessionmanager.CreateSession(w)
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
	return IsEqualStringConstantTime(username, authSettings.Username) &&
		IsEqualStringConstantTime(configuration.HashPasswordCustomSalt(password, authSettings.SaltAdmin), authSettings.Password)
}

// IsEqualStringConstantTime uses ConstantTimeCompare to prevent timing attack.
func IsEqualStringConstantTime(s1, s2 string) bool {
	return subtle.ConstantTimeCompare(
		[]byte(strings.ToLower(s1)),
		[]byte(strings.ToLower(s2))) == 1
}

// Sends a redirect HTTP output to the client. Variable url is used to redirect to ./url
func redirect(w http.ResponseWriter, url string) {
	_, _ = io.WriteString(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head></html>")
}

// Logout logs the user out and removes the session
func Logout(w http.ResponseWriter, r *http.Request) {
	if authSettings.Method == Internal || authSettings.Method == OAuth2 {
		sessionmanager.LogoutSession(w, r)
	}
	redirect(w, "login")
}

// IsLogoutAvailable returns true if a logout button should be shown with the current form of authentication
func IsLogoutAvailable() bool {
	return authSettings.Method == Internal || authSettings.Method == OAuth2
}
