package authentication

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/webserver/sessionmanager"
	"net/http"
	"strings"
)

func IsAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	if byDisabledLogin() {
		return true
	}
	if byHeader(r) {
		return true
	}
	if byInternalSession(w, r) {
		return true
	}
	return false
}

// byHeader returns true if the user was authenticated by a proxy header if enabled
func byHeader(r *http.Request) bool {
	settings := configuration.GetServerSettingsReadOnly()
	defer configuration.ReleaseReadOnly()

	if settings.LoginHeaderKey == "" {
		return false
	}

	value := r.Header.Get(settings.LoginHeaderKey)
	if value == "" {
		return false
	}
	if settings.LoginHeaderForceUsername {
		return strings.ToLower(value) == strings.ToLower(settings.AdminName)
	} else {
		return true
	}
}

// byDisabledLogin returns true if login has been disabled
func byDisabledLogin() bool {
	return configuration.IsLoginDisabled()
}

// byInternalSession returns true if the user holds a valid internal session cookie
func byInternalSession(w http.ResponseWriter, r *http.Request) bool {
	return sessionmanager.IsValidSession(w, r)
}

// IsCorrectUsernameAndPassword checks if a provided username and password is correct
func IsCorrectUsernameAndPassword(username, password string) bool {
	settings := configuration.GetServerSettingsReadOnly()
	configuration.ReleaseReadOnly()
	return strings.ToLower(username) == strings.ToLower(settings.AdminName) && configuration.HashPassword(password, false) == settings.AdminPassword
}
