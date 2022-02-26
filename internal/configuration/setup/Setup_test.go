package setup

import (
	"bytes"
	"context"
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var jsonForms []jsonFormObject

func TestMain(m *testing.M) {
	testconfiguration.SetDirEnv()
	testconfiguration.Delete()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestInputToJson(t *testing.T) {
	var err error
	_, r := test.GetRecorder("POST", "/setupResult", nil, nil, bytes.NewBufferString("invalid"))
	jsonForms, err = inputToJsonForm(r)
	test.IsNotNil(t, err)
	buf := bytes.NewBufferString(testInputInternalAuth)
	_, r = test.GetRecorder("POST", "/setupResult", nil, nil, buf)
	jsonForms, err = inputToJsonForm(r)
	test.IsNil(t, err)
	for _, item := range jsonForms {
		if item.Name == "auth_username" {
			test.IsEqualString(t, item.Value, "admin")
		}
	}
}

var config = models.Configuration{
	Authentication: models.AuthenticationConfig{},
	Port:           "",
	ServerUrl:      "",
	RedirectUrl:    "",
	ConfigVersion:  0,
	LengthId:       0,
	DataDir:        "",
	MaxMemory:      0,
	UseSsl:         false,
	MaxFileSizeMB:  0,
}

func TestToConfiguration(t *testing.T) {
	output, cloudConfig, err := toConfiguration(&jsonForms)
	test.IsNil(t, err)
	test.IsEqualInt(t, output.Authentication.Method, authentication.Internal)
	test.IsEqualString(t, cloudConfig.Aws.KeyId, "testapi")
	test.IsEqualString(t, output.Authentication.Username, "admin")
	test.IsNotEqualString(t, output.Authentication.Password, "adminadmin")
	test.IsNotEqualString(t, output.Authentication.Password, "")
	test.IsEqualString(t, output.RedirectUrl, "https://github.com/Forceu/Gokapi/")
}

func TestVerifyPortNumber(t *testing.T) {
	test.IsEqualInt(t, verifyPortNumber(2134), 2134)
	test.IsEqualInt(t, verifyPortNumber(-1), environment.DefaultPort)
	test.IsEqualInt(t, verifyPortNumber(666666), environment.DefaultPort)
}

func TestAddTrailingSlash(t *testing.T) {
	test.IsEqualString(t, addTrailingSlash("test"), "test/")
	test.IsEqualString(t, addTrailingSlash("test2/"), "test2/")
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	outputError(w, errors.New("test error"))
	test.IsEqualInt(t, w.Result().StatusCode, 500)
	test.ResponseBodyContains(t, w, "test error")
}

func TestBasicAuth(t *testing.T) {
	isAuth := false
	continueFunc := func(w http.ResponseWriter, r *http.Request) {
		isAuth = true
	}

	w, r := test.GetRecorder("GET", "/setup", nil, nil, nil)
	isInitialSetup = true
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, true)

	isAuth = false
	isInitialSetup = false
	username = "test"
	password = "testpw"
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, false)

	r.Header.Add("Authorization", "Basic dGVzdDp0ZXN0cHc=")
	isAuth = false
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, true)

	r.Header.Set("Authorization", "Basic dGVzdDppbnZhbGlk")
	isAuth = false
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, false)

	r.Header.Set("Authorization", "Basic aW52YWxpZDppbnZhbGlk")
	isAuth = false
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, false)

	r.Header.Set("Authorization", "Basic aW52YWxpZDp0ZXN0cHc=")
	isAuth = false
	basicAuth(continueFunc).ServeHTTP(w, r)
	test.IsEqualBool(t, isAuth, false)
}

func TestInitialSetup(t *testing.T) {
	testconfiguration.Create(false)
	test.CompletesWithinTime(t, RunIfFirstStart, 3*time.Second)
	testconfiguration.Delete()
	go func() {
		time.Sleep(1 * time.Second)
		srv.Shutdown(context.Background())
	}()
	RunIfFirstStart()
	test.IsEqualBool(t, isInitialSetup, true)
}

