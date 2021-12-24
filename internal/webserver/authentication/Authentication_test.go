package authentication

import (
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"testing"
)

var modelUserPW = models.AuthenticationConfig{
	Method:            Internal,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "admin",
	Password:          "7d23732d69c050bf7a2f5ab7d979f92f33bb585e",
	HeaderKey:         "",
	OauthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OauthUsers:        nil,
}

func TestInit(t *testing.T) {
	Init(modelUserPW)
	test.IsEqualInt(t, authSettings.Method, Internal)
	test.IsEqualString(t, authSettings.Username, "admin")
}

func TestIsLogoutAvailable(t *testing.T) {
	authSettings.Method = Internal
	test.IsEqualBool(t, IsLogoutAvailable(), true)
	authSettings.Method = OAuth2
	test.IsEqualBool(t, IsLogoutAvailable(), true)
	authSettings.Method = Header
	test.IsEqualBool(t, IsLogoutAvailable(), false)
	authSettings.Method = Disabled
	test.IsEqualBool(t, IsLogoutAvailable(), false)
}

func TestEqualString(t *testing.T) {
	test.IsEqualBool(t, isEqualStringConstantTime("yes", "no"), false)
	test.IsEqualBool(t, isEqualStringConstantTime("yes", "yes"), true)
}

func TestIsCorrectUsernameAndPassword(t *testing.T) {
	test.IsEqualBool(t, IsCorrectUsernameAndPassword("admin", "adminadmin"), true)
	test.IsEqualBool(t, IsCorrectUsernameAndPassword("admin", "wrong"), false)
}
