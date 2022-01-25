package authentication

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/coreos/go-oidc/v3/oidc"
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

func TestGetMethod(t *testing.T) {
	test.IsEqualInt(t, GetMethod(), Disabled)
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
	test.IsEqualBool(t, isValidOauthUser(""), false)
	test.IsEqualBool(t, isValidOauthUser("user"), true)
	authSettings.OauthUsers = []string{"otheruser"}
	test.IsEqualBool(t, isValidOauthUser("user"), false)
}

func TestLogout(t *testing.T) {
	Init(modelUserPW)
	w, r := test.GetRecorder("GET", "/", nil, nil, nil)
	Logout(w, r)
}

func TestCheckOauthUser(t *testing.T) {
	info := oidc.UserInfo{}
	test.IsEqualBool(t, strings.Contains(getOuthUserOutput(&info), "error-auth"), true)
	info.Email = "test@test"
	test.IsEqualBool(t, strings.Contains(getOuthUserOutput(&info), "admin"), true)
	authSettings.OauthUsers = []string{"test@test"}
	test.IsEqualBool(t, strings.Contains(getOuthUserOutput(&info), "admin"), true)
	authSettings.OauthUsers = []string{"otheruser@test"}
	test.IsEqualBool(t, strings.Contains(getOuthUserOutput(&info), "error-auth"), true)
}

func getOuthUserOutput(info *oidc.UserInfo) string {
	w := httptest.NewRecorder()
	CheckOauthUser(info, w)
	output, _ := io.ReadAll(w.Result().Body)
	return string(output)
}

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
var modelOauth = models.AuthenticationConfig{
	Method:            OAuth2,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "",
	OauthProvider:     "test",
	OAuthClientId:     "test",
	OAuthClientSecret: "test",
	HeaderUsers:       nil,
	OauthUsers:        nil,
}
var modelHeader = models.AuthenticationConfig{
	Method:            Header,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "testHeader",
	OauthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OauthUsers:        nil,
}
var modelDisabled = models.AuthenticationConfig{
	Method:            Disabled,
	SaltAdmin:         "1234",
	SaltFiles:         "1234",
	Username:          "",
	Password:          "",
	HeaderKey:         "",
	OauthProvider:     "",
	OAuthClientId:     "",
	OAuthClientSecret: "",
	HeaderUsers:       nil,
	OauthUsers:        nil,
}