func TestRunConfigModification(t *testing.T) {
	testconfiguration.Create(false)
	username = ""
	password = ""
	finish := make(chan bool)
	go func() {
		for !serverStarted {
			time.Sleep(100 * time.Millisecond)
		}
		test.HttpPageResult(t, test.HttpTestConfig{
			Url:             "http://localhost:53842/setup/start",
			IsHtml:          false,
			ExcludedContent: []string{"Gokapi Setup"},
			Method:          "GET",
			ResultCode:      401,
		})
		time.Sleep(1 * time.Second)
		srv.Shutdown(context.Background())
		finish <- true
	}()
	RunConfigModification()
	test.IsEqualInt(t, len(username), 6)
	test.IsEqualInt(t, len(password), 10)
	isInitialSetup = true
	<-finish
}

func TestIntegration(t *testing.T) {
	testconfiguration.Delete()
	test.FileDoesNotExist(t, "test/config.json")
	go RunIfFirstStart()
	for !serverStarted {
		time.Sleep(100 * time.Millisecond)
	}

	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/admin",
		IsHtml:          false,
		RequiredContent: []string{"Server is in maintenance mode"},
		ExcludedContent: []string{"Downloads"},
		Method:          "GET",
		ResultCode:      200,
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/start",
		IsHtml:          false,
		RequiredContent: []string{"Thank you for choosing Gokapi"},
		Method:          "GET",
		ResultCode:      200,
	})
	test.HttpPostRequest(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		ExcludedContent: []string{"\"result\": \"OK\""},
		RequiredContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		PostValues:      nil,
		ResultCode:      500,
	})

	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      200,
		Body:            strings.NewReader(testInputInternalAuth),
	})

	for serverStarted {
		time.Sleep(100 * time.Millisecond)
	}
	test.FileExists(t, "test/config.json")
	settings := configuration.Get()
	test.IsEqualInt(t, settings.Authentication.Method, 0)
	test.IsEqualString(t, settings.Authentication.Username, "admin")
	test.IsEqualString(t, settings.Authentication.OauthProvider, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "")
	test.IsEqualInt(t, len(settings.Authentication.OauthUsers), 0)
	test.IsEqualString(t, settings.Authentication.HeaderKey, "")
	test.IsEqualInt(t, len(settings.Authentication.HeaderUsers), 0)
	test.IsEqualBool(t, strings.Contains(settings.Port, "127.0.0.1"), true)
	test.IsEqualBool(t, strings.Contains(settings.Port, ":53842"), true)
	test.IsEqualBool(t, settings.UseSsl, false)
	test.IsEqualString(t, settings.ServerUrl, "http://127.0.0.1:53842/")
	test.IsEqualString(t, settings.RedirectUrl, "https://github.com/Forceu/Gokapi/")
	cconfig, ok := cloudconfig.Load()
	test.IsEqualBool(t, ok, true)
	if os.Getenv("GOKAPI_AWS_BUCKET") == "" {
		test.IsEqualString(t, cconfig.Aws.Bucket, "testbucket")
		test.IsEqualString(t, cconfig.Aws.Region, "testregion")
		test.IsEqualString(t, cconfig.Aws.KeyId, "testapi")
		test.IsEqualString(t, cconfig.Aws.KeySecret, "testsecret")
		test.IsEqualString(t, cconfig.Aws.Endpoint, "testendpoint")
	}
	test.FileExists(t, "test/cloudconfig.yml")

	go RunConfigModification()
	for !serverStarted {
		time.Sleep(100 * time.Millisecond)
	}

	username = "test"
	password = "testpw"

	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/start",
		RequiredContent: []string{"Unauthorized"},
		ExcludedContent: []string{"\"result\":"},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      401,
		Body:            strings.NewReader(testInputHeaderAuth),
	})
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/start",
		RequiredContent: []string{"You can now change the Gokapi configuration."},
		ExcludedContent: []string{"Unauthorized"},
		IsHtml:          false,
		Method:          "POST",
		Headers:         []test.Header{{Name: "Authorization", Value: "Basic dGVzdDp0ZXN0cHc="}},
		ResultCode:      200,
	})

	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"Unauthorized"},
		ExcludedContent: []string{"\"result\":"},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      401,
		Body:            strings.NewReader(testInputHeaderAuth),
	})
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		Headers:         []test.Header{{Name: "Authorization", Value: "Basic dGVzdDp0ZXN0cHc="}},
		ResultCode:      200,
		Body:            strings.NewReader(testInputHeaderAuth),
	})

	for serverStarted {
		time.Sleep(100 * time.Millisecond)
	}
	test.FileExists(t, "test/config.json")
	settings = configuration.Get()
	test.IsEqualInt(t, settings.Authentication.Method, 2)
	test.IsEqualString(t, settings.Authentication.Username, "")
	test.IsEqualString(t, settings.Authentication.OauthProvider, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "")
	test.IsEqualInt(t, len(settings.Authentication.OauthUsers), 0)
	test.IsEqualString(t, settings.Authentication.HeaderKey, "testkey")
	headerUsers := len(settings.Authentication.HeaderUsers)
	test.IsEqualInt(t, headerUsers, 2)
	if headerUsers == 2 {
		test.IsEqualString(t, settings.Authentication.HeaderUsers[0], "test1")
		test.IsEqualString(t, settings.Authentication.HeaderUsers[1], "test2")
	}
	test.IsEqualBool(t, strings.Contains(settings.Port, "127.0.0.1"), false)
	test.IsEqualBool(t, strings.Contains(settings.Port, ":53842"), true)
	test.IsEqualBool(t, settings.UseSsl, true)
	test.IsEqualString(t, settings.ServerUrl, "http://127.0.0.1:53842/")
	test.IsEqualString(t, settings.RedirectUrl, "https://test.com")
	_, ok = cloudconfig.Load()
	if os.Getenv("GOKAPI_AWS_BUCKET") == "" {
		test.IsEqualBool(t, ok, false)
	}
	test.FileDoesNotExist(t, "test/cloudconfig.yml")

	go RunConfigModification()
	for !serverStarted {
		time.Sleep(100 * time.Millisecond)
	}
	username = "test"
	password = "testpw"

	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		Headers:         []test.Header{{Name: "Authorization", Value: "Basic dGVzdDp0ZXN0cHc="}},
		ResultCode:      200,
		Body:            strings.NewReader(testInputOauth),
	})

	for serverStarted {
		time.Sleep(100 * time.Millisecond)
	}

	test.IsEqualString(t, settings.Authentication.OauthProvider, "provider")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "id")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "secret")
	oauthUsers := len(settings.Authentication.OauthUsers)
	test.IsEqualInt(t, oauthUsers, 2)
	if oauthUsers == 2 {
		test.IsEqualString(t, settings.Authentication.OauthUsers[0], "oatest1")
		test.IsEqualString(t, settings.Authentication.OauthUsers[1], "oatest2")
	}
}

