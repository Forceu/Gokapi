package authentication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication/csrftoken"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	Init(modelUserPW)
	test.IsEqualInt(t, authSettings.Method, models.AuthenticationInternal)
	test.IsEqualString(t, authSettings.Username, "test")
}

func TestIsValid(t *testing.T) {
	config := models.AuthenticationConfig{
		Method:    models.AuthenticationInternal,
		SaltAdmin: "1234",
		SaltFiles: "1234",
		Username:  "2s",
	}
	err := checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.Username = "long name"
	err = checkAuthConfig(config)
	test.IsNil(t, err)

	config.Method = models.AuthenticationHeader
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.HeaderKey = "header"
	err = checkAuthConfig(config)
	test.IsNil(t, err)

	config.Method = models.AuthenticationOAuth2
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.OAuthProvider = "xxx"
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.OAuthClientId = "xxx"
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.OAuthClientSecret = "xxx"
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.OAuthRecheckInterval = -1
	err = checkAuthConfig(config)
	test.IsNotNil(t, err)
	config.OAuthRecheckInterval = 1
	err = checkAuthConfig(config)
	test.IsNil(t, err)
}

func TestIsCorrectUsernameAndPassword(t *testing.T) {
	user, ok, csfrOk := IsCorrectUsernameAndPassword("test", "adminadmin", csrftoken.Generate(csrftoken.TypeLogin))
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, csfrOk, true)

	user, ok, csfrOk = IsCorrectUsernameAndPassword("Test", "adminadmin", csrftoken.Generate(csrftoken.TypeLogin))
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, csfrOk, true)
	test.IsEqualInt(t, user.Id, 5)
	user, ok, csfrOk = IsCorrectUsernameAndPassword("user", "useruser", csrftoken.Generate(csrftoken.TypeLogin))
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, csfrOk, true)
	test.IsEqualInt(t, user.Id, 7)
	_, ok, csfrOk = IsCorrectUsernameAndPassword("test", "wrong", csrftoken.Generate(csrftoken.TypeLogin))
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, csfrOk, true)
	_, ok, csfrOk = IsCorrectUsernameAndPassword("invalid", "adminadmin", csrftoken.Generate(csrftoken.TypeLogin))
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, csfrOk, true)
	_, ok, csfrOk = IsCorrectUsernameAndPassword("test", "adminadmin", "invalidToken")
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, csfrOk, false)
}

func TestIsAuthenticated(t *testing.T) {
	testAuthSession(t)
	testAuthHeader(t)
	testAuthDisabled(t)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	authSettings.Method = -1
	_, ok, _ := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
}

func testAuthSession(t *testing.T) {

	exitCode := 0
	osExit = func(code int) {
		exitCode = code
	}

	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelUserPW)
	_, ok, _ := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	Init(modelOauth)
	_, ok, _ = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	Init(modelUserPW)
	w, r = test.GetRecorder("GET", "/", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	user, ok, _ := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 7)
	test.IsEqualInt(t, exitCode, 0)

	Init(models.AuthenticationConfig{
		Method: 10,
	})
	test.IsEqualInt(t, exitCode, 3)

}

func testAuthHeader(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelHeader)
	_, ok, err := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	test.IsNotNil(t, err)
	w, r = test.GetRecorder("GET", "/", nil, []test.Header{{
		Name:  "testHeader",
		Value: "testUser",
	}}, nil)

	user, ok, err := IsAuthenticated(w, r)
	test.IsEqualString(t, user.Name, "testuser")
	test.IsEqualBool(t, ok, true)
	test.IsNil(t, err)
	authSettings.OnlyRegisteredUsers = true
	w, r = test.GetRecorder("GET", "/", nil, []test.Header{{
		Name:  "testHeader",
		Value: "testUser",
	}}, nil)
	_, ok, err = IsAuthenticated(w, r)
	test.IsNil(t, err)
	test.IsEqualBool(t, ok, true)
	w, r = test.GetRecorder("GET", "/", nil, []test.Header{{
		Name:  "testHeader",
		Value: "otherUser2",
	}}, nil)
	_, ok, _ = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	authSettings.OnlyRegisteredUsers = false
}

func testAuthDisabled(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelDisabled)
	user, ok, _ := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 5)
}

