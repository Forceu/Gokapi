package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"log"
	"net"
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
	if debugDisableAuth {
		fmt.Println("Debug mode is still on! Exiting test")
		os.Exit(1)
	}
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
		_, _, _, _, err = toConfiguration(&formObjects)
		test.IsNotNilWithMessage(t, err, invalid.toJson())
	}
}

func TestEncryptionSetup(t *testing.T) {
	var e2eConfig configuration.End2EndReconfigParameters
	input := createInputOAuth()
	input.EncryptionLevel.Value = "1"
	formObjects, err := input.toFormObject()
	test.IsNil(t, err)
	config, _, e2eConfig, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualInt(t, len(config.Encryption.Cipher), 32)
	test.IsEqualString(t, config.Encryption.Checksum, "")
	test.IsEqualBool(t, e2eConfig.DeleteEncryptedStorage, false)
	test.IsEqualBool(t, e2eConfig.DeleteEnd2EndEncryption, false)

	input.EncryptionLevel.Value = "2"
	input.EncryptionPassword.Value = "testpw12"
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, _, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualString(t, string(config.Encryption.Cipher), "")
	test.IsEqualInt(t, len(config.Encryption.Checksum), 64)

	isInitialSetup = false

	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	configuration.Get().Encryption.Level = 3
	id := testconfiguration.WriteEncryptedFile()
	file, ok := database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, e2eConfig, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualBool(t, e2eConfig.DeleteEncryptedStorage, true)
	test.IsEqualBool(t, e2eConfig.DeleteEnd2EndEncryption, false)

	configuration.Get().Encryption.Level = 2
	input.EncryptionPassword.Value = "unc"
	id = testconfiguration.WriteEncryptedFile()
	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, _, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	file, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedTime, true)

	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	configuration.Get().Encryption.Level = 2
	input.EncryptionPassword.Value = "otherpw12"
	id = testconfiguration.WriteEncryptedFile()
	_, ok = database.GetMetaDataById(id)
	test.IsEqualBool(t, ok, true)
	formObjects, err = input.toFormObject()
	test.IsNil(t, err)
	config, _, e2eConfig, _, err = toConfiguration(&formObjects)
	test.IsNil(t, err)
	test.IsEqualBool(t, e2eConfig.DeleteEncryptedStorage, true)
	test.IsEqualBool(t, e2eConfig.DeleteEnd2EndEncryption, false)

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
	output, cloudConfig, _, authsettings, err := toConfiguration(&jsonForms)
	test.IsNil(t, err)
	test.IsEqualInt(t, output.Authentication.Method, models.AuthenticationInternal)
	test.IsEqualString(t, cloudConfig.Aws.KeyId, "testapi")
	test.IsEqualString(t, output.Authentication.Username, "admin")
	test.IsNotEqualString(t, authsettings.PasswordInternalAuth, "adminadmin")
	test.IsNotEqualString(t, authsettings.PasswordInternalAuth, "")
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
	credentialUsername = "test"
	credentialPassword = "testpw"
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

type dbFormTest struct {
	DatabaseType   string `form:"dbtype_sel"`
	SqliteLocation string `form:"sqlite_location"`
	RedisLocation  string `form:"redis_location"`
	RedisPrefix    string `form:"redis_prefix"`
	RedisUser      string `form:"redis_user"`
	RedisPw        string `form:"redis_password"`
	RedisUseSsl    string `form:"redis_ssl_sel"`
}

func generateDbFormValues(input dbFormTest) []jsonFormObject {
	result := make([]jsonFormObject, 0)
	v := reflect.ValueOf(input)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		result = append(result, jsonFormObject{
			Name:  t.Field(i).Tag.Get("form"),
			Value: v.Field(i).Interface().(string),
		})
	}
	return result
}

