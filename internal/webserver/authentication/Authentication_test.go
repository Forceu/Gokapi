package authentication

import (
	"encoding/json"
	"errors"
	"github.com/coreos/go-oidc/v3/oidc"
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
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	Init(modelUserPW)
	test.IsEqualInt(t, authSettings.Method, Internal)
	test.IsEqualString(t, authSettings.Username, "admin")
}

func TestIsCorrectUsernameAndPassword(t *testing.T) {
	test.IsEqualBool(t, IsCorrectUsernameAndPassword("admin", "adminadmin"), true)
	test.IsEqualBool(t, IsCorrectUsernameAndPassword("admin", "wrong"), false)
}

func TestIsAuthenticated(t *testing.T) {
	testAuthSession(t)
	testAuthHeader(t)
	testAuthDisabled(t)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	authSettings.Method = -1
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
}

func testAuthSession(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelUserPW)
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
	Init(modelOauth)
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
	Init(modelUserPW)
	w, r = test.GetRecorder("GET", "/", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	test.IsEqualBool(t, IsAuthenticated(w, r), true)
}

func testAuthHeader(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelHeader)
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
	w, r = test.GetRecorder("GET", "/", nil, []test.Header{{
		Name:  "testHeader",
		Value: "testUser",
	}}, nil)
	test.IsEqualBool(t, IsAuthenticated(w, r), true)
	authSettings.HeaderUsers = []string{"testUser"}
	test.IsEqualBool(t, IsAuthenticated(w, r), true)
	authSettings.HeaderUsers = []string{"otherUser"}
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
	authSettings.HeaderKey = ""
	authSettings.HeaderUsers = []string{}
	test.IsEqualBool(t, IsAuthenticated(w, r), false)
}

func testAuthDisabled(t *testing.T) {
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Init(modelDisabled)
	test.IsEqualBool(t, IsAuthenticated(w, r), true)
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
	info := oidc.UserInfo{Subject: "randomid"}
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "", []string{}), true)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "test1", []string{"test2"}), true)
	authSettings.OAuthUserScope = "user"
	authSettings.OAuthUsers = []string{"otheruser"}
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "test1", []string{}), false)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "otheruser", []string{}), true)
	authSettings.OAuthGroupScope = "group"
	authSettings.OAuthGroups = []string{"othergroup"}
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "test1", []string{}), false)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "otheruser", []string{}), false)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "test1", []string{"testgroup"}), false)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "test1", []string{"testgroup", "othergroup"}), false)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "otheruser", []string{"othergroup"}), true)
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "otheruser", []string{"testgroup", "othergroup"}), true)
	info.Subject = ""
	test.IsEqualBool(t, isValidOauthUser(info.Subject, "otheruser", []string{"testgroup", "othergroup"}), false)
}

func TestLogout(t *testing.T) {
	Init(modelUserPW)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Logout(w, r)
}

type testInfo struct {
	Output  []byte
	Subject string
}

func (t *testInfo) Claims(v interface{}) error {
	if t.Output == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(t.Output, v)
}

func TestCheckOauthUser(t *testing.T) {
	Init(modelOauth)
	info := testInfo{Output: []byte(`{"amr":["pwd","hwk","user","pin","mfa"],"aud":["gokapi-dev"],"auth_time":1705573822,"azp":"gokapi-dev","client_id":"gokapi-dev","email":"test@test.com","email_verified":true,"groups":["admins","dev"],"iat":1705577400,"iss":"https://auth.test.com","name":"gokapi","preferred_username":"gokapi","rat":1705577400,"sub":"944444cf3e-0546-44f2-acfa-a94444444360"}`)}
	output, err := getOuthUserOutput(t, &info, info.Subject)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")
	info.Subject = "random"
	output, err = getOuthUserOutput(t, &info, info.Subject)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "admin")
	authSettings.OAuthUserScope = "email"
	authSettings.OAuthUsers = []string{"otheruser"}
	output, err = getOuthUserOutput(t, &info, info.Subject)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")
	authSettings.OAuthUsers = []string{"test@test.com"}
	output, err = getOuthUserOutput(t, &info, info.Subject)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "admin")
	authSettings.OAuthUsers = []string{"otheruser@test"}
	output, err = getOuthUserOutput(t, &info, info.Subject)
	test.IsNil(t, err)
	test.IsEqualString(t, redirectsToSite(output), "error-auth")
	authSettings.OAuthUserScope = "invalidScope"
	output, err = getOuthUserOutput(t, &info, info.Subject)
	test.IsNotNil(t, err)
	info.Output = []byte("{invalid")
	output, err = getOuthUserOutput(t, &info, info.Subject)
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

func getOuthUserOutput(t *testing.T, info OAuthUserInfo, infoSubject string) (string, error) {
	t.Helper()
	w := httptest.NewRecorder()
	err := CheckOauthUserAndRedirect(info, infoSubject, w)
	if err != nil {
		return "", err
	}
	output, _ := io.ReadAll(w.Result().Body)
	return string(output), nil
}

var modelUserPW = models.AuthenticationConfig{
	Method:            Internal,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "admin",
	Password:          "7d23732d69c050bf7a2f5ab7d979f92f33bb585e",
	HeaderKey:         "",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthUserScope:    "",
	OAuthGroupScope:   "",
}
var modelOauth = models.AuthenticationConfig{
	Method:            OAuth2,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "",
	OAuthProvider:     "test",
	OAuthClientId:     "test",
	OAuthClientSecret: "test",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthUserScope:    "",
	OAuthGroupScope:   "",
}
var modelHeader = models.AuthenticationConfig{
	Method:            Header,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "testHeader",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthUserScope:    "",
	OAuthGroupScope:   "",
}
var modelDisabled = models.AuthenticationConfig{
	Method:            Disabled,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "",
	OAuthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OAuthUsers:        nil,
	OAuthGroups:       nil,
	OAuthUserScope:    "",
	OAuthGroupScope:   "",
}
