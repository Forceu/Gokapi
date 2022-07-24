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
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"io"
	"mime/multipart"
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
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

const maxMemory = 20

var newKeyId string

func TestNewKey(t *testing.T) {
	newKeyId = NewKey()
	key, ok := database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
}

func TestDeleteKey(t *testing.T) {
	key, ok := database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
	result := DeleteKey(newKeyId)
	test.IsEqualBool(t, result, true)
	_, ok = database.GetApiKey(newKeyId)
	test.IsEqualBool(t, ok, false)
	result = DeleteKey("invalid")
	test.IsEqualBool(t, result, false)
}

func TestIsValidApiKey(t *testing.T) {
	test.IsEqualBool(t, IsValidApiKey("", false), false)
	test.IsEqualBool(t, IsValidApiKey("invalid", false), false)
	test.IsEqualBool(t, IsValidApiKey("validkey", false), true)
	key, ok := database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, true)
	test.IsEqualBool(t, IsValidApiKey("validkey", true), true)
	key, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 0, false)
}

func TestProcess(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = test.GetRecorder("GET", "/api/invalid", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Unauthorized")
	w, r = test.GetRecorder("GET", "/api/invalid", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid request")
	w, r = test.GetRecorder("GET", "/api/invalid", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid request")
}

func TestAuthDisabledLogin(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	configuration.Get().Authentication.Method = authentication.Disabled
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	configuration.Get().Authentication.Method = authentication.Internal
}

func TestChangeFriendlyName(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid api key provided.")
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)

	key, ok := database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Unnamed key")
	w, r = test.GetRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}, {
		Name: "friendlyName", Value: "NewName"}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	key, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "NewName")
	w = httptest.NewRecorder()
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
}

func TestDeleteFile(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	w, r = test.GetRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "invalid",
	},
	}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
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
	Process(w, r, maxMemory)
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

	Process(w, r, maxMemory)
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	result := models.Result{}
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Result, "OK")
	test.IsEqualString(t, result.FileInfo.Size, "3 B")
	test.IsEqualInt(t, result.FileInfo.DownloadsRemaining, 200)
	test.IsEqualBool(t, result.FileInfo.IsPasswordProtected, true)
	test.IsEqualString(t, result.Url, "http://127.0.0.1:53843/d?id=")
	newFileId := result.FileInfo.Id

	w, r = test.GetRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Content-Type isn't multipart/form-data")
	test.IsEqualInt(t, w.Code, 400)

	//
	// Duplication
	//
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"},
	}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("§$§$%&(&//&/invalid"))
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "invalid URL escape")

	data := url.Values{}
	data.Set("id", newFileId)

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("expiryDays", "invalid")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "strconv.Atoi: parsing \"invalid\": invalid syntax")

	data = url.Values{}
	data.Set("id", newFileId)
	data.Set("expiryDays", "20")

	w, r = test.GetRecorder("POST", "/api/files/duplicate", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "id", Value: "invalid"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
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
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")

	w, r = test.GetRecorder("POST", "/api/chunk/complete", nil, []test.Header{
		{Name: "apikey", Value: "validkey"},
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("invalid&&§$%"))
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 400)
	test.ResponseBodyContains(t, w, "error")
}

func TestList(t *testing.T) {
	w, r := test.GetRecorder("GET", "/api/files/list", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "picture.jpg")
}
