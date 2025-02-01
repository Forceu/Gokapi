package api

import (
	"bytes"
	"encoding/json"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
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
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	configuration.ConnectDatabase()
	generateTestData()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

const (
	idInvalidUser            = 99
	idSuperAdmin             = 100
	idAdmin                  = 101
	idUser                   = 102
	idApiKeyAdmin            = "ApiKeyAdmin"
	idApiKeySuperAdmin       = "ApiKeySuperAdmin"
	idPublicApiKeySuperAdmin = "OGeidahfiep1Akeevahkoh1quechieP6ael"
	idFileUser               = "newTestFile"
	idFileAdmin              = "otherTestFile"
)

func generateTestData() {
	newUser := models.User{
		Id:            idUser,
		Name:          "TestUser",
		Permissions:   models.UserPermissionNone,
		UserLevel:     models.UserLevelUser,
		ResetPassword: false,
	}
	newAdmin := models.User{
		Id:            idAdmin,
		Name:          "TestAdmin",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelAdmin,
		ResetPassword: false,
	}
	newSuperAdmin := models.User{
		Id:            idSuperAdmin,
		Name:          "TestSuperAdmin",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelSuperAdmin,
		ResetPassword: false,
	}
	database.SaveUser(newUser, false)
	database.SaveUser(newAdmin, false)
	database.SaveUser(newSuperAdmin, false)
	database.SaveApiKey(models.ApiKey{
		Id:           idApiKeyAdmin,
		PublicId:     idApiKeyAdmin,
		FriendlyName: "Admin",
		Permissions:  models.ApiPermAll,
		UserId:       idAdmin,
	})
	database.SaveApiKey(models.ApiKey{
		Id:           idApiKeySuperAdmin,
		PublicId:     idPublicApiKeySuperAdmin,
		FriendlyName: "SuperAdmin",
		Permissions:  models.ApiPermAll,
		UserId:       idSuperAdmin,
	})
	database.SaveMetaData(models.File{
		Id:                 idFileUser,
		Name:               idFileUser + "Name",
		SHA1:               "e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
		UserId:             idUser,
	})
	database.SaveMetaData(models.File{
		Id:                 idFileAdmin,
		Name:               idFileAdmin + "Name",
		SHA1:               "e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
		UserId:             idAdmin,
	})
}

func getRecorder(url, apikey string, headers []test.Header) (*httptest.ResponseRecorder, *http.Request) {
	return getRecorderWithBody(url, apikey, "GET", headers, nil)
}

func getRecorderPost(url, apikey string, headers []test.Header, body io.Reader) (*httptest.ResponseRecorder, *http.Request) {
	headers = append(headers, test.Header{Name: "Content-type", Value: "application/x-www-form-urlencoded"})
	return getRecorderWithBody(url, apikey, "POST", headers, body)
}

func getRecorderWithBody(url, apikey, method string, headers []test.Header, body io.Reader) (*httptest.ResponseRecorder, *http.Request) {
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
	return test.GetRecorder(method, url, nil, passedHeaders, body)
}

func testAuthorisation(t *testing.T, url string, requiredPermission models.ApiPermission) models.ApiKey {
	t.Helper()
	w, r := getRecorder(url, "", []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	w, r = getRecorder(url, "invalid", []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	newApiKeyUser := generateNewKey(false, idUser, "")
	w, r = getRecorder(url, newApiKeyUser.Id, []test.Header{{}})
	Process(w, r)
	test.IsEqualBool(t, w.Code != 200, true)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)

	for _, permission := range getAvailableApiPermissions(t) {
		if permission == requiredPermission {
			continue
		}
		setPermissionApikey(t, newApiKeyUser.Id, permission)
		w, r = getRecorder(url, newApiKeyUser.Id, []test.Header{{}})
		Process(w, r)
		test.IsEqualBool(t, w.Code != 200, true)
		test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Unauthorized"}`)
		removePermissionApikey(t, newApiKeyUser.Id, permission)
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

func testInvalidParameters(t *testing.T, url, apiKey string, correctHeaders []test.Header, headerName string, invalidValues []invalidParameterValue) {
	t.Helper()
	for _, invalidHeader := range invalidValues {
		headers := make([]test.Header, len(correctHeaders))
		copy(headers, correctHeaders)
		headers = append(headers, test.Header{
			Name:  headerName,
			Value: invalidHeader.Value,
		})
		w, r := getRecorderWithBody(url, apiKey, "GET", headers, nil)
		Process(w, r)
		test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
		test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		if invalidHeader.Value == "" {
			w, r = getRecorder(url, apiKey, correctHeaders)
			Process(w, r)
			test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
			test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		}
	}
}

func testInvalidUserId(t *testing.T, url, apiKey string) {
	t.Helper()
	const headerUserId = "userid"

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid user id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        strconv.Itoa(idInvalidUser),
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid user id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid request"}`,
			StatusCode:   400,
		},
		{
			Value:        strconv.Itoa(idUser),
			ErrorMessage: `{"Result":"error","ErrorMessage":"Cannot`,
			StatusCode:   400,
		},
		{
			Value:        strconv.Itoa(idSuperAdmin),
			ErrorMessage: `{"Result":"error","ErrorMessage":"Cannot`,
			StatusCode:   400,
		},
	}
	testInvalidParameters(t, url, apiKey, []test.Header{{}}, headerUserId, invalidParameter)
}

func testInvalidApiKey(t *testing.T, url, apiKey string) {
	t.Helper()
	const headerApiKey = "apiKeyToModify"

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid api key provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid api key provided."}`,
			StatusCode:   404,
		},
		{
			Value:        idApiKeySuperAdmin,
			ErrorMessage: `{"Result":"error","ErrorMessage":"No permission to `,
			StatusCode:   401,
		},
		{
			Value:        idPublicApiKeySuperAdmin,
			ErrorMessage: `{"Result":"error","ErrorMessage":"No permission to `,
			StatusCode:   401,
		},
		{
			Value:        idApiKeyAdmin,
			ErrorMessage: `{"Result":"error","ErrorMessage":"No permission to `,
			StatusCode:   401,
		},
	}
	testInvalidParameters(t, url, apiKey, []test.Header{{}}, headerApiKey, invalidParameter)
}

