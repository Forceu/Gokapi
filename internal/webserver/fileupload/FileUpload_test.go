package fileupload

import (
	"bytes"
	"encoding/json"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
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
	configuration.ConnectDatabase()
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
	config, err := parseConfig(data)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.IsEndToEndEncrypted, false)
	test.IsEqualInt64(t, config.RealSize, 0)

	test.IsEqualInt(t, config.AllowedDownloads, 9)
	test.IsEqualString(t, config.Password, "123")
	test.IsEqualInt(t, config.Expiry, 5)

	config, err = parseConfig(data)
	test.IsNil(t, err)

	data.allowedDownloads = ""
	data.expiryDays = "invalid"

	config, err = parseConfig(data)
	test.IsNil(t, err)
	test.IsEqualInt(t, config.AllowedDownloads, 1)
	test.IsEqualInt(t, config.Expiry, 14)
	test.IsEqualBool(t, config.UnlimitedTime, false)
	test.IsEqualBool(t, config.UnlimitedDownload, false)

	data.allowedDownloads = "0"
	data.expiryDays = "0"
	config, err = parseConfig(data)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.UnlimitedTime, true)
	test.IsEqualBool(t, config.UnlimitedDownload, true)

	data.isE2E = "true"
	data.realSize = "200"
	config, err = parseConfig(data)
	test.IsNil(t, err)
	test.IsEqualBool(t, config.IsEndToEndEncrypted, true)
	test.IsEqualInt64(t, config.RealSize, 200)
}

func TestProcess(t *testing.T) {
	w, r := test.GetRecorder("POST", "/upload", nil, nil, strings.NewReader("invalid§$%&%§"))
	err := ProcessCompleteFile(w, r, 9, 20)
	test.IsNotNil(t, err)

	w = httptest.NewRecorder()
	r = getFileUploadRecorder(false)
	err = ProcessCompleteFile(w, r, 9, 20)
	test.IsNil(t, err)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	result := models.Result{}
	err = json.Unmarshal(body, &result)
	test.IsNil(t, err)
	test.IsEqualString(t, result.Result, "OK")
	test.IsEqualString(t, result.FileInfo.UrlDownload, "http://127.0.0.1:53843/d?id="+result.FileInfo.Id)
	test.IsEqualString(t, result.FileInfo.UrlHotlink, "http://127.0.0.1:53843/downloadFile?id="+result.FileInfo.Id)
	test.IsEqualString(t, result.FileInfo.Name, "testFile")
	test.IsEqualString(t, result.FileInfo.Size, "11 B")
	test.IsEqualBool(t, result.FileInfo.UnlimitedTime, false)
	test.IsEqualBool(t, result.FileInfo.UnlimitedDownloads, false)
	test.IsEqualInt(t, result.FileInfo.UploaderId, 9)
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
	body := strings.NewReader("%")
	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, _, _, err := ParseFileHeader(r)
	test.IsNotNil(t, err)

	w := httptest.NewRecorder()
	r = getFileUploadRecorder(false)
	_, _, _, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	data := url.Values{}
	data.Set("isE2E", "true")
	data.Set("realSize", "none")
	w, r = test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader(data.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	chunkId, header, config, err := ParseFileHeader(r)
	test.IsNotNil(t, err)

	data.Del("isE2E")
	data.Del("realSize")
	data.Set("allowedDownloads", "9")
	data.Set("expiryDays", "5")
	data.Set("password", "123")
	data.Set("chunkid", "randomchunkuuid")
	data.Set("filename", "random.file")
	data.Set("filesize", "13")
	w, r = test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader(data.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	chunkId, header, config, err = ParseFileHeader(r)
	test.IsNil(t, err)
	file, err := CompleteChunk(chunkId, header, 9, config)
	test.IsNil(t, err)
	test.IsEqualString(t, file.Name, "random.file")

	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(response), "")

	data.Set("chunkid", "invalid")
	w, r = test.GetRecorder("POST", "/uploadComplete", nil, nil, strings.NewReader(data.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	_, _, _, err = ParseFileHeader(r)
	test.IsNil(t, err)
	_, err = CompleteChunk(chunkId, header, 9, config)
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
