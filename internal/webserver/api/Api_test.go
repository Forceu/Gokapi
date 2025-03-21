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
	"os"
	"path/filepath"
	"strconv"
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

func testInvalidParameters(t *testing.T, url, apiKey string, validHeaders []test.Header, headerName string, invalidValues []invalidParameterValue) {
	t.Helper()
	for _, invalidHeader := range invalidValues {
		headers := make([]test.Header, len(validHeaders))
		copy(headers, validHeaders)
		headers = append(headers, test.Header{
			Name:  headerName,
			Value: invalidHeader.Value,
		})
		w, r := getRecorderWithBody(url, apiKey, "GET", headers, nil)
		Process(w, r)
		test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
		test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		if invalidHeader.Value == "" {
			w, r = getRecorder(url, apiKey, validHeaders)
			Process(w, r)
			test.IsEqualInt(t, w.Code, invalidHeader.StatusCode)
			test.ResponseBodyContains(t, w, invalidHeader.ErrorMessage)
		}
	}
}

func testInvalidUserId(t *testing.T, url, apiKey string, validHeaders []test.Header) {
	t.Helper()
	const headerUserId = "userid"

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"header userid is required"}`,
			StatusCode:   400,
		},
		{
			Value:        strconv.Itoa(idInvalidUser),
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid user id provided."}`,
			StatusCode:   404,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid value in header userid supplied"}`,
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
	testInvalidParameters(t, url, apiKey, validHeaders, headerUserId, invalidParameter)
}

func testInvalidApiKey(t *testing.T, url, apiKey string, validHeaders []test.Header) {
	t.Helper()
	const headerApiKey = "targetKey"

	var invalidParameter = []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"header targetKey is required"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid key ID provided."}`,
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
	testInvalidParameters(t, url, apiKey, validHeaders, headerApiKey, invalidParameter)
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
			ErrorMessage: `{"Result":"error","ErrorMessage":"header id is required"}`,
			StatusCode:   400,
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
			ErrorMessage: `{"Result":"error","ErrorMessage":"header username is required"}`,
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

	defer test.ExpectPanic(t)
	apiCreateUser(w, &paramAuthCreate{}, models.User{Id: 7})
}

func TestUserChangeRank(t *testing.T) {
	const apiUrl = "/user/changeRank"
	const headerUserId = "userid"
	const headerNewRank = "newRank"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id, []test.Header{{Name: headerNewRank, Value: "admin"}})
	var validHeaders = []test.Header{
		{
			Name:  headerUserId,
			Value: strconv.Itoa(idAdmin),
		},
	}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"header newRank is required"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid rank"}`,
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

	defer test.ExpectPanic(t)
	apiChangeUserRank(w, &paramAuthCreate{}, models.User{Id: 7})
}

func TestUserDelete(t *testing.T) {
	const apiUrl = "/user/delete"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id, []test.Header{})
	testDeleteUserCall(t, apiKey.Id, deleteUserCallModeDeleteFiles)
	testDeleteUserCall(t, apiKey.Id, deleteUserCallModeKeepFiles)
	testDeleteUserCall(t, apiKey.Id, deleteUserCallModeInvalidOperator)

	defer test.ExpectPanic(t)
	apiDeleteUser(nil, &paramAuthCreate{}, models.User{Id: 7})
}

const (
	deleteUserCallModeDeleteFiles     = iota
	deleteUserCallModeKeepFiles       = iota
	deleteUserCallModeInvalidOperator = iota
)

func testDeleteUserCall(t *testing.T, apiKey string, mode int) {
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

	var deleteMetaFile string
	switch mode {
	case deleteUserCallModeDeleteFiles:
		deleteMetaFile = "true"
	case deleteUserCallModeKeepFiles:
		deleteMetaFile = "false"
	case deleteUserCallModeInvalidOperator:
		deleteMetaFile = "invalid"
	}

	w, r := getRecorder(apiUrl, apiKey, []test.Header{{
		Name:  headerUserId,
		Value: strconv.Itoa(retrievedUser.Id),
	}, {
		Name:  headerDeleteFiles,
		Value: deleteMetaFile,
	}})
	Process(w, r)

	if mode == deleteUserCallModeInvalidOperator {
		test.IsEqualInt(t, w.Code, 400)
	} else {
		test.IsEqualInt(t, w.Code, 200)
		_, ok = database.GetUser(retrievedUser.Id)
		test.IsEqualBool(t, ok, false)
		_, ok = database.GetSession("sessionApiDelete")
		test.IsEqualBool(t, ok, false)
		_, ok = database.GetApiKey(userApiKey.Id)
		test.IsEqualBool(t, ok, false)
		testFile, ok = database.GetMetaDataById("testFileApiDelete")
		test.IsEqualBool(t, ok, mode == deleteUserCallModeKeepFiles)
		if mode == deleteUserCallModeKeepFiles {
			test.IsEqualBool(t, ok, true)
			test.IsEqualInt(t, testFile.UserId, idUser)
		} else {
			test.IsEqualBool(t, ok, false)
		}
	}

}