func TestIsLogoutAvailable(t *testing.T) {
	authSettings.Method = models.AuthenticationInternal
	test.IsEqualBool(t, IsLogoutAvailable(), true)
	authSettings.Method = models.AuthenticationOAuth2
	test.IsEqualBool(t, IsLogoutAvailable(), true)
	authSettings.Method = models.AuthenticationHeader
	test.IsEqualBool(t, IsLogoutAvailable(), false)
	authSettings.Method = models.AuthenticationDisabled
	test.IsEqualBool(t, IsLogoutAvailable(), false)
}

func TestEqualString(t *testing.T) {
	test.IsEqualBool(t, IsEqualStringConstantTime("yes", "no"), false)
	test.IsEqualBool(t, IsEqualStringConstantTime("yes", "yes"), true)
}

func TestGetUserFromRequest(t *testing.T) {
	_, r := test.GetRecorder("GET", "/", nil, nil, nil)
	_, err := GetUserFromRequest(r)
	test.IsNotNil(t, err)
	c := context.WithValue(r.Context(), userNameContextKey, "invalid")
	rInvalid := r.WithContext(c)
	_, err = GetUserFromRequest(rInvalid)
	test.IsNotNil(t, err)

	user := models.User{
		Id:            1,
		Name:          "test",
		Permissions:   1,
		UserLevel:     2,
		LastOnline:    3,
		Password:      "12345",
		ResetPassword: true,
	}

	rValid := SetUserInRequest(r, user)
	retrievedUser, err := GetUserFromRequest(rValid)
	test.IsNil(t, err)
	test.IsEqual(t, retrievedUser, user)
}

func TestIsValidOauthUser(t *testing.T) {
	Init(modelOauth)
	info := OAuthUserInfo{Email: "", Subject: "randomid"}
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "newemail"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), true)
	test.IsEqualBool(t, isValidOauthUser(info, []string{"test2"}), true)
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), true)
	authSettings.OAuthGroupScope = "group"
	authSettings.OAuthGroups = []string{"othergroup"}
	info.Email = "test1"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "otheruser"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "test1"
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup"}), false)
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup", "othergroup"}), true)
	info.Email = "otheruser"
	test.IsEqualBool(t, isValidOauthUser(info, []string{"othergroup"}), true)
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup", "othergroup"}), true)
	info.Subject = ""
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup", "othergroup"}), false)
}

func TestWildcardMatch(t *testing.T) {
	type testPattern struct {
		Pattern string
		Input   string
		Result  bool
	}
	tests := []testPattern{{
		Pattern: "test",
		Input:   "test",
		Result:  true,
	}, {
		Pattern: "test*",
		Input:   "test",
		Result:  true,
	}, {
		Pattern: "*test",
		Input:   "test",
		Result:  true,
	}, {
		Pattern: "te*st",
		Input:   "test",
		Result:  true,
	}, {
		Pattern: "test*",
		Input:   "1test",
		Result:  false,
	}, {
		Pattern: "*test",
		Input:   "test1",
		Result:  false,
	}, {
		Pattern: "te*st",
		Input:   "teeeeeeeest",
		Result:  true,
	}, {
		Pattern: "te*st",
		Input:   "teast",
		Result:  true,
	}, {
		Pattern: "te*st",
		Input:   "te@st",
		Result:  true,
	}, {
		Pattern: "*@github.com",
		Input:   "email@github.com",
		Result:  true,
	}, {
		Pattern: "@github.com",
		Input:   "email@github.com",
		Result:  false,
	}, {
		Pattern: "@github.com",
		Input:   "email@gokapi.com",
		Result:  false,
	}, {
		Pattern: "*@github.com",
		Input:   "email@gokapi.com",
		Result:  false,
	}}
	for _, patternTest := range tests {
		fmt.Printf("Testing: %s == %s, expecting %v\n", patternTest.Pattern, patternTest.Input, patternTest.Result)
		result, err := matchesWithWildcard(patternTest.Pattern, patternTest.Input)
		test.IsNil(t, err)
		test.IsEqualBool(t, result, patternTest.Result)
	}
}

func getRecorder(cookies []test.Cookie) (*httptest.ResponseRecorder, *http.Request, bool, int) {
	w, r := test.GetRecorder("GET", "/", cookies, nil, nil)
	return w, r, false, 1
}