var testInputInternalAuth = "[{\"name\":\"authentication_sel\",\"value\":\"0\"},{\"name\":\"auth_username\",\"value\":\"admin\"},{\"name\":\"auth_pw\",\"value\":\"adminadmin\"},{\"name\":\"auth_pw2\",\"value\":\"adminadmin\"},{\"name\":\"oauth_provider\",\"value\":\"\"},{\"name\":\"oauth_id\",\"value\":\"\"},{\"name\":\"oauth_secret\",\"value\":\"\"},{\"name\":\"oauth_header_users\",\"value\":\"\"},{\"name\":\"auth_headerkey\",\"value\":\"\"},{\"name\":\"auth_header_users\",\"value\":\"\"},{\"name\":\"storage_sel\",\"value\":\"cloud\"},{\"name\":\"s3_bucket\",\"value\":\"testbucket\"},{\"name\":\"s3_region\",\"value\":\"testregion\"},{\"name\":\"s3_api\",\"value\":\"testapi\"},{\"name\":\"s3_secret\",\"value\":\"testsecret\"},{\"name\":\"s3_endpoint\",\"value\":\"testendpoint\"},{\"name\":\"localhost_sel\",\"value\":\"1\"},{\"name\":\"ssl_sel\",\"value\":\"0\"},{\"name\":\"port\",\"value\":\"53842\"},{\"name\":\"url\",\"value\":\"http://127.0.0.1:53842/\"},{\"name\":\"url_redirection\",\"value\":\"https://github.com/Forceu/Gokapi/\"},{\"name\":\"encrypt_sel\",\"value\":\"0\"}]\n"
var testInputHeaderAuth = "[{\"name\":\"authentication_sel\",\"value\":\"2\"},{\"name\":\"auth_username\",\"value\":\"\"},{\"name\":\"auth_pw\",\"value\":\"\"},{\"name\":\"auth_pw2\",\"value\":\"\"},{\"name\":\"oauth_provider\",\"value\":\"\"},{\"name\":\"oauth_id\",\"value\":\"\"},{\"name\":\"oauth_secret\",\"value\":\"\"},{\"name\":\"oauth_header_users\",\"value\":\"\"},{\"name\":\"auth_headerkey\",\"value\":\"testkey\"},{\"name\":\"auth_header_users\",\"value\":\"test1 ;test2\"},{\"name\":\"storage_sel\",\"value\":\"local\"},{\"name\":\"s3_bucket\",\"value\":\"\"},{\"name\":\"\",\"value\":\"\"},{\"name\":\"s3_api\",\"value\":\"\"},{\"name\":\"s3_secret\",\"value\":\"\"},{\"name\":\"s3_endpoint\",\"value\":\"\"},{\"name\":\"localhost_sel\",\"value\":\"0\"},{\"name\":\"ssl_sel\",\"value\":\"1\"},{\"name\":\"port\",\"value\":\"53842\"},{\"name\":\"url\",\"value\":\"http://127.0.0.1:53842/\"},{\"name\":\"url_redirection\",\"value\":\"https://test.com\"},{\"name\":\"encrypt_sel\",\"value\":\"0\"}]\n"
var testInputOauth = "[{\"name\":\"authentication_sel\",\"value\":\"1\"},{\"name\":\"auth_username\",\"value\":\"\"},{\"name\":\"auth_pw\",\"value\":\"\"},{\"name\":\"auth_pw2\",\"value\":\"\"},{\"name\":\"oauth_provider\",\"value\":\"provider\"},{\"name\":\"oauth_id\",\"value\":\"id\"},{\"name\":\"oauth_secret\",\"value\":\"secret\"},{\"name\":\"oauth_header_users\",\"value\":\"oatest1; oatest2\"},{\"name\":\"auth_headerkey\",\"value\":\"testkey\"},{\"name\":\"auth_header_users\",\"value\":\"\"},{\"name\":\"storage_sel\",\"value\":\"local\"},{\"name\":\"s3_bucket\",\"value\":\"\"},{\"name\":\"\",\"value\":\"\"},{\"name\":\"s3_api\",\"value\":\"\"},{\"name\":\"s3_secret\",\"value\":\"\"},{\"name\":\"s3_endpoint\",\"value\":\"\"},{\"name\":\"localhost_sel\",\"value\":\"0\"},{\"name\":\"ssl_sel\",\"value\":\"1\"},{\"name\":\"port\",\"value\":\"53842\"},{\"name\":\"url\",\"value\":\"http://127.0.0.1:53842/\"},{\"name\":\"url_redirection\",\"value\":\"https://test.com\"},{\"name\":\"encrypt_sel\",\"value\":\"0\"}]\n"