func TestUserModify(t *testing.T) {
	const apiUrl = "/user/modify"
	const headerUserId = "userid"
	const headerPermission = "userpermission"
	const headerModifier = "permissionModifier"
	const idNewKey = "idNewKey"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermManageUsers)
	testInvalidUserId(t, apiUrl, apiKey.Id, []test.Header{{Name: headerPermission, Value: "PERM_REPLACE"}, {Name: headerModifier, Value: "GRANT"}})

	var validHeaders = []test.Header{
		{
			Name:  headerUserId,
			Value: strconv.Itoa(idAdmin),
		}, {
			Name:  headerModifier,
			Value: "GRANT",
		},
	}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"header userpermission is required"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid permission"}`,
			StatusCode:   400,
		},
		{
			Value:        "PERM_REPLACEE",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid permission"}`,
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

	defer test.ExpectPanic(t)
	apiModifyUser(nil, &paramAuthCreate{}, models.User{Id: 7})
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
	testInvalidUserId(t, apiUrl, apiKey.Id, []test.Header{})
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

	defer test.ExpectPanic(t)
	apiResetPassword(w, &paramAuthCreate{}, models.User{Id: 7})
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

	defer test.ExpectPanic(t)
	apiCreateApiKey(nil, &paramUserCreate{}, models.User{Id: 7})
}

func TestIsValidApiKey(t *testing.T) {
	user, apiKey, isValid := isValidApiKey("", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, false)
	_, _, isValid = isValidApiKey("invalid", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, false)
	user, apiKey, isValid = isValidApiKey("validkey", false, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	test.IsEqualString(t, apiKey.Id, "validkey")
	test.IsEqualInt(t, user.Id, 5)
	key, ok := database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, true)
	user, apiKey, isValid = isValidApiKey("validkey", true, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	test.IsEqualInt(t, user.Id, 5)
	test.IsEqualString(t, apiKey.Id, "validkey")
	key, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, false)

	newApiKey := generateNewKey(false, 5, "")
	user, _, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermNone)
	test.IsEqualBool(t, isValid, true)
	for _, permission := range getAvailableApiPermissions(t) {
		_, _, isValid = isValidApiKey(newApiKey.Id, true, permission)
		test.IsEqualBool(t, isValid, false)
	}
	for _, newPermission := range getAvailableApiPermissions(t) {
		setPermissionApikey(t, newApiKey.Id, newPermission)
		for _, permission := range getAvailableApiPermissions(t) {
			_, _, isValid = isValidApiKey(newApiKey.Id, true, permission)
			test.IsEqualBool(t, isValid, permission == newPermission)
		}
		removePermissionApikey(t, newApiKey.Id, newPermission)
	}
	setPermissionApikey(t, newApiKey.Id, models.ApiPermEdit|models.ApiPermDelete)
	_, _, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermEdit)
	test.IsEqualBool(t, isValid, true)
	_, _, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermAll)
	test.IsEqualBool(t, isValid, false)
	_, _, isValid = isValidApiKey(newApiKey.Id, true, models.ApiPermView)
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
		models.ApiPermManageUsers,
		models.ApiPermManageLogs}
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
	result[models.ApiPermManageLogs] = "PERM_MANAGE_LOGS"

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
	const headerApiDelete = "targetKey"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id, []test.Header{})

	database.SaveApiKey(models.ApiKey{
		Id:       "toDelete",
		PublicId: "toDelete",
		UserId:   idUser,
	})
	_, ok := database.GetApiKey("toDelete")
	test.IsEqualBool(t, ok, true)

	invalidParameter := []invalidParameterValue{
		{
			Value:        "",
			ErrorMessage: `{"Result":"error","ErrorMessage":"header targetKey is required"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"Invalid key ID provided."}`,
			StatusCode:   404,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey.Id, []test.Header{}, headerApiDelete, invalidParameter)
	_, ok = database.GetApiKey(apiKey.Id)
	test.IsEqualBool(t, ok, true)

	w, r := getRecorder(apiUrl, apiKey.Id, []test.Header{{
		Name:  headerApiDelete,
		Value: apiKey.Id,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	_, ok = database.GetApiKey(apiKey.Id)
	test.IsEqualBool(t, ok, false)

	defer test.ExpectPanic(t)
	apiDeleteKey(w, &paramAuthCreate{}, models.User{Id: 7})
}

func countApiKeys() int {
	return len(database.GetAllApiKeys())
}

func TestChangeFriendlyName(t *testing.T) {
	const apiUrl = "/auth/friendlyname"
	const headerApiKeyModify = "targetKey"
	const headerNewName = "friendlyName"
	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id, []test.Header{{Name: headerNewName, Value: "new name"}})
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
	test.IsEqualInt(t, w.Code, 400)

	defer test.ExpectPanic(t)
	apiChangeFriendlyName(w, &paramAuthCreate{}, models.User{Id: 7})
}

func TestApikeyModify(t *testing.T) {
	const apiUrl = "/auth/modify"
	const headerApiKeyModify = "targetKey"
	const headerPermission = "permission"
	const headerModifier = "permissionModifier"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermApiMod)
	testInvalidApiKey(t, apiUrl, apiKey.Id, []test.Header{{Name: headerPermission, Value: "PERM_VIEW"}, {Name: headerModifier, Value: "GRANT"}})

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
			ErrorMessage: `{"Result":"error","ErrorMessage":"header permission is required"}`,
			StatusCode:   400,
		},
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid permission"}`,
			StatusCode:   400,
		},
		{
			Value:        "PERM_VIEWW",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid permission"}`,
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
	grantUserPermission(t, idUser, models.UserPermManageLogs)

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
	removeUserPermission(t, idUser, models.UserPermManageLogs)
}