func testInvalidFileId(t *testing.T, url, apiKey string, isReplacingCall bool) {
	t.Helper()
	const headerId = "id"
	const headerIdReplace = "idNewContent"

	header := headerId
	if isReplacingCall {
		header = headerIdReplace
	}

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "invalidFile",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        idFileAdmin,
			ErrorMessage: `{"Result":"error","ErrorMessage":"No permission to `,
			StatusCode:   401,
		},
	}
	testInvalidParameters(t, url, apiKey, []test.Header{{}}, header, invalidParameter)
}

func TestInvalidRouting(t *testing.T) {

	const apiUrl = "/invalid"
	w, r := getRecorder(apiUrl, "invalid", []test.Header{{}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"Invalid request"}`)
}

// ## /user/##

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
	testInvalidParameters(t, apiUrl, apiKey.Id, []test.Header{{}}, headerUsername, invalidParameter)
}

func TestUserChangeRank(t *testing.T) {
	const apiUrl = "/user/changeRank"
	const headerUserId = "userid"
	const headerNewRank = "newRank"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id)
	var validHeaders = []test.Header{
		{
			Name:  headerUserId,
			Value: strconv.Itoa(idAdmin),
		},
	}
	invalidParameter := []invalidParameterValue{
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
	testInvalidParameters(t, apiUrl, apiKey.Id, validHeaders, headerNewRank, invalidParameter)

	user, ok := database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelAdmin)
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(idAdmin),
	}, {
		Name:  headerNewRank,
		Value: "USER",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelUser)
	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(idAdmin),
	}, {
		Name:  headerNewRank,
		Value: "ADMIN",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, user.UserLevel, models.UserLevelAdmin)
}

func TestUserDelete(t *testing.T) {
	const apiUrl = "/user/delete"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id)
	testDeleteUserCall(t, apiKey.Id, false)
	testDeleteUserCall(t, apiKey.Id, true)
}

func testDeleteUserCall(t *testing.T, apiKey string, testDeleteFiles bool) {
	const apiUrl = "/user/delete"
	const headerUserId = "userid"
	const headerDeleteFiles = "deleteFiles"

	user := models.User{
		Name:      "ToDelete",
		UserLevel: models.UserLevelAdmin,
	}
	database.SaveUser(user, true)
	retrievedUser, ok := database.GetUserByName("ToDelete")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedUser.Id != idUser, true)
	session := models.Session{
		RenewAt:    2147483645,
		ValidUntil: 2147483645,
		UserId:     retrievedUser.Id,
	}
	database.SaveSession("sessionApiDelete", session)
	_, ok = database.GetSession("sessionApiDelete")
	test.IsEqualBool(t, ok, true)
	userApiKey := generateNewKey(false, retrievedUser.Id, "")
	_, ok = database.GetApiKey(userApiKey.Id)
	test.IsEqualBool(t, ok, true)
	testFile := models.File{
		Id:                 "testFileApiDelete",
		Name:               "testFileApiDelete",
		UserId:             retrievedUser.Id,
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	database.SaveMetaData(testFile)
	testFile, ok = database.GetMetaDataById("testFileApiDelete")
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, testFile.UserId, retrievedUser.Id)

	deleteMetaFile := "invalid"
	if testDeleteFiles {
		deleteMetaFile = "true"
	}

	w, r := getRecorder(apiUrl, apiKey, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(retrievedUser.Id),
	}, {
		Name:  headerDeleteFiles,
		Value: deleteMetaFile,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	_, ok = database.GetUser(retrievedUser.Id)
	test.IsEqualBool(t, ok, false)
	_, ok = database.GetSession("sessionApiDelete")
	test.IsEqualBool(t, ok, false)
	_, ok = database.GetApiKey(userApiKey.Id)
	test.IsEqualBool(t, ok, false)
	testFile, ok = database.GetMetaDataById("testFileApiDelete")
	test.IsEqualBool(t, ok, !testDeleteFiles)
	if !testDeleteFiles {
		test.IsEqualInt(t, testFile.UserId, idUser)
	}
}

func TestUserModify(t *testing.T) {
	const apiUrl = "/user/modify"
	const headerUserId = "userid"
	const headerPermission = "userpermission"
	const idNewKey = "idNewKey"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id)

	var validHeaders = []test.Header{
		{
			Name:  headerUserId,
			Value: strconv.Itoa(idAdmin),
		},
	}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "PERM_REPLACEE",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey.Id, validHeaders, headerPermission, invalidParameter)

	user := models.User{
		Name:        "ToModify",
		UserLevel:   models.UserLevelAdmin,
		Permissions: models.UserPermissionNone,
	}
	database.SaveUser(user, true)
	retrievedUser, ok := database.GetUserByName("ToModify")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedUser.Id != idUser, true)
	systemKeyId := GetSystemKey(retrievedUser.Id)
	systemKey, ok := database.GetApiKey(systemKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, systemKey.HasPermissionReplace(), false)
	test.IsEqualBool(t, systemKey.HasPermissionManageUsers(), false)

	for permissionUint, permissionString := range getUserPermMap(t) {
		test.IsEqualBool(t, retrievedUser.HasPermission(permissionUint), false)
		testUserModifyCall(t, apiKey.Id, retrievedUser.Id, permissionString, true)
		retrievedUser, ok = database.GetUserByName("ToModify")
		test.IsEqualBool(t, ok, true)
		test.IsEqualBool(t, retrievedUser.HasPermission(permissionUint), true)
		if permissionUint == models.UserPermReplaceUploads || permissionUint == models.UserPermManageUsers {
			affectedPermission := getAffectedApiPerm(t, permissionUint)
			systemKey, ok = database.GetApiKey(systemKeyId)
			test.IsEqualBool(t, ok, true)
			test.IsEqualBool(t, systemKey.HasPermission(affectedPermission), true)
			key := models.ApiKey{
				Id:          idNewKey,
				PublicId:    idNewKey,
				Permissions: models.ApiPermNone,
				UserId:      retrievedUser.Id,
			}
			key.GrantPermission(affectedPermission)
			database.SaveApiKey(key)
			newKey, ok := database.GetApiKey(idNewKey)
			test.IsEqualBool(t, ok, true)
			test.IsEqualBool(t, newKey.HasPermission(affectedPermission), true)
		}

		testUserModifyCall(t, apiKey.Id, retrievedUser.Id, permissionString, false)
		retrievedUser, ok = database.GetUserByName("ToModify")
		test.IsEqualBool(t, ok, true)
		test.IsEqualBool(t, retrievedUser.HasPermission(permissionUint), false)
		if permissionUint == models.UserPermReplaceUploads || permissionUint == models.UserPermManageUsers {
			affectedPermission := getAffectedApiPerm(t, permissionUint)
			newKey, ok := database.GetApiKey(idNewKey)
			test.IsEqualBool(t, ok, true)
			test.IsEqualBool(t, newKey.HasPermission(affectedPermission), false)
			systemKey, ok = database.GetApiKey(systemKeyId)
			test.IsEqualBool(t, systemKey.HasPermission(affectedPermission), false)
		}
	}
	database.DeleteApiKey(systemKeyId)
}

func getAffectedApiPerm(t *testing.T, permission models.UserPermission) models.ApiPermission {
	switch permission {
	case models.UserPermManageUsers:
		return models.ApiPermManageUsers
	case models.UserPermReplaceUploads:
		return models.ApiPermReplace
	default:
		t.Errorf("Invalid permission %d", permission)
		return models.ApiPermNone
	}
}

func TestUserPasswordReset(t *testing.T) {
	const apiUrl = "/user/resetPassword"
	const headerUserId = "userid"
	const headerSetNewPw = "generateNewPassword"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id)
	user, ok := database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, user.ResetPassword, false)
	user.Password = "1234"
	database.SaveUser(user, false)
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(idAdmin),
	}, {
		Name:  headerSetNewPw,
		Value: "false",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, user.ResetPassword, true)
	test.IsEqualString(t, user.Password, "1234")
	test.ResponseBodyContains(t, w, `{"Result":"ok","password":""}`)

	user.ResetPassword = false
	database.SaveUser(user, false)
	user, ok = database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, user.ResetPassword, false)

	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(idAdmin),
	}, {
		Name:  headerSetNewPw,
		Value: "true",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	user, ok = database.GetUser(idAdmin)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, user.ResetPassword, true)
	test.IsEqualBool(t, user.Password != "1234", true)
	type response struct {
		Result   string `json:"Result"`
		Password string `json:"password"`
	}
	var resp response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	test.IsNil(t, err)
	test.IsEqualString(t, resp.Result, "ok")
	test.IsNotEmpty(t, resp.Password)
}

func testUserModifyCall(t *testing.T, apiKey string, userId int, permission string, grant bool) {
	const apiUrl = "/user/modify"
	const headerUserId = "userid"
	const headerPermission = "userpermission"
	const headerPermModifier = "permissionModifier"

	modifier := "REVOKE"
	if grant {
		modifier = "GRANT"
	}
	w, r := getRecorder(apiUrl, apiKey, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(userId),
	}, {
		Name:  headerPermission,
		Value: permission,
	}, {
		Name:  headerPermModifier,
		Value: modifier,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
}

// ## /auth ##

func TestNewApiKey(t *testing.T) {
	const apiUrl = "/auth/create"
	const headerFriendlyName = "friendlyName"
	const headerDefaultPerm = "basicPermissions"

	const (
		testNoParam = iota
		testFriendlyName
		testBasicPermission
		testBoth
	)

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	keysBefore := countApiKeys()

	for i := testNoParam; i <= testBoth; i++ {
		friendlyName := "Unnamed key"
		expectedPermissions := models.ApiPermNone
		var headers []test.Header
		if i == testFriendlyName || i == testBoth {
			friendlyName = helper.GenerateRandomString(40)
			headers = append(headers, test.Header{
				Name:  headerFriendlyName,
				Value: friendlyName,
			})
		}
		if i == testBasicPermission || i == testBoth {
			headers = append(headers, test.Header{
				Name:  headerDefaultPerm,
				Value: "true",
			})
			expectedPermissions = models.ApiPermDefault
		}
		w, r := getRecorder(apiUrl, apiKey.Id, headers)
		Process(w, r)
		test.IsEqualInt(t, w.Code, 200)
		var response models.ApiKeyOutput
		err := json.Unmarshal(w.Body.Bytes(), &response)
		test.IsNil(t, err)
		test.IsEqualString(t, response.Result, "OK")
		test.IsNotEmpty(t, response.Id)
		test.IsNotEmpty(t, response.PublicId)
		test.IsEqualBool(t, response.PublicId != response.Id, true)
		retrievedKey, ok := database.GetApiKey(response.Id)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, response.PublicId, retrievedKey.PublicId)
		test.IsEqualString(t, retrievedKey.FriendlyName, friendlyName)
		test.IsEqualInt(t, countApiKeys(), keysBefore+i+1)
		test.IsEqual(t, retrievedKey.Permissions, expectedPermissions)
	}
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

	newApiKey := generateNewKey(false, 5, "")
	user, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	for _, permission := range getAvailableApiPermissions(t) {
		_, isValid = isValidApiKey(newApiKey.Id, true, permission)
		test.IsEqualBool(t, isValid, false)
	}
	for _, newPermission := range getAvailableApiPermissions(t) {
		setPermissionApikey(t, newApiKey.Id, newPermission)
		for _, permission := range getAvailableApiPermissions(t) {
			_, isValid = isValidApiKey(newApiKey.Id, true, permission)
			test.IsEqualBool(t, isValid, permission == newPermission)
		}
		removePermissionApikey(t, newApiKey.Id, newPermission)
	}
	setPermissionApikey(t, newApiKey.Id, models.ApiPermEdit|models.ApiPermDelete)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermEdit)
	test.IsEqualBool(t, isValid, true)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermAll)
	test.IsEqualBool(t, isValid, false)
	_, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermView)
	test.IsEqualBool(t, isValid, false)
}

func setPermissionApikey(t *testing.T, key string, newPermission models.ApiPermission) {
	apiKey, ok := database.GetApiKey(key)
	test.IsEqualBool(t, ok, true)
	apiKey.GrantPermission(newPermission)
	database.SaveApiKey(apiKey)
}
func removePermissionApikey(t *testing.T, key string, newPermission models.ApiPermission) {
	apiKey, ok := database.GetApiKey(key)
	test.IsEqualBool(t, ok, true)
	apiKey.RemovePermission(newPermission)
	database.SaveApiKey(apiKey)
}

func getAvailableApiPermissions(t *testing.T) []models.ApiPermission {
	result := []models.ApiPermission{
		models.ApiPermView,
		models.ApiPermUpload,
		models.ApiPermDelete,
		models.ApiPermApiMod,
		models.ApiPermEdit,
		models.ApiPermReplace,
		models.ApiPermManageUsers}
	sum := 0
	for _, perm := range result {
		sum = sum + int(perm)
	}
	if sum != int(models.ApiPermAll) {
		t.Fatal("List of permissions are incorrect")
	}
	return result
}

func getApiPermMap(t *testing.T) map[models.ApiPermission]string {
	result := make(map[models.ApiPermission]string)
	result[models.ApiPermView] = "PERM_VIEW"
	result[models.ApiPermUpload] = "PERM_UPLOAD"
	result[models.ApiPermDelete] = "PERM_DELETE"
	result[models.ApiPermApiMod] = "PERM_API_MOD"
	result[models.ApiPermEdit] = "PERM_EDIT"
	result[models.ApiPermReplace] = "PERM_REPLACE"
	result[models.ApiPermManageUsers] = "PERM_MANAGE_USERS"

	sum := 0
	for perm, _ := range result {
		sum = sum + int(perm)
	}
	if sum != int(models.ApiPermAll) {
		t.Fatal("List of permissions are incorrect")
	}

	return result
}

func getUserPermMap(t *testing.T) map[models.UserPermission]string {
	result := make(map[models.UserPermission]string)
	result[models.UserPermReplaceUploads] = "PERM_REPLACE"
	result[models.UserPermListOtherUploads] = "PERM_LIST"
	result[models.UserPermEditOtherUploads] = "PERM_EDIT"
	result[models.UserPermReplaceOtherUploads] = "PERM_REPLACE_OTHER"
	result[models.UserPermDeleteOtherUploads] = "PERM_DELETE"
	result[models.UserPermManageLogs] = "PERM_LOGS"
	result[models.UserPermManageApiKeys] = "PERM_API"
	result[models.UserPermManageUsers] = "PERM_USERS"

	sum := 0
	for perm, _ := range result {
		sum = sum + int(perm)
	}
	if sum != int(models.UserPermissionAll) {
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
	GetSystemKey(idInvalidUser)
}

func grantUserPermission(t *testing.T, userId int, permission models.UserPermission) {
	user, ok := database.GetUser(userId)
	test.IsEqualBool(t, ok, true)
	user.GrantPermission(permission)
	database.SaveUser(user, false)
}
func removeUserPermission(t *testing.T, userId int, permission models.UserPermission) {
	user, ok := database.GetUser(userId)
	test.IsEqualBool(t, ok, true)
	user.RemovePermission(permission)
	database.SaveUser(user, false)
}

func TestDeleteApiKey(t *testing.T) {
	const apiUrl = "/auth/delete"
	const headerApiDelete = "apiKeyToModify"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id)

	database.SaveApiKey(models.ApiKey{
		Id:       "toDelete",
		PublicId: "toDelete",
		UserId:   idUser,
	})
	_, ok := database.GetApiKey("toDelete")
	test.IsEqualBool(t, ok, true)
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerApiDelete,
		Value: apiKey.Id,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	_, ok = database.GetApiKey(apiKey.Id)
	test.IsEqualBool(t, ok, false)

}

func countApiKeys() int {
	return len(database.GetAllApiKeys())
}

func TestChangeFriendlyName(t *testing.T) {
	const apiUrl = "/auth/friendlyname"
	const headerApiKeyModify = "apiKeyToModify"
	const headerNewName = "friendlyName"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id)
	test.IsEqualString(t, apiKey.FriendlyName, "Unnamed key")
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerApiKeyModify,
		Value: apiKey.Id,
	}, {
		Name:  headerNewName,
		Value: "New name for the key",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	key, ok := database.GetApiKey(apiKey.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "New name for the key")

	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerApiKeyModify,
		Value: apiKey.Id,
	}, {
		Name:  headerNewName,
		Value: "",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	key, ok = database.GetApiKey(apiKey.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
}

func TestApikeyModify(t *testing.T) {
	const apiUrl = "/auth/modify"
	const headerApiKeyModify = "apiKeyToModify"
	const headerPermission = "permission"
	const headerModifier = "permissionModifier"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id)

	newApiKey := models.ApiKey{
		Id:           "modifyTest",
		PublicId:     "modifyTest",
		FriendlyName: "modifyTest",
		UserId:       idUser,
	}
	database.SaveApiKey(newApiKey)
	retrievedApiKey, ok := database.GetApiKey("modifyTest")
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, retrievedApiKey.Permissions, models.ApiPermNone)

	var validHeaders = []test.Header{
		{
			Name:  headerApiKeyModify,
			Value: retrievedApiKey.Id,
		},
		{
			Name:  headerModifier,
			Value: "GRANT",
		},
	}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "PERM_VIEWW",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
		{
			Value:        "PERM_REPLACE",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Insufficient user permission for owner to set this API permission"}`,
			StatusCode:   401,
		},
		{
			Value:        "PERM_MANAGE_USERS",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Insufficient user permission for owner to set this API permission"}`,
			StatusCode:   401,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey.Id, validHeaders, headerPermission, invalidParameter)

	grantUserPermission(t, idUser, models.UserPermReplaceUploads)
	grantUserPermission(t, idUser, models.UserPermManageUsers)

	for permissionUint, permissionString := range getApiPermMap(t) {
		test.IsEqualBool(t, retrievedApiKey.HasPermission(permissionUint), false)
		testApiModifyCall(t, apiKey.Id, retrievedApiKey.Id, permissionString, true)
		retrievedApiKey, ok = database.GetApiKey("modifyTest")
		test.IsEqualBool(t, ok, true)
		test.IsEqualBool(t, retrievedApiKey.HasPermission(permissionUint), true)
		testApiModifyCall(t, apiKey.Id, retrievedApiKey.Id, permissionString, false)
		retrievedApiKey, ok = database.GetApiKey("modifyTest")
		test.IsEqualBool(t, ok, true)
		test.IsEqualBool(t, retrievedApiKey.HasPermission(permissionUint), false)
	}
	removeUserPermission(t, idUser, models.UserPermReplaceUploads)
	removeUserPermission(t, idUser, models.UserPermManageUsers)
}