func TestParseDatabaseSettings(t *testing.T) {
	output := models.Configuration{}
	input := generateDbFormValues(dbFormTest{
		DatabaseType:   "0",
		SqliteLocation: "./data/test.sqlite",
		RedisUseSsl:    "0",
	})
	expected := "sqlite://./data/test.sqlite"
	err := parseDatabaseSettings(&output, &input)
	test.IsNil(t, err)
	test.IsEqualString(t, output.DatabaseUrl, expected)

	input = generateDbFormValues(dbFormTest{
		DatabaseType:  "1",
		RedisLocation: "127.0.0.1:1234",
		RedisUseSsl:   "0",
	})
	expected = "redis://127.0.0.1:1234"
	err = parseDatabaseSettings(&output, &input)
	test.IsNil(t, err)
	test.IsEqualString(t, output.DatabaseUrl, expected)

	input = generateDbFormValues(dbFormTest{
		DatabaseType:  "1",
		RedisLocation: "127.0.0.1:1234",
		RedisPrefix:   "pre_",
		RedisUser:     "testuser",
		RedisPw:       "testpw",
		RedisUseSsl:   "1",
	})
	expected = "redis://testuser:testpw@127.0.0.1:1234?prefix=pre_&ssl=true"
	err = parseDatabaseSettings(&output, &input)
	test.IsNil(t, err)
	test.IsEqualString(t, output.DatabaseUrl, expected)
}