func TestLogout(t *testing.T) {
	Init(modelUserPW)
	w, r, _, _ := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "logoutsession"},
	})
	_, ok := sessionmanager.IsValidSession(w, r, false, 0)
	test.IsEqualBool(t, ok, true)
	Logout(w, r)
	_, ok = database.GetSession("logoutsession")
	test.IsEqualBool(t, ok, false)
	_, ok = sessionmanager.IsValidSession(w, r, false, 0)
	test.IsEqualBool(t, ok, false)
	test.ResponseIsRedirect(t, w, "login", false)

	Init(modelOauth)
	w, r, _, _ = getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "logoutsession2"},
	})
	_, ok = sessionmanager.IsValidSession(w, r, false, 0)
	test.IsEqualBool(t, ok, true)
	Logout(w, r)
	_, ok = database.GetSession("logoutsession")
	test.IsEqualBool(t, ok, false)
	_, ok = sessionmanager.IsValidSession(w, r, false, 0)
	test.IsEqualBool(t, ok, false)
	test.ResponseIsRedirect(t, w, "login?consent=true", false)
}

type testInfo struct {
	Output []byte
}

func (t testInfo) Claims(v interface{}) error {
	if t.Output == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(t.Output, v)
}
func getOauthUserOutput(t *testing.T, info OAuthUserInfo) (*httptest.ResponseRecorder, error) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	err := CheckOauthUserAndRedirect(w, r, info)
	if err != nil {
		return w, err
	}
	return w, nil
}
func TestCheckOauthUser(t *testing.T) {
	Init(modelOauth)
	info := OAuthUserInfo{
		ClaimsSent: testInfo{Output: []byte(`{"amr":["pwd","hwk","user","pin","mfa"],"aud":["gokapi-dev"],"auth_time":1705573822,"azp":"gokapi-dev","client_id":"gokapi-dev","email":"test@test.com","email_verified":true,"groups":["admins","dev"],"iat":1705577400,"iss":"https://auth.test.com","name":"gokapi","preferred_username":"gokapi","rat":1705577400,"sub":"944444cf3e-0546-44f2-acfa-a94444444360"}`)},
	}
	w, err := getOauthUserOutput(t, info)
	test.IsNil(t, err)
	test.ResponseIsRedirect(t, w, "error", true)

	info.Subject = "random"
	w, err = getOauthUserOutput(t, info)
	test.IsNil(t, err)
	test.ResponseIsRedirect(t, w, "error", true)

	info.Email = "random"
	w, err = getOauthUserOutput(t, info)
	test.IsNil(t, err)
	test.ResponseIsRedirect(t, w, "admin", false)

	info.Email = "test@test-invalid.com"
	authSettings.OnlyRegisteredUsers = true
	w, err = getOauthUserOutput(t, info)
	test.IsNil(t, err)
	test.ResponseIsRedirect(t, w, "error", true)

	info.Email = "random"
	w, err = getOauthUserOutput(t, info)
	test.IsNil(t, err)
	test.ResponseIsRedirect(t, w, "admin", false)

	authSettings.OnlyRegisteredUsers = false
	authSettings.OAuthGroups = []string{"otheruser@test"}
	authSettings.OAuthGroupScope = "groupscope"
	newClaims := testInfo{Output: []byte("{invalid")}
	info.ClaimsSent = newClaims
	_, err = getOauthUserOutput(t, info)
	test.IsNotNil(t, err)
}

var modelUserPW = models.AuthenticationConfig{
	Method:    models.AuthenticationInternal,
	SaltAdmin: testconfiguration.SaltAdmin,
	SaltFiles: "1234",
	Username:  "test",
}
var modelOauth = models.AuthenticationConfig{
	Method:               models.AuthenticationOAuth2,
	SaltAdmin:            testconfiguration.SaltAdmin,
	SaltFiles:            "1234",
	OAuthProvider:        "test",
	OAuthClientId:        "test",
	OAuthClientSecret:    "test",
	OAuthRecheckInterval: 1,
}
var modelHeader = models.AuthenticationConfig{
	Method:    models.AuthenticationHeader,
	SaltAdmin: testconfiguration.SaltAdmin,
	SaltFiles: "1234",
	HeaderKey: "testHeader",
}
var modelDisabled = models.AuthenticationConfig{
	Method:    models.AuthenticationDisabled,
	SaltAdmin: testconfiguration.SaltAdmin,
	SaltFiles: "1234",
}