func testApiModifyCall(t *testing.T, apiKey, apiKeyToModify string, permission string, grant bool) {
	const apiUrl = "/auth/modify"
	const headerApiKeyModify = "apiKeyToModify"
	const headerPermission = "permission"
	const headerModifier = "permissionModifier"

	modifier := "REVOKE"
	if grant {
		modifier = "GRANT"
	}
	w, r := getRecorder(apiUrl, apiKey, []test.Header{{
		Name:  headerApiKeyModify,
		Value: apiKeyToModify,
	}, {
		Name:  headerPermission,
		Value: permission,
	}, {
		Name:  headerModifier,
		Value: modifier,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
}

// ## /files ##

func TestDeleteFile(t *testing.T) {
	database.SaveMetaData(models.File{
		Id:                 "smalltestfile1",
		Name:               "smalltestfile1",
		SHA1:               "smalltestfile1",
		ExpireAt:           2147483646,
		DownloadsRemaining: 1,
		UserId:             idUser,
	})
	database.SaveMetaData(models.File{
		Id:                 "smalltestfile2",
		Name:               "smalltestfile2",
		SHA1:               "smalltestfile2",
		ExpireAt:           2147483646,
		DownloadsRemaining: 1,
		UserId:             idSuperAdmin,
	})
	_, ok := database.GetMetaDataById("smalltestfile1")
	test.IsEqualBool(t, ok, true)
	_, ok = database.GetMetaDataById("smalltestfile2")
	test.IsEqualBool(t, ok, true)

	apiKey := testAuthorisation(t, "/files/delete", models.ApiPermDelete)
	testDeleteFileCall(t, apiKey.Id, "", 404, `{"Result":"error","ErrorMessage":"Invalid file ID provided."}`)
	testDeleteFileCall(t, apiKey.Id, "invalid", 404, `{"Result":"error","ErrorMessage":"Invalid file ID provided."}`)
	testDeleteFileCall(t, apiKey.Id, "smalltestfile1", 200, "")
	testDeleteFileCall(t, apiKey.Id, "smalltestfile2", 401, `{"Result":"error","ErrorMessage":"No permission to delete this file"}`)
	_, ok = database.GetMetaDataById("smalltestfile2")
	test.IsEqualBool(t, ok, true)
	grantUserPermission(t, idUser, models.UserPermDeleteOtherUploads)
	testDeleteFileCall(t, apiKey.Id, "smalltestfile2", 200, "")
	removeUserPermission(t, idUser, models.UserPermDeleteOtherUploads)
	time.Sleep(200 * time.Millisecond)
	_, ok = database.GetMetaDataById("smalltestfile1")
	test.IsEqualBool(t, ok, false)
	_, ok = database.GetMetaDataById("smalltestfile2")
	test.IsEqualBool(t, ok, false)
}

func testDeleteFileCall(t *testing.T, apiKey, fileId string, resultCode int, expectedResponse string) {
	t.Helper()
	const apiUrl = "/files/delete"
	const headerFileId = "id"
	headers := []test.Header{{}}
	if fileId != "" {
		headers = append(headers, test.Header{Name: headerFileId, Value: fileId})
	}
	w, r := getRecorder(apiUrl, apiKey, headers)
	Process(w, r)
	test.IsEqualInt(t, w.Code, resultCode)
	if expectedResponse != "" {
		test.ResponseBodyContains(t, w, expectedResponse)
	}
}

func TestList(t *testing.T) {
	const apiUrl = "/files/list"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermView)

	database.DeleteMetaData(idFileUser)
	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "null")
	generateTestData()

	var result []models.FileApiOutput
	grantUserPermission(t, idUser, models.UserPermListOtherUploads)
	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	err := json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)
	test.IsEqualInt(t, len(result), 11)

	removeUserPermission(t, idUser, models.UserPermListOtherUploads)
	w, r = getRecorder(apiUrl, apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	err = json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)
	test.IsEqualInt(t, len(result), 1)
	test.IsEqualString(t, result[0].Name, "newTestFileName")
}

