// +build test

package api

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "Unnamed key")
}

func TestDeleteKey(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "Unnamed key")
	result := DeleteKey(newKeyId)
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "")
	test.IsEqualBool(t, result, true)
	result = DeleteKey("invalid")
	test.IsEqualBool(t, result, false)
}

func TestIsValidApiKey(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualBool(t, IsValidApiKey("", false), false)
	test.IsEqualBool(t, IsValidApiKey("invalid", false), false)
	test.IsEqualBool(t, IsValidApiKey("validkey", false), true)
	test.IsEqualBool(t, settings.ApiKeys["validkey"].LastUsed == 0, true)
	test.IsEqualBool(t, IsValidApiKey("validkey", true), true)
	test.IsEqualBool(t, settings.ApiKeys["validkey"].LastUsed == 0, false)
}

func TestProcess(t *testing.T) {
	w, r := getRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = getRecorder("GET", "/api/invalid", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Unauthorized")
	w, r = getRecorder("GET", "/api/invalid", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid request")
	w, r = getRecorder("GET", "/api/invalid", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid request")
}

func TestAuthDisabledLogin(t *testing.T) {
	w, r := getRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	settings := configuration.GetServerSettings()
	settings.DisableLogin = true
	configuration.Release()
	w, r = getRecorder("GET", "/api/auth/friendlyname", nil, nil, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	settings.DisableLogin = false
}

func TestChangeFriendlyName(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	w, r := getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid api key provided.")
	w, r = getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.ApiKeys["validkey"].FriendlyName, "Unnamed key")
	w, r = getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}, {
		Name: "friendlyName", Value: "NewName"}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.ApiKeys["validkey"].FriendlyName, "NewName")
	w = httptest.NewRecorder()
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
}

func TestDeleteFile(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	w, r := getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	w, r = getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "invalid",
	},
	}, nil)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	test.IsEqualString(t, settings.Files["jpLXGJKigM4hjtA6T6sN2"].Id, "jpLXGJKigM4hjtA6T6sN2")
	w, r = getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "jpLXGJKigM4hjtA6T6sN2",
	},
	}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.Files["jpLXGJKigM4hjtA6T6sN2"].Id, "")
}

func TestUpload(t *testing.T) {
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
	w, r := getRecorder("POST", "/api/files/add", nil, []test.Header{{
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
	test.IsNotEqualString(t, result.FileInfo.PasswordHash, "")
	test.IsEqualString(t, result.Url, "http://127.0.0.1:53843/d?id=")
	w, r = getRecorder("POST", "/api/files/add", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, body)
	Process(w, r, maxMemory)
	test.ResponseBodyContains(t, w, "Content-Type isn't multipart/form-data")
	test.IsEqualInt(t, w.Code, 400)
}

func TestList(t *testing.T) {
	w, r := getRecorder("GET", "/api/files/list", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}}, nil)
	Process(w, r, maxMemory)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "picture.jpg")
}

func getRecorder(method, target string, cookies []test.Cookie, headers []test.Header, body io.Reader) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, target, body)
	if cookies != nil {
		for _, cookie := range cookies {
			r.AddCookie(&http.Cookie{
				Name:  cookie.Name,
				Value: cookie.Value,
				Path:  "/",
			})
		}
	}
	if headers != nil {
		for _, header := range headers {
			r.Header.Set(header.Name, header.Value)
		}
	}
	return w, r
}
