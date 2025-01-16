package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	configuration.ConnectDatabase()
	generateTestUsers()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

var newKeyId string

func generateTestUsers() {
	newUser := models.User{
		Id:            100,
		Name:          "TestUser",
		Permissions:   models.UserPermissionNone,
		UserLevel:     models.UserLevelUser,
		ResetPassword: false,
	}
	newSuperAdmin := models.User{
		Id:            101,
		Name:          "TestSuperAdmin",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelSuperAdmin,
		ResetPassword: false,
	}
	newAdmin := models.User{
		Id:            102,
		Name:          "TestAdmin",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelAdmin,
		ResetPassword: false,
	}
	database.SaveUser(newUser, false)
	database.SaveUser(newSuperAdmin, false)
	database.SaveUser(newAdmin, false)
}

func getRecorder(url, apikey string, headers []test.Header) (*httptest.ResponseRecorder, *http.Request) {
	var passedHeaders []test.Header
	if apikey != "" {
		passedHeaders = append(passedHeaders, test.Header{
			Name:  "apikey",
			Value: apikey,
		})
	}
	for _, header := range headers {
		passedHeaders = append(passedHeaders, header)
	}
	return test.GetRecorder("GET", url, nil, passedHeaders, nil)
}

func testAuthorisation(t *testing.T, url string, requiredPermission uint8) models.ApiKey {
	t.Helper()
	w, r := getRecorder(url, "", []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	w, r = getRecorder(url, "invalid", []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	newApiKeyUser := generateNewKey(false, 100)
	w, r = getRecorder(url, newApiKeyUser.Id, []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	for _, permission := range getAvailablePermissions(t) {
		if permission == requiredPermission {
			continue
		}
		newApiKeyUser.GrantPermission(permission)
		database.SaveApiKey(newApiKeyUser)
		w, r = getRecorder(url, newApiKeyUser.Id, []test.Header{{}})
		Process(w, r)
		test.IsEqualBool(t, w.Code != 200, true)
		test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)
		newApiKeyUser.RemovePermission(permission)
		database.SaveApiKey(newApiKeyUser)
	}
	newApiKeyUser.Permissions = models.ApiPermAll
	newApiKeyUser.RemovePermission(requiredPermission)
	database.SaveApiKey(newApiKeyUser)
	w, r = getRecorder(url, newApiKeyUser.Id, []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)
	newApiKeyUser.Permissions = models.ApiPermNone
	newApiKeyUser.GrantPermission(requiredPermission)
	database.SaveApiKey(newApiKeyUser)
	return newApiKeyUser
}

type invalidParameterValue struct {
	Value        string
	ErrorMessage string
	StatusCode   int
}

func testInvalidParameters(t *testing.T, url string, apiKey models.ApiKey, correctHeaders []test.Header, headerName string, invalidValues []invalidParameterValue) {
	t.Helper()
	for _, invalidHeader := range invalidValues {
		headers := make([]test.Header, len(correctHeaders))
		copy(headers, correctHeaders)
		headers = append(headers, test.Header{
			Name:  headerName,
			Value: invalidHeader.Value,
		})
		w, r := getRecorder(url, apiKey.Id, headers)
		Process(w, r)
		test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
		test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		if invalidHeader.Value == "" {
			w, r = getRecorder(url, apiKey.Id, correctHeaders)
			Process(w, r)
			test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
			test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		}
	}
}

func TestUserCreate(t *testing.T) {
	const apiUrl = "/user/create"
	const headerUsername = "username"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)

	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUsername,
		Value: "1234",
	}})
	Process(w, r)
	test.ResponseBodyContains(t, w, `{"id":103,"name":"1234","permissions":0,"userLevel":2,"lastOnline":0,"resetPassword":false}`)

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid username provided."}`,
			StatusCode:   400,
		},
		{
			Value:        "123",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid username provided."}`,
			StatusCode:   400,
		},
		{
			Value:        "123",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid username provided."}`,
			StatusCode:   400,
		},
		{
			Value:        "1234",
			ErrorMessage: `{"Result":"error","ErrorMessage":"User already exists."}`,
			StatusCode:   409,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey, []test.Header{{}}, headerUsername, invalidParameter)
}

