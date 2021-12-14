package authentication

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/webserver/sessionmanager"
	"net/http"
	"strings"
)

func IsAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	settings := configuration.GetServerSettingsReadOnly()
	configuration.ReleaseReadOnly()
	switch settings.AuthenticationMethod {
	case configuration.AuthenticationInternal:
		return isGrantedSession(w, r)
	case configuration.AuthenticationOAuth2:
		return false // TODO
	case configuration.AuthenticationHeader:
		return isGrantedHeader(r)
	case configuration.AuthenticationDisabled:
		return true
	}
	if isGrantedHeader(r) {
		return true
	}
	if isGrantedSession(w, r) {
		return true
	}
	return false
}

// isGrantedHeader returns true if the user was authenticated by a proxy header if enabled
func isGrantedHeader(r *http.Request) bool {
	settings := configuration.GetServerSettingsReadOnly()
	defer configuration.ReleaseReadOnly()

	if settings.LoginHeaderKey == "" {
		return false
	}

	value := r.Header.Get(settings.LoginHeaderKey)
	if value == "" {
		return false
	}
	// TODO
	// if settings.LoginHeaderForceUsername {
	//	return strings.ToLower(value) == strings.ToLower(settings.AdminName)
	// } else {
	return true
	// }
}

// isGrantedSession returns true if the user holds a valid internal session cookie
func isGrantedSession(w http.ResponseWriter, r *http.Request) bool {
	return sessionmanager.IsValidSession(w, r)
}

// IsCorrectUsernameAndPassword checks if a provided username and password is correct
func IsCorrectUsernameAndPassword(username, password string) bool {
	settings := configuration.GetServerSettingsReadOnly()
	configuration.ReleaseReadOnly()
	return strings.ToLower(username) == strings.ToLower(settings.AdminName) && configuration.HashPassword(password, false) == settings.AdminPassword
}
