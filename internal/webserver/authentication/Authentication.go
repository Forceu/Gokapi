package authentication

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// CookieOauth is the cookie name used for login
const CookieOauth = "state"

// Internal authentication method uses a user / password combination handled by Gokapi
const Internal = 0

// OAuth2 authentication retrieves the users email with Open Connect ID
const OAuth2 = 1

// Header authentication relies on a header from a reverse proxy to parse the username
const Header = 2

// Disabled authentication ignores all internal authentication procedures. A reverse proxy needs to restrict access
const Disabled = 3

var authSettings models.AuthenticationConfig

// Init needs to be called first to process the authentication configuration
func Init(config models.AuthenticationConfig) {
	valid, err := isValid(config)
	if !valid {
		log.Println("Error while initiating authentication method:")
		log.Fatal(err)
	}
	authSettings = config
}

// isValid checks if the config is actually valid, and returns true or returns false and an error
func isValid(config models.AuthenticationConfig) (bool, error) {
	switch config.Method {
	case Internal:
		if len(config.Username) < 3 {
			return false, errors.New("username too short")
		}
		if len(config.Password) != 40 {
			return false, errors.New("password does not appear to be a SHA-1 hash")
		}
		return true, nil
	case OAuth2:
		if config.OAuthProvider == "" {
			return false, errors.New("oauth provider was not set")
		}
		if config.OAuthClientId == "" {
			return false, errors.New("oauth client id was not set")
		}
		if config.OAuthClientSecret == "" {
			return false, errors.New("oauth client secret was not set")
		}
		return true, nil
	case Header:
		if config.HeaderKey == "" {
			return false, errors.New("header key is not set")
		}
		return true, nil
	case Disabled:
		return true, nil
	default:
		return false, errors.New("unknown authentication selected")
	}
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

func isUserInArray(userEntered string, allowedUsers []string) bool {
	for _, allowedUser := range allowedUsers {
		matches, err := matchesWithWildcard(strings.ToLower(allowedUser), strings.ToLower(userEntered))
		helper.Check(err)
		if matches {
			return true
		}
	}
	return false
}

func matchesWithWildcard(pattern, input string) (bool, error) {
	components := strings.Split(pattern, "*")
	if len(components) == 1 {
		// if len is 1, there are no *'s, return exact match pattern
		return regexp.MatchString("^"+pattern+"$", input)
	}
	var result strings.Builder
	for i, literal := range components {
		// Replace * with .*
		if i > 0 {
			result.WriteString(".*")
		}
		// Quote any regular expression meta characters in the
		// literal text.
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return regexp.MatchString("^"+result.String()+"$", input)
}

func isGroupInArray(userGroups []string, allowedGroups []string) bool {
	for _, group := range userGroups {
		for _, allowedGroup := range allowedGroups {
			matches, err := matchesWithWildcard(strings.ToLower(allowedGroup), strings.ToLower(group))
			helper.Check(err)
			if matches {
				return true
			}
		}
	}
	return false
}

func extractOauthGroups(userInfo OAuthUserClaims, groupScope string) ([]string, error) {
	var claims json.RawMessage
	var data map[string]interface{}

	err := userInfo.Claims(&claims)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(claims, &data)
	if err != nil {
		return nil, err
	}

	// Extract the "groups" field
	groupsInterface, ok := data[groupScope]
	if !ok {
		return nil, fmt.Errorf("claim %s was not passed on", groupScope)
	}

	// Convert the interface{} to a []interface{} and then to []string
	var groups []string
	for _, group := range groupsInterface.([]interface{}) {
		groups = append(groups, group.(string))
	}

	return groups, nil
}

func extractFieldValue(userInfo OAuthUserClaims, fieldName string) (string, error) {
	var claims json.RawMessage

	err := userInfo.Claims(&claims)
	if err != nil {
		return "", err
	}
	var fieldMap map[string]interface{}
	err = json.Unmarshal(claims, &fieldMap)
	if err != nil {
		return "", err
	}

	// Extract the field value based on the provided fieldName
	fieldValue, ok := fieldMap[fieldName]
	if !ok {
		return "", fmt.Errorf("%s scope not found in reply", fieldName)
	}

	strValue, ok := fieldValue.(string)
	if !ok {
		return "", fmt.Errorf("value of %s scope is not a string", fieldName)
	}

	return strValue, nil
}

// OAuthUserInfo is used to make testing easier. This results in an additional parameter for the subject unfortunately
type OAuthUserInfo struct {
	Subject    string
	Email      string
	ClaimsSent OAuthUserClaims
}

// OAuthUserClaims contains the claims
type OAuthUserClaims interface {
	Claims(v interface{}) error
}

// CheckOauthUserAndRedirect checks if the user is allowed to use the Gokapi instance
func CheckOauthUserAndRedirect(userInfo OAuthUserInfo, w http.ResponseWriter) error {
	var username string
	var groups []string
	var err error

	if authSettings.OAuthUserScope != "" {
		if authSettings.OAuthUserScope == "email" {
			username = userInfo.Email
		} else {
			username, err = extractFieldValue(userInfo.ClaimsSent, authSettings.OAuthUserScope)
			if err != nil {
				return err
			}
		}
	}
	if authSettings.OAuthGroupScope != "" {
		groups, err = extractOauthGroups(userInfo.ClaimsSent, authSettings.OAuthGroupScope)
		if err != nil {
			return err
		}
	}
	if isValidOauthUser(userInfo, username, groups) {
		sessionmanager.CreateSession(w, authSettings.Method == OAuth2, authSettings.OAuthRecheckInterval)
		redirect(w, "admin")
		return nil
	}
	redirect(w, "error-auth")
	return nil
}

func isValidOauthUser(userInfo OAuthUserInfo, username string, groups []string) bool {
	if userInfo.Subject == "" {
		return false
	}
	isValidUser := true
	if len(authSettings.OAuthUsers) > 0 {
		isValidUser = isUserInArray(username, authSettings.OAuthUsers)
	}
	isValidGroup := true
	if len(authSettings.OAuthGroups) > 0 {
		isValidGroup = isGroupInArray(groups, authSettings.OAuthGroups)
	}
	return isValidUser && isValidGroup
}

// isGrantedSession returns true if the user holds a valid internal session cookie
func isGrantedSession(w http.ResponseWriter, r *http.Request) bool {
	return sessionmanager.IsValidSession(w, r, authSettings.Method == OAuth2, authSettings.OAuthRecheckInterval)
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
	if authSettings.Method == OAuth2 {
		redirect(w, "login?consent=true")
	} else {
		redirect(w, "login")
	}
}

// IsLogoutAvailable returns true if a logout button should be shown with the current form of authentication
func IsLogoutAvailable() bool {
	return authSettings.Method == Internal || authSettings.Method == OAuth2
}