func TestListSingle(t *testing.T) {
	const apiUrl = "/files/list/"
	_ = testAuthorisation(t, apiUrl, models.ApiPermView)
	apiKey := testAuthorisation(t, apiUrl+"newTestFile", models.ApiPermView)
	var result models.FileApiOutput

	w, r := getRecorder(apiUrl+"newTestFile", apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	err := json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Name, "newTestFileName")

	w, r = getRecorder(apiUrl+"e4TjE7CokWK0giiLNxDL", apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 401)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"No permission to view file"}`)
	w, r = getRecorder(apiUrl+"invalid", apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 404)
	test.ResponseBodyContains(t, w, `{"Result":"error","ErrorMessage":"File not found"}`)

	grantUserPermission(t, idUser, models.UserPermListOtherUploads)
	w, r = getRecorder(apiUrl+"e4TjE7CokWK0giiLNxDL", apiKey.Id, []test.Header{})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	err = json.Unmarshal(w.Body.Bytes(), &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Id, "e4TjE7CokWK0giiLNxDL")
	removeUserPermission(t, idUser, models.UserPermListOtherUploads)
}

func TestUploadAndDuplication(t *testing.T) {
	// Upload
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
	// newFileId := result.FileInfo.Id
	w, r = test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Content-Type isn't multipart/form-data")
	test.IsEqualInt(t, w.Code, 400)
	//
	// // Duplication
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{{
	// 	Name:  "apikey",
	// 	Value: "validkey",
	// }}, nil)
	// Process(w, r)
	// test.ResponseBodyContains(t, w, "Invalid id provided.")
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"},
	// }, nil)
	// Process(w, r)
	// test.ResponseBodyContains(t, w, "Invalid id provided.")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader("§$§$%&(&//&/invalid"))
	// Process(w, r)
	// test.ResponseBodyContains(t, w, "invalid URL escape")
	//
	// data := url.Values{}
	// data.Set("id", newFileId)
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication := models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualInt(t, resultDuplication.DownloadsRemaining, 200)
	// test.IsEqualBool(t, resultDuplication.UnlimitedTime, false)
	// test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, false)
	// test.IsEqualInt(t, resultDuplication.DownloadCount, 0)
	// test.IsEqualBool(t, resultDuplication.IsPasswordProtected, false)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("allowedDownloads", "100")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualInt(t, resultDuplication.DownloadsRemaining, 100)
	// test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, false)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("allowedDownloads", "0")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualBool(t, resultDuplication.UnlimitedDownloads, true)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("allowedDownloads", "invalid")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("expiryDays", "invalid")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("expiryDays", "20")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualBool(t, resultDuplication.UnlimitedTime, false)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("expiryDays", "0")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "id", Value: "invalid"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualBool(t, resultDuplication.UnlimitedTime, true)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("password", "")
	// data.Set("originalPassword", "true")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualBool(t, resultDuplication.IsPasswordProtected, true)
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("password", "")
	//
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualBool(t, resultDuplication.IsPasswordProtected, false)
	// test.IsEqualString(t, resultDuplication.Name, "fileupload.jpg")
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("filename", "")
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualString(t, resultDuplication.Name, "fileupload.jpg")
	//
	// data = url.Values{}
	// data.Set("id", newFileId)
	// data.Set("filename", "test.test")
	// w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
	// 	{Name: "apikey", Value: "validkey"},
	// 	{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
	// 	strings.NewReader(data.Encode()))
	// Process(w, r)
	// resultDuplication = models.FileApiOutput{}
	// response, err = io.ReadAll(w.Result().Body)
	// test.IsNil(t, err)
	// err = json.Unmarshal(response, &resultDuplication)
	// test.IsNil(t, err)
	// test.IsEqualString(t, resultDuplication.Name, "test.test")
}

func TestDuplicate(t *testing.T) {
	const apiUrl = "/files/duplicate"
	const headerId = "id"
	const headerAllowedDownloads = "allowedDownloads"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermUpload)
	testInvalidFileId(t, apiUrl, apiKey.Id, false)
	testInvalidForm(t, apiUrl, apiKey.Id)

	validHeader := []test.Header{{Name: headerId, Value: idFileUser}}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid permission sent"}`,
			StatusCode:   400,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey.Id, validHeader, headerAllowedDownloads, invalidParameter)
}

func testInvalidForm(t *testing.T, apiUrl, apiKey string) {
	w, r := getRecorderPost(apiUrl, apiKey, []test.Header{},
		strings.NewReader("§$§$%&(&//&/invalid"))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)

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
		{Name: "apikey", Value: "validkey"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	result := struct {
		FileInfo models.FileApiOutput `json:"FileInfo"`
	}{}
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.FileInfo.Name, "test.upload")

	data.Set("filesize", "15")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"}},
		strings.NewReader(data.Encode()))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"}},
		strings.NewReader("invalid&&§$%"))
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")
}

func TestApiRequestToUploadRequest(t *testing.T) {
	_, r := test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{}, strings.NewReader("invalid&&§$%"))
	_, _, _, err := apiRequestToUploadRequest(r)
	test.IsNotNil(t, err)
}