func TestUserChangeRank(t *testing.T) {
	const apiUrl = "/user/changeRank"
	const headerUserId = "userid"
	const headerNewRank = "newRank"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid user id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "99",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid user id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "100",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Cannot modify yourself"}`,
			StatusCode:   400,
		},
		{
			Value:        "101",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Cannot modify super admin"}`,
			StatusCode:   400,
		},
	}

	testInvalidParameters(t, apiUrl, apiKey, []test.Header{{}}, headerUserId, invalidParameter)
	var validHeaders = []test.Header{
		{
			Name:  headerUserId,
			Value: "102",
		},
	}
	invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid rank sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid rank sent"}`,
			StatusCode:   400,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey, validHeaders, headerNewRank, invalidParameter)

	user, ok := database.GetUser(102)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelAdmin)
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: "102",
	}, {
		Name:  headerNewRank,
		Value: "USER",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(102)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelUser)
	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: "102",
	}, {
		Name:  headerNewRank,
		Value: "ADMIN",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(102)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelAdmin)
}

func TestNewKey(t *testing.T) {
	newKey := generateNewKey(true, 5)
	newKeyId = newKey.Id
	key, ok := database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, newKey, key)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
	test.IsEqualBool(t, key.Permissions == models.ApiPermDefault, true)
	test.IsEqualInt(t, key.UserId, 5)

	newKey = generateNewKey(false, 5)
	newKeyId = newKey.Id
	key, ok = database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.Permissions == models.ApiPermNone, true)
}

func TestDeleteKey(t *testing.T) {
	key, ok := database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
	result := deleteApiKey(newKeyId)
	test.IsEqualBool(t, result, true)
	_, ok = database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, false)
	result = deleteApiKey("invalid")
	test.IsEqualBool(t, result, false)
}

func TestIsValidApiKey(t *testing.T) {
	user, isValid := isValidApiKey("", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, false)
	_, isValid = isValidApiKey("invalid", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, false)
	user, isValid = isValidApiKey("validkey", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	test.IsEqualInt(t, user.Id, 5)
	key, ok := database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, true)
	user, isValid = isValidApiKey("validkey", true, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	test.IsEqualInt(t, user.Id, 5)
	key, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, false)

	newApiKey := generateNewKey(false, 5)
	user, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	for _, permission := range getAvailablePermissions(t) {
		_, isValid = isValidApiKey(newApiKey.Id, true, permission)
		test.IsEqualBool(t, isValid, false)
	}
	for _, newPermission := range getAvailablePermissions(t) {
		setPermissionApikey(newApiKey.Id, newPermission, t)
		for _, permission := range getAvailablePermissions(t) {
			_, isValid = isValidApiKey(newApiKey.Id, true, permission)
			test.IsEqualBool(t, isValid, permission == newPermission)
		}
	}
	setPermissionApikey(newApiKey.Id, models.ApiPermEdit|models.ApiPermDelete, t)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermEdit)
	test.IsEqualBool(t, isValid, true)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermAll)
	test.IsEqualBool(t, isValid, false)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermView)
	test.IsEqualBool(t, isValid, false)
}

func setPermissionApikey(key string, newPermission uint8, t *testing.T) {
	apiKey, ok := database.GetApiKey(key)
	test.IsEqualBool(t, ok, true)
	apiKey.Permissions = newPermission
	database.SaveApiKey(apiKey)
}

func getAvailablePermissions(t *testing.T) []uint8 {
	result := []uint8{models.ApiPermView, models.ApiPermUpload, models.ApiPermDelete, models.ApiPermApiMod, models.ApiPermEdit, models.ApiPermReplace, models.ApiPermManageUsers}
	sum := 0
	for _, perm := range result {
		sum = sum + int(perm)
	}
	if sum != models.ApiPermAll {
		t.Fatal("List of permissions are incorrect")
	}
	return result
}