func TestRunConfigModification(t *testing.T) {
	testconfiguration.Create(false)
	credentialUsername = ""
	credentialPassword = ""
	finish := make(chan bool)
	go func() {
		waitForServer(t, true)
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
	test.IsEqualInt(t, len(credentialUsername), 6)
	test.IsEqualInt(t, len(credentialPassword), 10)
	isInitialSetup = true
	<-finish
}

func waitForServer(t *testing.T, expectedRunning bool) bool {
	const maxCount = 100
	counter := 0
	for counter < maxCount {
		isRunning := isServerRunning(t)
		if isRunning == expectedRunning {
			return true
		}
		counter++
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Timeout after 10 seconds")
	return false
}

func isServerRunning(t *testing.T) bool {
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	t.Helper()
	req, err := http.NewRequest("GET", "http://localhost:53842/admin", nil)
	test.IsNil(t, err)
	_, err = client.Do(req)
	return err == nil
}

func TestIntegration(t *testing.T) {
	testconfiguration.Delete()
	test.FileDoesNotExist(t, "test/config.json")
	go RunIfFirstStart()
	waitForServer(t, true)

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

	setupVals := createInputInternalAuth()
	test.HttpPageResultJson(t, test.HttpTestConfig{
		Url:             "http://localhost:53842/setup/setupResult",
		RequiredContent: []string{"\"result\": \"OK\""},
		ExcludedContent: []string{"\"result\": \"Error\""},
		IsHtml:          false,
		Method:          "POST",
		ResultCode:      200,
		Body:            strings.NewReader(setupVals.toJson()),
	})

	waitForServer(t, false)
	test.FileExists(t, "test/config.json")
	settings := configuration.Get()
	test.IsEqualBool(t, settings.IncludeFilename, true)
	test.IsEqualInt(t, settings.Authentication.Method, 0)
	test.IsEqualString(t, settings.Authentication.Username, "admin")
	test.IsEqualString(t, settings.Authentication.OAuthProvider, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "")
	test.IsEqualBool(t, settings.Authentication.OnlyRegisteredUsers, false)
	test.IsEqualString(t, settings.Authentication.HeaderKey, "")
	test.IsEqualBool(t, strings.Contains(settings.Port, "127.0.0.1"), true)
	test.IsEqualBool(t, strings.Contains(settings.Port, ":53842"), true)
	test.IsEqualBool(t, settings.UseSsl, false)
	test.IsEqualBool(t, settings.SaveIp, false)
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
		test.IsEqualBool(t, cconfig.Aws.ProxyDownload, true)
	}
	test.FileExists(t, "test/cloudconfig.yml")

	go RunConfigModification()
	waitForServer(t, true)

	credentialUsername = "test"
	credentialPassword = "testpw"

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

	waitForServer(t, false)

	test.FileExists(t, "test/config.json")
	settings = configuration.Get()
	test.IsEqualBool(t, settings.IncludeFilename, false)
	test.IsEqualInt(t, settings.Authentication.Method, 2)
	test.IsEqualString(t, settings.Authentication.Username, "test1")
	test.IsEqualString(t, settings.Authentication.OAuthProvider, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "")
	test.IsEqualString(t, settings.Authentication.HeaderKey, "testkey")
	test.IsEqualBool(t, settings.Authentication.OnlyRegisteredUsers, true)
	test.IsEqualBool(t, strings.Contains(settings.Port, "127.0.0.1"), false)
	test.IsEqualBool(t, strings.Contains(settings.Port, ":53842"), true)
	test.IsEqualBool(t, settings.UseSsl, true)
	test.IsEqualBool(t, settings.SaveIp, true)
	test.IsEqualString(t, settings.ServerUrl, "http://127.0.0.1:53842/")
	test.IsEqualString(t, settings.RedirectUrl, "https://test.com")
	test.IsEqualBool(t, settings.PicturesAlwaysLocal, false)
	cconfig, ok = cloudconfig.Load()
	if os.Getenv("GOKAPI_AWS_BUCKET") == "" {
		test.IsEqualBool(t, ok, false)
	}
	test.FileDoesNotExist(t, "test/cloudconfig.yml")

	go RunConfigModification()
	waitForServer(t, true)

	credentialUsername = "test"
	credentialPassword = "testpw"

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

	waitForServer(t, false)

	test.IsEqualBool(t, settings.PicturesAlwaysLocal, true)
	test.IsEqualBool(t, cconfig.Aws.ProxyDownload, false)
	test.IsEqualString(t, settings.Authentication.OAuthProvider, "provider")
	test.IsEqualString(t, settings.Authentication.OAuthClientId, "id")
	test.IsEqualString(t, settings.Authentication.OAuthClientSecret, "secret")
	test.IsEqualString(t, settings.Authentication.Username, "oatest1")
	test.IsEqualBool(t, settings.Authentication.OnlyRegisteredUsers, true)
}

type setupValues struct {
	BindLocalhost                 setupEntry `form:"localhost_sel" isBool:"true"`
	UseSsl                        setupEntry `form:"ssl_sel" isBool:"true"`
	SaveIp                        setupEntry `form:"logip_sel" isBool:"true"`
	Port                          setupEntry `form:"port" isInt:"true"`
	PublicName                    setupEntry `form:"public_name"`
	ExtUrl                        setupEntry `form:"url"`
	RedirectUrl                   setupEntry `form:"url_redirection"`
	IncludeFilename               setupEntry `form:"showfilename_sel" isBool:"true"`
	AuthenticationMode            setupEntry `form:"authentication_sel" isInt:"true"`
	AuthUsername                  setupEntry `form:"auth_username"`
	AuthPassword                  setupEntry `form:"auth_pw"`
	OAuthProvider                 setupEntry `form:"oauth_provider"`
	OAuthClientId                 setupEntry `form:"oauth_id"`
	OAuthClientSecret             setupEntry `form:"oauth_secret"`
	OAuthAuthorisedGroups         setupEntry `form:"oauth_allowed_groups"`
	OAuthAdminUser                setupEntry `form:"oauth_admin_user"`
	OAuthScopeGroup               setupEntry `form:"oauth_scope_groups"`
	OAuthRestrictGroups           setupEntry `form:"oauth_restrict_groups" isBool:"true"`
	OAuthRecheckInterval          setupEntry `form:"oauth_recheck_interval" isInt:"true"`
	OAuthOnlyRegisteredUsers      setupEntry `form:"oauth_only_registered_users" isBool:"true"`
	AuthHeaderKey                 setupEntry `form:"auth_headerkey"`
	AuthHeaderAdmin               setupEntry `form:"auth_header_admin"`
	AuthHeaderOnlyRegisteredUsers setupEntry `form:"auth_header_only_registered_users" isBool:"true"`
	StorageSelection              setupEntry `form:"storage_sel"`
	PicturesAlwaysLocal           setupEntry `form:"storage_sel_image"`
	ProxyDownloads                setupEntry `form:"storage_sel_proxy"`
	S3Bucket                      setupEntry `form:"s3_bucket"`
	S3Region                      setupEntry `form:"s3_region"`
	S3ApiKey                      setupEntry `form:"s3_api"`
	S3ApiSecret                   setupEntry `form:"s3_secret"`
	S3Endpoint                    setupEntry `form:"s3_endpoint"`
	EncryptionLevel               setupEntry `form:"encrypt_sel" isInt:"true"`
	EncryptionPassword            setupEntry `form:"enc_pw"`
	DatabaseType                  setupEntry `form:"dbtype_sel" isInt:"true"`
	SqliteLocation                setupEntry `form:"sqlite_location"`
	RedisLocation                 setupEntry `form:"redis_location"`
	RedisPrefix                   setupEntry `form:"redis_prefix"`
	RedisUser                     setupEntry `form:"redis_user"`
	RedisPw                       setupEntry `form:"redis_password"`
	RedisUseSsl                   setupEntry `form:"redis_ssl_sel" isBool:"true"`
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

func createInvalidSetupValues() []setupValues {
	var result []setupValues
	input := createInputInternalAuth()
	t := reflect.TypeOf(setupValues{})
	for i := 0; i < t.NumField(); i++ {
		invalidSetup := input
		v := reflect.ValueOf(&invalidSetup)
		v.Elem().Field(i).FieldByName("FormName").SetString("XXXinvalidNameXXX")
		result = append(result, invalidSetup)

		tag := t.Field(i).Tag.Get("isInt")
		if tag == "true" {
			invalidSetup = input
			v.Elem().Field(i).FieldByName("Value").SetString("XXXnotIntXXX")
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
	values.PublicName.Value = "Test Name"
	values.UseSsl.Value = "0"
	values.IncludeFilename.Value = "1"
	values.Port.Value = "53842"
	values.ExtUrl.Value = "http://127.0.0.1:53842/"
	values.RedirectUrl.Value = "https://github.com/Forceu/Gokapi/"
	values.AuthenticationMode.Value = "0"
	values.AuthUsername.Value = "admin"
	values.AuthPassword.Value = "adminadmin"
	values.StorageSelection.Value = "cloud"
	values.ProxyDownloads.Value = "proxy"
	values.S3Bucket.Value = "testbucket"
	values.S3Region.Value = "testregion"
	values.S3ApiKey.Value = "testapi"
	values.S3ApiSecret.Value = "testsecret"
	values.S3Endpoint.Value = "testendpoint"
	values.EncryptionLevel.Value = "0"
	values.PicturesAlwaysLocal.Value = "nochange"
	values.SaveIp.Value = "0"
	values.OAuthOnlyRegisteredUsers.Value = "false"
	values.OAuthRestrictGroups.Value = "false"
	values.OAuthRecheckInterval.Value = "12"
	values.AuthHeaderOnlyRegisteredUsers.Value = "false"
	values.DatabaseType.Value = "0"
	values.SqliteLocation.Value = "./test/gokapi.sqlite"
	values.RedisUseSsl.Value = "0"

	return values
}

func createInputHeaderAuth() setupValues {
	values := setupValues{}
	values.init()

	values.BindLocalhost.Value = "0"
	values.PublicName.Value = "Test Name"
	values.UseSsl.Value = "1"
	values.Port.Value = "53842"
	values.ExtUrl.Value = "http://127.0.0.1:53842/"
	values.RedirectUrl.Value = "https://test.com"
	values.AuthenticationMode.Value = "2"
	values.AuthHeaderKey.Value = "testkey"
	values.AuthHeaderAdmin.Value = "test1"
	values.AuthHeaderOnlyRegisteredUsers.Value = "true"
	values.StorageSelection.Value = "local"
	values.EncryptionLevel.Value = "0"
	values.SaveIp.Value = "1"
	values.OAuthOnlyRegisteredUsers.Value = "false"
	values.OAuthRestrictGroups.Value = "false"
	values.OAuthRecheckInterval.Value = "12"
	values.IncludeFilename.Value = "0"
	values.DatabaseType.Value = "0"
	values.SqliteLocation.Value = "./test/gokapi.sqlite"
	values.RedisUseSsl.Value = "0"

	return values
}

func createInputOAuth() setupValues {
	values := createInputHeaderAuth()
	values.AuthenticationMode.Value = "1"
	values.OAuthProvider.Value = "provider"
	values.OAuthClientId.Value = "id"
	values.OAuthClientSecret.Value = "secret"
	values.OAuthOnlyRegisteredUsers.Value = "true"
	values.OAuthRestrictGroups.Value = "true"
	values.OAuthAdminUser.Value = "oatest1"
	values.OAuthScopeGroup.Value = "groups"
	values.OAuthAuthorisedGroups.Value = "group1; group2"
	values.StorageSelection.Value = "local"
	values.PicturesAlwaysLocal.Value = "local"
	values.ProxyDownloads.Value = "default"
	return values
}

func TestIsErrorAddressAlreadyInUse(t *testing.T) {

	l, err := net.Listen("tcp", "127.0.0.1:19888")
	test.IsNil(t, err)
	srv2 := http.Server{
		Addr: ":19888",
	}
	httpError := make(chan error)
	go func() {
		sErr := srv2.ListenAndServe()
		httpError <- sErr
	}()
	select {
	case err = <-httpError:
		test.IsEqualBool(t, isErrorAddressAlreadyInUse(err), true)
	case <-time.After(15 * time.Second):
		t.Fatalf("httpError timeout after 15 seconds")
	}
	err = errors.New("other error")
	test.IsEqualBool(t, isErrorAddressAlreadyInUse(err), false)
	l.Close()
}
