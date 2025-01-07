package authentication

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
	test.IsEqualString(t, authSettings.Username, "admin")
}

func TestIsCorrectUsernameAndPassword(t *testing.T) {
	user, ok := IsCorrectUsernameAndPassword("admin", "adminadmin")
	test.IsEqualBool(t, ok, true)
	user, ok = IsCorrectUsernameAndPassword("Admin", "adminadmin")
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 5)
	user, ok = IsCorrectUsernameAndPassword("user", "useruser")
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 7)
	_, ok = IsCorrectUsernameAndPassword("admin", "wrong")
	test.IsEqualBool(t, ok, false)
}

func TestIsAuthenticated(t *testing.T) {
	testAuthSession(t)
	testAuthHeader(t)
	testAuthDisabled(t)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	authSettings.Method = -1
	_, ok := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
}

func testAuthSession(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelUserPW)
	_, ok := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	Init(modelOauth)
	_, ok = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	Init(modelUserPW)
	w, r = test.GetRecorder("GET", "/", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	user, ok := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 7)
}

func testAuthHeader(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelHeader)
	_, ok := IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	w, r = test.GetRecorder("GET", "/", nil, []test.Header{{
		Name:  "testHeader",
		Value: "testUser",
	}}, nil)

	user, ok := IsAuthenticated(w, r)
	test.IsEqualString(t, user.Name, "testuser")
	test.IsEqualBool(t, ok, true)
	authSettings.HeaderUsers = []string{"testUser"}
	_, ok = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, true)
	authSettings.HeaderUsers = []string{"otherUser"}
	_, ok = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
	authSettings.HeaderKey = ""
	authSettings.HeaderUsers = []string{}
	_, ok = IsAuthenticated(w, r)
	test.IsEqualBool(t, ok, false)
}

func testAuthDisabled(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelDisabled)
	user, ok := IsAuthenticated(w, r)
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

func TestRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	redirect(w, "test")
	output, err := io.ReadAll(w.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(output), "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./test\"></head></html>")
}

func TestIsValidOauthUser(t *testing.T) {
	Init(modelOauth)
	info := OAuthUserInfo{Email: "", Subject: "randomid"}
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "newemail"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), true)
	test.IsEqualBool(t, isValidOauthUser(info, []string{"test2"}), true)
	authSettings.OAuthUsers = []string{"otheruser"}
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "otheruser"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), true)
	authSettings.OAuthGroupScope = "group"
	authSettings.OAuthGroups = []string{"othergroup"}
	info.Email = "test1"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "otheruser"
	test.IsEqualBool(t, isValidOauthUser(info, []string{}), false)
	info.Email = "test1"
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup"}), false)
	test.IsEqualBool(t, isValidOauthUser(info, []string{"testgroup", "othergroup"}), false)
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

func TestLogout(t *testing.T) {
	Init(modelUserPW)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Logout(w, r)
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

func TestCheckOauthUser(t *testing.T) {
	Init(modelOauth)
	info := OAuthUserInfo{
		ClaimsSent: testInfo{Output: []byte(`{"amr":["pwd","hwk","user","pin","mfa"],"aud":["gokapi-dev"],"auth_time":1705573822,"azp":"gokapi-dev","client_id":"gokapi-dev","email":"test@test.com","email_verified":true,"groups":["admins","dev"],"iat":1705577400,"iss":"https://auth.test.com","name":"gokapi","preferred_username":"gokapi","rat":1705577400,"sub":"944444cf3e-0546-44f2-acfa-a94444444360"}`)},
	}
	output, err := getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")

	info.Subject = "random"
	output, err = getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")

	info.Email = "random"
	output, err = getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "admin")

	info.Email = "test@test.com"
	authSettings.OAuthUsers = []string{"otheruser"}
	output, err = getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")

	authSettings.OAuthUsers = []string{"test@test.com"}
	output, err = getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "admin")

	authSettings.OAuthUsers = []string{"otheruser@test"}
	output, err = getOuthUserOutput(t, info)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")

	authSettings.OAuthGroups = []string{"otheruser@test"}
	authSettings.OAuthGroupScope = "groupscope"
	newClaims := testInfo{Output: []byte("{invalid")}
	info.ClaimsSent = newClaims
	_, err = getOuthUserOutput(t, info)
	test.IsNotNil(t, err)
}

func redirectsToSite(input string) string {
	sites := []string{"admin", "error-auth"}
	for _, site := range sites {
		if strings.Contains(input, site) {
			return site
		}
	}
	return "other"
}

func getOuthUserOutput(t *testing.T, info OAuthUserInfo) (string, error) {
	t.Helper()
	w := httptest.NewRecorder()
	err := CheckOauthUserAndRedirect(info, w)
	if err != nil {
		return "", err
	}
	output, _ := io.ReadAll(w.Result().Body)
	return string(output), nil
}

var modelUserPW = models.AuthenticationConfig{
	Method:            models.AuthenticationInternal,
	SaltAdmin:         testconfiguration.SaltAdmin,
	SaltFiles:         "1234",
	Username:          "admin",
	HeaderKey:         "",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthGroupScope:   "",
}
var modelOauth = models.AuthenticationConfig{
	Method:            models.AuthenticationOAuth2,
	SaltAdmin:         testconfiguration.SaltAdmin,
	SaltFiles:         "1234",
	Username:          "",
	HeaderKey:         "",
	OAuthProvider:     "test",
	OAuthClientId:     "test",
	OAuthClientSecret: "test",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthGroupScope:   "",
}
var modelHeader = models.AuthenticationConfig{
	Method:            models.AuthenticationHeader,
	SaltAdmin:         testconfiguration.SaltAdmin,
	SaltFiles:         "1234",
	Username:          "",
	HeaderKey:         "testHeader",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthGroupScope:   "",
}
var modelDisabled = models.AuthenticationConfig{
	Method:            models.AuthenticationDisabled,
	SaltAdmin:         testconfiguration.SaltAdmin,
	SaltFiles:         "1234",
	Username:          "",
	HeaderKey:         "",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthGroupScope:   "",
}
