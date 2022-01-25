//go:build test
// +build test

package fileupload

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/dataStorage"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestParseConfig(t *testing.T) {
	data := testData{
		allowedDownloads: "9",
		expiryDays:       "5",
		password:         "123",
	}
	config := parseConfig(data, false)
	downloads, _, _ := dataStorage.GetUploadDefaults()
	test.IsEqualInt(t, config.AllowedDownloads, 9)
	test.IsEqualString(t, config.Password, "123")
	test.IsEqualInt(t, config.Expiry, 5)

	test.IsEqualInt(t, downloads, 3)
	config = parseConfig(data, true)
	downloads, _, _ = dataStorage.GetUploadDefaults()
	test.IsEqualInt(t, downloads, 9)
	dataStorage.SaveUploadDefaults(3, 20, "")
	data.allowedDownloads = ""
	data.expiryDays = "invalid"
	config = parseConfig(data, false)
	test.IsEqualInt(t, config.AllowedDownloads, 3)
	test.IsEqualInt(t, config.Expiry, 20)
}

func TestProcess(t *testing.T) {
	w := httptest.NewRecorder()
	r := getRecorder()
	err := Process(w, r, false, 20)
	test.IsNil(t, err)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	result := models.Result{}
	err = json.Unmarshal(body, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Result, "OK")
	test.IsEqualString(t, result.Url, "http://127.0.0.1:53843/d?id=")
	test.IsEqualString(t, result.HotlinkUrl, "http://127.0.0.1:53843/hotlink/")
	test.IsEqualString(t, result.FileInfo.Name, "testFile")
	test.IsEqualString(t, result.FileInfo.SHA256, "17513aad503256b7fdc94d613aeb87b8338c433a")
	test.IsEqualString(t, result.FileInfo.Size, "11 B")
}

func getRecorder() *http.Request {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	writer, _ := w.CreateFormFile("file", "testFile")
	io.WriteString(writer, "testContent")
	w.Close()
	r := httptest.NewRequest("POST", "/upload", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.Header.Add("allowedDownloads", "9")
	r.Header.Add("expiryDays", "5")
	r.Header.Add("password", "123")
	return r
}

type testData struct {
	allowedDownloads, expiryDays, password string
}

func (t testData) Get(key string) string {
	field := reflect.ValueOf(&t).Elem().FieldByName(key)
	if field.IsValid() {
		return field.String()
	}
	return ""
}