func testApiModifyCall(t *testing.T, apiKey, targetKey string, permission string, grant bool) {
	const apiUrl = "/auth/modify"
	const headerApiKeyModify = "targetKey"
	const headerPermission = "permission"
	const headerModifier = "permissionModifier"

	modifier := "REVOKE"
	if grant {
		modifier = "GRANT"
	}
	w, r := getRecorder(apiUrl, apiKey, []test.Header{{
		Name:  headerApiKeyModify,
		Value: targetKey,
	}, {
		Name:  headerPermission,
		Value: permission,
	}, {
		Name:  headerModifier,
		Value: modifier,
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)

	defer test.ExpectPanic(t)
	apiModifyApiKey(w, &paramAuthCreate{}, models.User{Id: 7})
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
	testDeleteFileCall(t, apiKey.Id, "", 400, `{"Result":"error","ErrorMessage":"header id is required"}`)
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

	defer test.ExpectPanic(t)
	apiDeleteFile(w, &paramAuthCreate{}, models.User{Id: 7})
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

	defer test.ExpectPanic(t)
	apiListSingle(w, &paramAuthCreate{}, models.User{Id: 7})
}

func TestUpload(t *testing.T) {
	result, body := uploadNewFile(t)
	test.IsEqualString(t, result.Result, "OK")
	test.IsEqualString(t, result.FileInfo.Size, "3 B")
	test.IsEqualInt(t, result.FileInfo.DownloadsRemaining, 200)
	test.IsEqualBool(t, result.FileInfo.IsPasswordProtected, true)
	test.IsEqualString(t, result.FileInfo.UrlDownload, "http://127.0.0.1:53843/d?id="+result.FileInfo.Id)
	// newFileId := result.FileInfo.Id
	w, r := test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Content-Type isn't multipart/form-data")
	test.IsEqualInt(t, w.Code, 400)

	defer test.ExpectPanic(t)
	apiUploadFile(w, &paramAuthCreate{}, models.User{Id: 7})
}

func uploadNewFile(t *testing.T) (models.Result, *bytes.Buffer) {
	file, err := os.Open("test/fileupload.jpg")
	test.IsNil(t, err)
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	test.IsNil(t, err)
	_, err = io.Copy(part, file)
	test.IsNil(t, err)
	err = writer.WriteField("allowedDownloads", "200")
	test.IsNil(t, err)
	err = writer.WriteField("expiryDays", "10")
	test.IsNil(t, err)
	err = writer.WriteField("password", "12345")
	test.IsNil(t, err)
	err = writer.Close()
	test.IsNil(t, err)
	newApiKeyUser := generateNewKey(true, idUser, "")
	w, r := test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: newApiKeyUser.Id,
	}}, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	result := models.Result{}
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	return result, body
}