func TestGetSystemKey(t *testing.T) {
	keys := database.GetAllApiKeys()
	for _, key := range keys {
		if key.IsSystemKey {
			t.Error("No system key expected, but found")
		}
	}
	systemKey := GetSystemKey(5)
	retrievedSystemKey, ok := database.GetApiKey(systemKey)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedSystemKey.IsSystemKey, true)
	test.IsEqualBool(t, retrievedSystemKey.Permissions == models.ApiPermAll, true)
	test.IsEqualBool(t, retrievedSystemKey.Expiry > time.Now().Add(time.Hour*47).Unix(), true)
	newKey := GetSystemKey(5)
	test.IsEqualBool(t, systemKey == newKey, true)
	retrievedSystemKey.Expiry = time.Now().Add(time.Hour * 23).Unix()
	database.SaveApiKey(retrievedSystemKey)
	newKey = GetSystemKey(5)
	test.IsEqualBool(t, systemKey != newKey, true)

	newUser := models.User{
		Id:          70,
		Name:        "TestNoUser",
		Permissions: models.UserPermissionAll,
		UserLevel:   models.UserLevelUser,
		LastOnline:  0,
	}
	newUser.RemovePermission(models.UserPermManageUsers)
	database.SaveUser(newUser, false)
	newUser = models.User{
		Id:          71,
		Name:        "TestNoReplace",
		Permissions: models.UserPermissionAll,
		UserLevel:   models.UserLevelUser,
		LastOnline:  0,
	}
	newUser.RemovePermission(models.UserPermReplaceUploads)
	database.SaveUser(newUser, false)

	newKey = GetSystemKey(70)
	systemApiKey, ok := database.GetApiKey(newKey)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, systemApiKey.HasPermissionEdit(), true)
	test.IsEqualBool(t, systemApiKey.HasPermissionManageUsers(), false)
	test.IsEqualBool(t, systemApiKey.HasPermissionReplace(), true)
	newKey = GetSystemKey(71)
	systemApiKey, ok = database.GetApiKey(newKey)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, systemApiKey.HasPermissionEdit(), true)
	test.IsEqualBool(t, systemApiKey.HasPermissionManageUsers(), true)
	test.IsEqualBool(t, systemApiKey.HasPermissionReplace(), false)

	defer test.ExpectPanic(t)
	GetSystemKey(99)
}

