package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var jsonForms []jsonFormObject

func TestMain(m *testing.M) {
	testconfiguration.SetDirEnv()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestDebugNotSet(t *testing.T) {
	test.IsEqualBool(t, debugDisableAuth, false)
}

func TestInputToJson(t *testing.T) {
	var err error
	_, r := test.GetRecorder("POST", "/setupResult", nil, nil, bytes.NewBufferString("invalid"))
	jsonForms, err = inputToJsonForm(r)
	test.IsNotNil(t, err)
	setupValues := createInputInternalAuth()
	buf := bytes.NewBufferString(setupValues.toJson())
	_, r = test.GetRecorder("POST", "/setupResult", nil, nil, buf)
	jsonForms, err = inputToJsonForm(r)
	test.IsNil(t, err)
	for _, item := range jsonForms {
		if item.Name == "auth_username" {
			test.IsEqualString(t, item.Value, "admin")
		}
	}
}

func TestMissingSetupValues(t *testing.T) {
	invalidInputs := createInvalidSetupValues()
	for _, invalid := range invalidInputs {
		formObjects, err := invalid.toFormObject()
		test.IsNil(t, err)
		_, _, err = toConfiguration(&formObjects)
		test.IsNotNilWithMessage(t, err, invalid.toJson())
	}
}

func TestEncryptionSetup(t *testing.T) {
	input := createInputOAuth()
	input.EncryptionLevel.Value = "1"
	formObjects, err := input.toFormObject()
	test.IsNil(t, err)
	config, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualInt(t, len(config.Encryption.Cipher), 32)
	test.IsEqualString(t, config.Encryption.Checksum, "")

	input.EncryptionLevel.Value = "2"
	input.EncryptionPassword.Value = "testpw"
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualString(t, string(config.Encryption.Cipher), "")
	test.IsEqualInt(t, len(config.Encryption.Checksum), 64)

	isInitialSetup = false

	testconfiguration.Create(false)
	configuration.Load()
	configuration.Get().Encryption.Level = 3
	id := testconfiguration.WriteEncryptedFile()
	file, ok := database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	file, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, false)

	configuration.Get().Encryption.Level = 2
	input.EncryptionPassword.Value = "unc"
	id = testconfiguration.WriteEncryptedFile()
	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	file, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, true)

	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	configuration.Get().Encryption.Level = 2
	input.EncryptionPassword.Value = "otherpw"
	id = testconfiguration.WriteEncryptedFile()
	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	file, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, false)

	database.Close()
	testconfiguration.Delete()

	isInitialSetup = true
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
		time.Sleep(500 * time.Millisecond)
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

	setupValues := createInputInternalAuth()
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      200,
		Body:            strings.NewReader(setupValues.toJson()),
	})

	counter := 0
	for serverStarted {
		time.Sleep(100 * time.Millisecond)
		counter++
		if counter > 100 {
			t.Fatal("Unbroken loop")
		}
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

	setupInput := createInputHeaderAuth()
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/start",
		RequiredContent: []string{"Unauthorized"},
		ExcludedContent: []string{"\"result\":"},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      401,
		Body:            strings.NewReader(setupInput.toJson()),
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
		Body:            strings.NewReader(setupInput.toJson()),
	})
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		Headers:         []test.Header{{Name: "Authorization", Value: "Basic dGVzdDp0ZXN0cHc="}},
		ResultCode:      200,
		Body:            strings.NewReader(setupInput.toJson()),
	})

	counter = 0
	for serverStarted {
		time.Sleep(100 * time.Millisecond)
		counter++
		if counter > 100 {
			t.Fatal("Unbroken loop")
		}
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
	test.IsEqualBool(t, settings.PicturesAlwaysLocal, false)
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

	setupInput = createInputOAuth()
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		Headers:         []test.Header{{Name: "Authorization", Value: "Basic dGVzdDp0ZXN0cHc="}},
		ResultCode:      200,
		Body:            strings.NewReader(setupInput.toJson()),
	})

	counter = 0
	for serverStarted {
		time.Sleep(100 * time.Millisecond)
		counter++
		if counter > 100 {
			t.Fatal("Unbroken loop")
		}
	}

	test.IsEqualBool(t, settings.PicturesAlwaysLocal, true)
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

type setupValues struct {
	BindLocalhost        setupEntry `form:"localhost_sel" isBool:"true"`
	UseSsl               setupEntry `form:"ssl_sel" isBool:"true"`
	Port                 setupEntry `form:"port" isInt:"true"`
	ExtUrl               setupEntry `form:"url"`
	RedirectUrl          setupEntry `form:"url_redirection"`
	AuthenticationMode   setupEntry `form:"authentication_sel" isInt:"true"`
	AuthUsername         setupEntry `form:"auth_username"`
	AuthPassword         setupEntry `form:"auth_pw"`
	OAuthProvider        setupEntry `form:"oauth_provider"`
	OAuthClientId        setupEntry `form:"oauth_id"`
	OAuthClientSecret    setupEntry `form:"oauth_secret"`
	OAuthAuthorisedUsers setupEntry `form:"oauth_header_users"`
	AuthHeaderKey        setupEntry `form:"auth_headerkey"`
	AuthHeaderUsers      setupEntry `form:"auth_header_users"`
	StorageSelection     setupEntry `form:"storage_sel"`
	PicturesAlwaysLocal  setupEntry `form:"storage_sel_image"`
	S3Bucket             setupEntry `form:"s3_bucket"`
	S3Region             setupEntry `form:"s3_region"`
	S3ApiKey             setupEntry `form:"s3_api"`
	S3ApiSecret          setupEntry `form:"s3_secret"`
	S3Endpoint           setupEntry `form:"s3_endpoint"`
	EncryptionLevel      setupEntry `form:"encrypt_sel" isInt:"true"`
	EncryptionPassword   setupEntry `form:"enc_pw"`
}