func TestDuplicate(t *testing.T) {
	const apiUrl = "/files/duplicate"
	const headerId = "id"
	const headerAllowedDownloads = "allowedDownloads"

	apiKey := testAuthorisation(t, apiUrl, models.ApiPermUpload)
	testInvalidFileId(t, apiUrl, apiKey.Id, false)

	validHeader := []test.Header{{Name: headerId, Value: idFileUser}}
	invalidParameter := []invalidParameterValue{
		{
			Value:        "invalid",
			ErrorMessage: `{"Result":"error","ErrorMessage":"invalid value in header allowedDownloads supplied"}`,
			StatusCode:   400,
		},
	}
	testInvalidParameters(t, apiUrl, apiKey.Id, validHeader, headerAllowedDownloads, invalidParameter)

	uploadedFile, _ := uploadNewFile(t)
	originalFile, ok := database.GetMetaDataById(uploadedFile.FileInfo.Id)
	test.IsEqualBool(t, ok, true)
	originalFile.DownloadCount = 20
	originalFile.PasswordHash = "abcde"
	database.SaveMetaData(originalFile)
	originalFile, ok = database.GetMetaDataById(originalFile.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, originalFile.Id, originalFile.Id)
	test.IsEqualInt(t, originalFile.DownloadCount, 20)

	for i := 0; i < 8; i++ {
		headers := []test.Header{{Name: "id", Value: originalFile.Id}}
		if i > 0 {
			if i == 1 {
				headers = append(headers, test.Header{Name: "allowedDownloads", Value: "0"})
			} else {
				headers = append(headers, test.Header{Name: "allowedDownloads", Value: "5"})
			}
		}
		if i > 2 {
			if i == 3 {
				headers = append(headers, test.Header{Name: "expiryDays", Value: "0"})
			} else {
				headers = append(headers, test.Header{Name: "expiryDays", Value: "7"})
			}
		}
		if i > 4 {
			headers = append(headers, test.Header{Name: "password", Value: "secret"})
		}
		if i > 5 {
			headers = append(headers, test.Header{Name: "originalPassword", Value: "true"})
		}
		if i > 6 {
			headers = append(headers, test.Header{Name: "filename", Value: "a_new_filename"})
		}

		w, r := getRecorder(apiUrl, apiKey.Id, headers)
		Process(w, r)
		test.IsEqualInt(t, w.Code, 200)
		output, err := io.ReadAll(w.Body)
		test.IsNil(t, err)

		var outputFile models.FileApiOutput
		err = json.Unmarshal(output, &outputFile)
		test.IsNil(t, err)

		newFile, ok := database.GetMetaDataById(outputFile.Id)
		test.IsEqualBool(t, ok, true)

		test.IsEqualString(t, newFile.Id, outputFile.Id)
		test.IsEqualBool(t, newFile.Id != originalFile.Id, true)
		if i > 6 {
			test.IsEqualString(t, newFile.Name, "a_new_filename")
		} else {
			test.IsEqualString(t, newFile.Name, originalFile.Name)
		}
		test.IsEqualString(t, newFile.Size, originalFile.Size)
		test.IsEqualString(t, newFile.SHA1, originalFile.SHA1)
		test.IsEqualBool(t, originalFile.PasswordHash == newFile.PasswordHash, i != 5)
		test.IsEqualBool(t, originalFile.ExpireAtString == newFile.ExpireAtString, i < 3)
		test.IsEqualBool(t, originalFile.ExpireAt == newFile.ExpireAt, i < 3)
		test.IsEqualBool(t, newFile.UnlimitedTime, i == 3)
		test.IsEqualInt64(t, originalFile.SizeBytes, newFile.SizeBytes)
		if i > 2 {
			test.IsEqualInt(t, newFile.DownloadsRemaining, 5)
		}
		if i == 0 {
			test.IsEqualInt(t, newFile.DownloadsRemaining, 200)
		}
		if i == 1 {
			test.IsEqualInt(t, newFile.DownloadsRemaining, 0)
		}
		test.IsEqualBool(t, newFile.UnlimitedDownloads, i == 1)
		test.IsEqualInt(t, newFile.DownloadCount, 0)
	}

	defer test.ExpectPanic(t)
	apiDuplicateFile(nil, &paramAuthCreate{}, models.User{Id: 7})
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

	defer test.ExpectPanic(t)
	apiChunkAdd(w, &paramAuthCreate{}, models.User{Id: 7})
}

func TestChunkComplete(t *testing.T) {
	w, r := test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "uuid", Value: "tmpupload123"},
		{Name: "filename", Value: "test.upload"},
		{Name: "filesize", Value: "13"}},
		nil)
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

	// data.Set("filesize", "15")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "uuid", Value: "tmpupload123"},
		{Name: "filename", Value: "test.upload"},
		{Name: "filesize", Value: "15"}}, nil)
	Process(w, r)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")

	defer test.ExpectPanic(t)
	apiChunkComplete(w, &paramAuthCreate{}, models.User{Id: 7})
}
