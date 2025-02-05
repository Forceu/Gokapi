package authentication

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// CookieOauth is the cookie name used for login
const CookieOauth = "state"

var authSettings models.AuthenticationConfig

// Init needs to be called first to process the authentication configuration
func Init(config models.AuthenticationConfig) {
	err := checkAuthConfig(config)
	if err != nil {
		log.Println("Error while initiating authentication method:")
		log.Println(err)
		osExit(3)
		return
	}
	authSettings = config
}

var osExit = os.Exit

// checkAuthConfig checks if the config is actually valid, and returns an error otherwise
func checkAuthConfig(config models.AuthenticationConfig) error {
	switch config.Method {
	case models.AuthenticationInternal:
		if len(config.Username) < 3 {
			return errors.New("username too short")
		}
		return nil
	case models.AuthenticationOAuth2:
		if config.OAuthProvider == "" {
			return errors.New("oauth provider was not set")
		}
		if config.OAuthClientId == "" {
			return errors.New("oauth client id was not set")
		}
		if config.OAuthClientSecret == "" {
			return errors.New("oauth client secret was not set")
		}
		if config.OAuthRecheckInterval < 1 {
			return errors.New("oauth recheck interval invalid")
		}
		return nil
	case models.AuthenticationHeader:
		if config.HeaderKey == "" {
			return errors.New("header key is not set")
		}
		return nil
	case models.AuthenticationDisabled:
		return nil
	default:
		return errors.New("unknown authentication selected")
	}
}

func GetUserFromRequest(r *http.Request) (models.User, error) {
	c := r.Context()
	user, ok := c.Value("user").(models.User)
	if !ok {
		return models.User{}, errors.New("user not found in context")
	}
	return user, nil
}

// IsAuthenticated returns true and the user ID if authenticated
func IsAuthenticated(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	switch authSettings.Method {
	case models.AuthenticationInternal:
		user, ok := isGrantedSession(w, r)
		if ok {
			return user, true
		}
	case models.AuthenticationOAuth2:
		user, ok := isGrantedSession(w, r)
		if ok {
			return user, true
		}
	case models.AuthenticationHeader:
		user, ok := isGrantedHeader(r)
		if ok {
			return user, true
		}
	case models.AuthenticationDisabled:
		adminUser, ok := database.GetSuperAdmin()
		if !ok {
			panic("no super admin found")
		}
		return adminUser, true
	}
	return models.User{}, false
}

// isGrantedHeader returns true if the user was authenticated by a proxy header if enabled
func isGrantedHeader(r *http.Request) (models.User, bool) {
	if authSettings.HeaderKey == "" {
		return models.User{}, false
	}
	userName := r.Header.Get(authSettings.HeaderKey)
	if userName == "" {
		return models.User{}, false
	}
	return getOrCreateUser(userName)
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

	// Convert the interface{} to a []string
	if groupsInterface == nil {
		return []string{}, nil
	}
	groupsCast, ok := groupsInterface.([]any)
	if !ok {
		return nil, fmt.Errorf("scope %s is not an array", groupScope)
	}
	var groups []string
	for _, group := range groupsCast {
		groups = append(groups, group.(string))
	}

	return groups, nil
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
	var groups []string
	var err error

	if authSettings.OAuthGroupScope != "" {
		groups, err = extractOauthGroups(userInfo.ClaimsSent, authSettings.OAuthGroupScope)
		if err != nil {
			return err
		}
	}
	if isValidOauthUser(userInfo, groups) {
		user, ok := getOrCreateUser(userInfo.Email)
		if ok {
			sessionmanager.CreateSession(w, true, authSettings.OAuthRecheckInterval, user.Id)
			redirect(w, "admin")
			return nil
		}
	}
	redirect(w, "error-auth")
	return nil
}

func getOrCreateUser(username string) (models.User, bool) {
	user, ok := database.GetUserByName(username)
	if !ok {
		if authSettings.OnlyRegisteredUsers {
			return models.User{}, false
		}
		user = models.User{
			Name:      username,
			UserLevel: models.UserLevelUser,
		}
		database.SaveUser(user, true)
		user, ok = database.GetUserByName(username)
		if !ok {
			panic("unable to read new user")
		}
	}
	return user, true
}

func isValidOauthUser(userInfo OAuthUserInfo, groups []string) bool {
	if userInfo.Subject == "" {
		return false
	}
	if userInfo.Email == "" {
		return false
	}
	isValidGroup := true
	if len(authSettings.OAuthGroups) > 0 {
		isValidGroup = isGroupInArray(groups, authSettings.OAuthGroups)
	}
	return isValidGroup
}

// isGrantedSession returns true if the user holds a valid internal session cookie
func isGrantedSession(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	return sessionmanager.IsValidSession(w, r, authSettings.Method == models.AuthenticationOAuth2, authSettings.OAuthRecheckInterval)
}

// IsCorrectUsernameAndPassword checks if a provided username and password is correct
func IsCorrectUsernameAndPassword(username, password string) (models.User, bool) {
	user, ok := database.GetUserByName(username)
	if !ok {
		return models.User{}, false
	}
	if IsEqualStringConstantTime(configuration.HashPasswordCustomSalt(password, authSettings.SaltAdmin), user.Password) {
		return user, true
	}
	return models.User{}, false
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
	if authSettings.Method == models.AuthenticationInternal || authSettings.Method == models.AuthenticationOAuth2 {
		sessionmanager.LogoutSession(w, r)
	}
	if authSettings.Method == models.AuthenticationOAuth2 {
		redirect(w, "login?consent=true")
	} else {
		redirect(w, "login")
	}
}

// IsLogoutAvailable returns true if a logout button should be shown with the current form of authentication
func IsLogoutAvailable() bool {
	return authSettings.Method == models.AuthenticationInternal || authSettings.Method == models.AuthenticationOAuth2
}