func (s *setupValues) init() {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(setupValues{})
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("form")
		v.Elem().Field(i).FieldByName("FormName").SetString(tag)
		v.Elem().Field(i).FieldByName("Enabled").SetBool(true)
	}
}

func (s *setupValues) toJson() string {
	var values []jsonFormObject

	v := reflect.ValueOf(s)
	t := reflect.TypeOf(setupValues{})
	for i := 0; i < t.NumField(); i++ {
		values = append(values, jsonFormObject{
			Name:  v.Elem().Field(i).FieldByName("FormName").String(),
			Value: v.Elem().Field(i).FieldByName("Value").String(),
		})
	}

	result, err := json.Marshal(values)
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}

func (s *setupValues) toFormObject() ([]jsonFormObject, error) {
	r := httptest.NewRequest("POST", "/setup", strings.NewReader(s.toJson()))
	setupResult, err := inputToJsonForm(r)
	if err != nil {
		return nil, err
	}
	return setupResult, err
}

func TestIsPwLongEnough(t *testing.T) {
	isInitialSetup = true
	test.IsEqualBool(t, isPwLongEnough("unc"), false)
	test.IsEqualBool(t, isPwLongEnough("12345"), false)
	test.IsEqualBool(t, isPwLongEnough("123456"), true)
	test.IsEqualBool(t, isPwLongEnough("1234567"), true)
	isInitialSetup = false
	test.IsEqualBool(t, isPwLongEnough("unc"), true)
	test.IsEqualBool(t, isPwLongEnough("12345"), false)
	test.IsEqualBool(t, isPwLongEnough("123456"), true)
	test.IsEqualBool(t, isPwLongEnough("1234567"), true)
}

func createInvalidSetupValues() []setupValues {
	var result []setupValues
	input := createInputInternalAuth()
	t := reflect.TypeOf(setupValues{})
	for i := 0; i < t.NumField(); i++ {
		invalidSetup := input
		v := reflect.ValueOf(&invalidSetup)
		v.Elem().Field(i).FieldByName("FormName").SetString("invalid")
		result = append(result, invalidSetup)

		tag := t.Field(i).Tag.Get("isInt")
		if tag == "true" {
			invalidSetup = input
			v.Elem().Field(i).FieldByName("Value").SetString("notInt")
			result = append(result, invalidSetup)
		}
		tag = t.Field(i).Tag.Get("isBool")
		if tag == "true" {
			invalidSetup = input
			v.Elem().Field(i).FieldByName("Value").SetString("2")
			result = append(result, invalidSetup)
		}
	}
	invalidSetup := input
	invalidSetup.AuthenticationMode.Value = "4"
	result = append(result, invalidSetup)

	invalidSetup = input
	invalidSetup.EncryptionLevel.Value = "-1"
	result = append(result, invalidSetup)

	invalidSetup = input
	invalidSetup.EncryptionLevel.Value = "9"
	result = append(result, invalidSetup)

	invalidSetup = input
	invalidSetup.EncryptionLevel.Value = "5" // e2e not implemented yet
	result = append(result, invalidSetup)

	invalidSetup = input
	invalidSetup.EncryptionLevel.Value = "4"
	invalidSetup.EncryptionPassword.Value = "2shrt"
	result = append(result, invalidSetup)

	return result
}

type setupEntry struct {
	FormName string
	Enabled  bool
	Value    string
}

func createInputInternalAuth() setupValues {
	values := setupValues{}
	values.init()

	values.BindLocalhost.Value = "1"
	values.UseSsl.Value = "0"
	values.Port.Value = "53842"
	values.ExtUrl.Value = "http://127.0.0.1:53842/"
	values.RedirectUrl.Value = "https://github.com/Forceu/Gokapi/"
	values.AuthenticationMode.Value = "0"
	values.AuthUsername.Value = "admin"
	values.AuthPassword.Value = "adminadmin"
	values.StorageSelection.Value = "cloud"
	values.S3Bucket.Value = "testbucket"
	values.S3Region.Value = "testregion"
	values.S3ApiKey.Value = "testapi"
	values.S3ApiSecret.Value = "testsecret"
	values.S3Endpoint.Value = "testendpoint"
	values.EncryptionLevel.Value = "0"
	values.PicturesAlwaysLocal.Value = "nochange"

	return values
}

func createInputHeaderAuth() setupValues {
	values := setupValues{}
	values.init()

	values.BindLocalhost.Value = "0"
	values.UseSsl.Value = "1"
	values.Port.Value = "53842"
	values.ExtUrl.Value = "http://127.0.0.1:53842/"
	values.RedirectUrl.Value = "https://test.com"
	values.AuthenticationMode.Value = "2"
	values.AuthHeaderKey.Value = "testkey"
	values.AuthHeaderUsers.Value = "test1 ;test2"
	values.StorageSelection.Value = "local"
	values.EncryptionLevel.Value = "0"

	return values
}

func createInputOAuth() setupValues {
	values := createInputHeaderAuth()
	values.AuthenticationMode.Value = "1"
	values.OAuthProvider.Value = "provider"
	values.OAuthClientId.Value = "id"
	values.OAuthClientSecret.Value = "secret"
	values.OAuthAuthorisedUsers.Value = "oatest1; oatest2"
	values.StorageSelection.Value = "local"
	values.PicturesAlwaysLocal.Value = "local"
	return values
}