func TestDelete(t *testing.T) {
	database.SaveApiKey(models.ApiKey{
		Id:       "toDelete",
		PublicId: "toDelete",
		UserId:   5,
	})
	_, ok := database.GetApiKey("toDelete")
	test.IsEqualBool(t, ok, true)

	w, r := test.GetRecorder("GET", "/auth/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "invalid",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = test.GetRecorder("GET", "/auth/delete", nil, []test.Header{{
		Name:  "apiKeyToModify",
		Value: "toDelete",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")

	w, r = test.GetRecorder("GET", "/auth/delete", nil, []test.Header{{
		Name:  "apiKeyToModify",
		Value: "toDelete",
	}, {
		Name:  "apikey",
		Value: getNewKeyWithPermissionMissing(t, models.ApiPermApiMod).Id,
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")

	w, r = test.GetRecorder("GET", "/auth/delete", nil, []test.Header{{
		Name:  "apiKeyToModify",
		Value: "toDelete",
	}, {
		Name:  "apikey",
		Value: getNewKeyWithAllPermissions(t).Id,
	}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	_, ok = database.GetApiKey("toDelete")
	test.IsEqualBool(t, ok, false)

	w, r = test.GetRecorder("GET", "/auth/delete", nil, []test.Header{{
		Name:  "apiKeyToModify",
		Value: "toDelete",
	}, {
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Invalid api key provided.\"}")
}

func getNewKeyWithAllPermissions(t *testing.T) models.ApiKey {
	key := generateNewKey(false, 5)
	validKey, ok := database.GetApiKey(key.Id)
	test.IsEqualBool(t, ok, true)
	validKey.GrantPermission(models.ApiPermAll)
	database.SaveApiKey(validKey)
	return validKey
}

func getNewKeyWithPermissionMissing(t *testing.T, removePerm uint8) models.ApiKey {
	key := generateNewKey(false, 5)
	validKey, ok := database.GetApiKey(key.Id)
	test.IsEqualBool(t, ok, true)
	validKey.GrantPermission(models.ApiPermAll)
	validKey.RemovePermission(removePerm)
	database.SaveApiKey(validKey)
	return validKey
}

func countApiKeys() int {
	return len(database.GetAllApiKeys())
}

func TestNewApiKey(t *testing.T) {
	keysBefore := countApiKeys()
	w, r := test.GetRecorder("GET", "/auth/create", nil, []test.Header{{
		Name:  "apikey",
		Value: "invalid",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = test.GetRecorder("GET", "/auth/create", nil, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")

	w, r = test.GetRecorder("GET", "/auth/create", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "friendlyName",
		Value: "New Key",
	}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	keysAfter := countApiKeys()
	test.IsEqualInt(t, keysAfter, keysBefore+1)
	var result models.ApiKeyOutput
	err := json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)

	newKey, ok := database.GetApiKey(result.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, newKey.FriendlyName, "New Key")

	w, r = test.GetRecorder("GET", "/auth/create", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	keysAfter = countApiKeys()
	test.IsEqualInt(t, keysAfter, keysBefore+2)
	err = json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)

	newKey, ok = database.GetApiKey(result.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, newKey.FriendlyName, "Unnamed key")

	w, r = test.GetRecorder("GET", "/auth/create", nil, []test.Header{{
		Name:  "apikey",
		Value: getNewKeyWithPermissionMissing(t, models.ApiPermApiMod).Id,
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
}

func TestProcess(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = test.GetRecorder("GET", "/api/invalid", nil, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid request")
	w, r = test.GetRecorder("GET", "/api/invalid", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid request")
}

func TestAuthDisabledLogin(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	configuration.Get().Authentication.Method = models.AuthenticationDisabled
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	configuration.Get().Authentication.Method = models.AuthenticationInternal
}

func TestChangeFriendlyName(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid api key provided.")
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)

	key, ok := database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}, {
		Name: "friendlyName", Value: "NewName"}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	key, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "NewName")
	w = httptest.NewRecorder()
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)

	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: getNewKeyWithPermissionMissing(t, models.ApiPermApiMod).Id}, {
		Name: "apiKeyToModify", Value: "validkey"}, {
		Name: "friendlyName", Value: "NewName2"}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
}

func TestDeleteFile(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid file ID provided")
	w, r = test.GetRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "invalid",
	},
	}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid file ID provided")
	file, ok := database.GetMetaDataById("jpLXGJKigM4hjtA6T6sN2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Id, "jpLXGJKigM4hjtA6T6sN2")
	w, r = test.GetRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "jpLXGJKigM4hjtA6T6sN2",
	},
	}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	time.Sleep(time.Second)
	_, ok = database.GetMetaDataById("jpLXGJKigM4hjtA6T6sN2")
	test.IsEqualBool(t, ok, false)
}

func TestUploadAndDuplication(t *testing.T) {
	//
	// Upload
	//
	file, err := os.Open("test/fileupload.jpg")
	test.IsNil(t, err)
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	test.IsNil(t, err)
	io.Copy(part, file)
	writer.WriteField("allowedDownloads", "200")
	writer.WriteField("expiryDays", "10")
	writer.WriteField("password", "12345")
	writer.Close()
	w, r := test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	Process(w, r)
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	result := models.Result{}
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Result, "OK")
	test.IsEqualString(t, result.FileInfo.Size, "3 B")
	test.IsEqualInt(t, result.FileInfo.DownloadsRemaining, 200)
	test.IsEqualBool(t, result.FileInfo.IsPasswordProtected, true)
	test.IsEqualString(t, result.FileInfo.UrlDownload, "http://127.0.0.1:53843/d?id="+result.FileInfo.Id)
	newFileId := result.FileInfo.Id

	w, r = test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Content-Type isn't multipart/form-data")
	test.IsEqualInt(t, w.Code, 400)

	//
	// Duplication
	//
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"},
	}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid id provided.")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("§$§$%&(&//&/invalid"))
	Process(w, r)
	test.ResponseBodyContains(t, w, "invalid URL escape")

	data := url.Values{}
	data.Set("id", newFileId)

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication := models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualInt(t, resultDuplication.DownloadsRemaining, 200)
	test.IsEqualBool(t, resultDuplication.UnlimitedTime, false)
	test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, false)
	test.IsEqualInt(t, resultDuplication.DownloadCount, 0)
	test.IsEqualBool(t, resultDuplication.IsPasswordProtected, false)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("allowedDownloads", "100")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualInt(t, resultDuplication.DownloadsRemaining, 100)
	test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, false)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("allowedDownloads", "0")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, true)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("allowedDownloads", "invalid")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("expiryDays", "invalid")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("expiryDays", "20")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualBool(t, resultDuplication.UnlimitedTime, false)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("expiryDays", "0")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualBool(t, resultDuplication.UnlimitedTime, true)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("password", "")
	data.Set("originalPassword", "true")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualBool(t, resultDuplication.IsPasswordProtected, true)

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("password", "")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualBool(t, resultDuplication.IsPasswordProtected, false)
	test.IsEqualString(t, resultDuplication.Name, "fileupload.jpg")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("filename", "")
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualString(t, resultDuplication.Name, "fileupload.jpg")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("filename", "test.test")
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	resultDuplication = models.FileApiOutput{}
	response, err = io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &resultDuplication)
	test.IsNil(t, err)
	test.IsEqualString(t, resultDuplication.Name, "test.test")
}

func TestChunkUpload(t *testing.T) {
	err := os.WriteFile("test/tmpupload", []byte("chunktestfile"), 0600)
	test.IsNil(t, err)
	body, formcontent := test.FileToMultipartFormBody(t, test.HttpTestConfig{
		UploadFileName:  "test/tmpupload",
		UploadFieldName: "file",
		PostValues: []test.PostBody{{
			Key:   "filesize",
			Value: "13",
		}, {
			Key:   "offset",
			Value: "0",
		}, {
			Key:   "uuid",
			Value: "tmpupload123",
		}},
	})
	w, r := test.GetRecorder("POST", "/api/chunk/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	r.Header.Add("Content-Type", formcontent)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "OK")

	body, formcontent = test.FileToMultipartFormBody(t, test.HttpTestConfig{
		UploadFileName:  "test/tmpupload",
		UploadFieldName: "file",
		PostValues: []test.PostBody{{
			Key:   "dztotalfilesize",
			Value: "13",
		}, {
			Key:   "dzchunkbyteoffset",
			Value: "0",
		}, {
			Key:   "dzuuid",
			Value: "tmpupload123",
		}},
	})
	w, r = test.GetRecorder("POST", "/api/chunk/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	r.Header.Add("Content-Type", formcontent)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")
}

func TestChunkComplete(t *testing.T) {
	data := url.Values{}
	data.Set("uuid", "tmpupload123")
	data.Set("filename", "test.upload")
	data.Set("filesize", "13")

	w, r := test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	result := struct {
		FileInfo models.FileApiOutput `json:"FileInfo"`
	}{}
	response, err := io.ReadAll(w.Result().Body)
	fmt.Println(string(response))
	test.IsNil(t, err)
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.FileInfo.Name, "test.upload")

	data.Set("filesize", "15")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("invalid&&§$%"))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")
}

func TestList(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/files/list", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "picture.jpg")
}

func TestApiRequestToUploadRequest(t *testing.T) {
	_, r := test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}}, strings.NewReader("invalid&&§$%"))
	_, _, _, err := apiRequestToUploadRequest(r)
	test.IsNotNil(t, err)
}
