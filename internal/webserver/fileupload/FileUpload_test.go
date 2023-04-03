package fileupload

import (
	"bytes"
	"encoding/json"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/processingstatus"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/r3labs/sse/v2"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	processingstatus.Init(sse.New())
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestParseConfig(t *testing.T) {
	data := testData{
		allowedDownloads: "9",
		expiryDays:       "5",
		password:         "123",
		isE2E:            "",
		realSize:         "",
	}
	config, err := parseConfig(data, false)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.IsEndToEndEncrypted, false)
	test.IsEqualInt64(t, config.RealSize, 0)

	defaults := database.GetUploadDefaults()
	test.IsEqualInt(t, config.AllowedDownloads, 9)
	test.IsEqualString(t, config.Password, "123")
	test.IsEqualInt(t, config.Expiry, 5)
	test.IsEqualInt(t, defaults.Downloads, 3)

	config, err = parseConfig(data, true)
	test.IsNil(t, err)
	defaults = database.GetUploadDefaults()
	test.IsEqualInt(t, defaults.Downloads, 9)
	database.SaveUploadDefaults(models.LastUploadValues{Downloads: 3, TimeExpiry: 20})

	data.allowedDownloads = ""
	data.expiryDays = "invalid"

	config, err = parseConfig(data, false)
	test.IsNil(t, err)
	test.IsEqualInt(t, config.AllowedDownloads, 3)
	test.IsEqualInt(t, config.Expiry, 20)
	test.IsEqualBool(t, config.UnlimitedTime, false)
	test.IsEqualBool(t, config.UnlimitedDownload, false)

	data.allowedDownloads = "0"
	data.expiryDays = "0"
	config, err = parseConfig(data, false)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.UnlimitedTime, true)
	test.IsEqualBool(t, config.UnlimitedDownload, true)

	data.isE2E = "true"
	data.realSize = "200"
	config, err = parseConfig(data, false)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.IsEndToEndEncrypted, true)
	test.IsEqualInt64(t, config.RealSize, 200)
}

func TestProcess(t *testing.T) {
	w, r := test.GetRecorder("POST", "/upload", nil, nil, strings.NewReader("invalid§$%&%§"))
	err := Process(w, r, false, 20)
	test.IsNotNil(t, err)

	data := url.Values{}
	data.Set("file", "invalid")

	w = httptest.NewRecorder()
	r = getFileUploadRecorder(false)
	err = Process(w, r, false, 20)
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
	test.IsEqualString(t, result.FileInfo.Size, "11 B")
	test.IsEqualBool(t, result.FileInfo.UnlimitedTime, false)
	test.IsEqualBool(t, result.FileInfo.UnlimitedDownloads, false)
}

func TestProcessNewChunk(t *testing.T) {
	w, r := test.GetRecorder("POST", "/uploadChunk", nil, nil, strings.NewReader("invalid§$%&%§"))
	err := ProcessNewChunk(w, r, false)
	test.IsNotNil(t, err)

	w = httptest.NewRecorder()
	r = getFileUploadRecorder(false)
	err = ProcessNewChunk(w, r, false)
	test.IsNotNil(t, err)

	w = httptest.NewRecorder()
	r = getFileUploadRecorder(true)
	err = ProcessNewChunk(w, r, false)
	test.IsNil(t, err)
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(response), "{\"result\":\"OK\"}")
}

func TestCompleteChunk(t *testing.T) {
	w, r := test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader("invalid§$%&%§"))
	err := CompleteChunk(w, r, false)
	test.IsNotNil(t, err)

	w = httptest.NewRecorder()
	r = getFileUploadRecorder(false)
	err = CompleteChunk(w, r, false)
	test.IsNotNil(t, err)

	data := url.Values{}
	data.Set("allowedDownloads", "9")
	data.Set("expiryDays", "5")
	data.Set("password", "123")
	data.Set("chunkid", "randomchunkuuid")
	data.Set("filename", "random.file")
	data.Set("filesize", "13")
	w, r = test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader(data.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	err = CompleteChunk(w, r, false)
	test.IsNil(t, err)

	result := struct {
		FileInfo models.FileApiOutput `json:"FileInfo"`
	}{}
	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	err = json.Unmarshal(response, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.FileInfo.Name, "random.file")

	data.Set("chunkid", "invalid")
	w, r = test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader(data.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	err = CompleteChunk(w, r, false)
	test.IsNotNil(t, err)
}

func getFileUploadRecorder(addChunkInfo bool) *http.Request {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if addChunkInfo {
		w.WriteField("dztotalfilesize", "13")
		w.WriteField("dzchunkbyteoffset", "0")
		w.WriteField("dzuuid", "randomchunkuuid")
	}
	writer, _ := w.CreateFormFile("file", "testFile")
	io.WriteString(writer, "testContent")
	w.Close()
	r := httptest.NewRequest("POST", "/upload", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())
	if !addChunkInfo {
		r.Header.Add("allowedDownloads", "9")
		r.Header.Add("expiryDays", "5")
		r.Header.Add("password", "123")
	}
	return r
}

type testData struct {
	allowedDownloads, expiryDays, password, isE2E, realSize string
}

func (t testData) Get(key string) string {
	field := reflect.ValueOf(&t).Elem().FieldByName(key)
	if field.IsValid() {
		return field.String()
	}
	return ""
}
