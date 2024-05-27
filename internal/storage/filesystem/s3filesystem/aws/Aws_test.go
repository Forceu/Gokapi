//go:build awstest && !awsmock && test

package aws

import (
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var testFile, invalidFile, invalidBucket, invalidAll models.File

var isRealAwsServer bool

func TestMain(m *testing.M) {
	testFile.AwsBucket = "gokapi-test"
	testFile.SHA1 = "testfile"
	testFile.Name = "Testfile.jpg"
	invalidFile.AwsBucket = "gokapi-test"
	invalidFile.SHA1 = "invalid"
	invalidBucket.AwsBucket = "invalid"
	invalidBucket.SHA1 = "testfile"
	invalidAll.AwsBucket = "invalid"
	invalidAll.SHA1 = "invalid"
	if os.Getenv("REAL_AWS_CREDENTIALS") != "true" {
		ts := startMockServer()
		os.Setenv("GOKAPI_AWS_ENDPOINT", ts.URL)
		defer ts.Close()
	} else {
		isRealAwsServer = true
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func startMockServer() *httptest.Server {
	os.Setenv("GOKAPI_AWS_BUCKET", "gokapi-test")
	os.Setenv("GOKAPI_AWS_REGION", "mock-region-1")
	os.Setenv("GOKAPI_AWS_KEY", "accId")
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "accKey")
	backend := s3mem.New()
	_ = backend.CreateBucket("gokapi")
	_ = backend.CreateBucket("gokapi-test")
	faker := gofakes3.New(backend)
	return httptest.NewServer(faker.Server())
}

func TestInit(t *testing.T) {
	config, ok := cloudconfig.Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, Init(config.Aws), true)
	// For testing Backblaze, as the bucket name in the dev account is gokapi instead of gokapi-test
	if os.Getenv("GOKAPI_AWS_ENDPOINT") != "" {
		testFile.AwsBucket = "gokapi"
		invalidFile.AwsBucket = "gokapi"
	}
}

func TestAddBucketName(t *testing.T) {
	file := models.File{Name: "Test"}
	AddBucketName(&file)
	test.IsEqualString(t, file.AwsBucket, "gokapi-test")
}

func TestIsAvailable(t *testing.T) {
	test.IsEqualBool(t, IsAvailable(), true)
}

func TestUploadToAws(t *testing.T) {
	os.WriteFile("test", []byte("testfile-content"), 0777)
	file, _ := os.Open("test")
	location, err := Upload(file, testFile)
	test.IsNil(t, err)
	test.IsNotEmpty(t, location)
	os.Remove("test")
}

func TestDownloadFromAws(t *testing.T) {
	test.FileDoesNotExist(t, "test")
	file, _ := os.Create("test")
	size, err := Download(file, testFile)
	test.IsNil(t, err)
	test.IsEqualBool(t, size == 16, true)
	test.FileExists(t, "test")
	content, _ := os.ReadFile("test")
	test.IsEqualString(t, string(content), "testfile-content")
	os.Remove("test")
}

func TestServeFile(t *testing.T) {
	awsConfig.ProxyDownload = false
	testServing(t, true, false)
	testServing(t, true, true)

	awsConfig.ProxyDownload = true
	testServing(t, false, false)
	testServing(t, false, true)
	awsConfig.ProxyDownload = false
}

func testServing(t *testing.T, expectRedirect, forceDownload bool) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download", nil)

	isBlockng, err := ServeFile(w, r, testFile, forceDownload)
	test.IsEqualBool(t, isBlockng, !expectRedirect)
	test.IsNil(t, err)

	response, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)

	if !expectRedirect {
		test.IsEqualString(t, string(response), "testfile-content")
		test.IsEqualInt(t, w.Code, 200)
		// content-disposition not implemented in s3mem
		if isRealAwsServer {
			expectedContentDisposition := "inline; filename=\"" + testFile.Name + "\""
			if forceDownload {
				expectedContentDisposition = "Attachment; filename=\"" + testFile.Name + "\""
			}
			test.IsEqualString(t, w.Header().Get("Content-Disposition"), expectedContentDisposition)
		}
	} else {
		for _, s := range []string{"<a href=\"http", "Testfile.jpg"} {
			test.IsEqualBool(t, strings.Contains(string(response), s), true)
		}
		test.IsEqualInt(t, w.Code, 307)

		// Get the redirect URL from the response headers
		redirectURL := w.Header().Get("Location")
		if redirectURL == "" {
			t.Fatal("Expected redirect URL in Location header")
		}

		// Follow the redirect and download the content
		resp, err := http.Get(redirectURL)
		test.IsNil(t, err)
		defer resp.Body.Close()

		// Read the content of the downloaded file and  check the content of the downloaded file
		downloadedContent, err := io.ReadAll(resp.Body)
		test.IsNil(t, err)
		test.IsEqualString(t, string(downloadedContent), "testfile-content")

		// content-disposition not implemented in s3mem
		if isRealAwsServer {
			expectedContentDisposition := "inline; filename=\"" + testFile.Name + "\""
			if forceDownload {
				expectedContentDisposition = "Attachment; filename=\"" + testFile.Name + "\""
			}
			test.IsEqualString(t, resp.Header.Get("Content-Disposition"), expectedContentDisposition)
		}
	}
}

func TestFileExists(t *testing.T) {
	result, _, err := FileExists(invalidFile)
	test.IsEqualBool(t, result, false)
	test.IsNil(t, err)
	result, _, _ = FileExists(invalidBucket)
	test.IsEqualBool(t, result, false)
	result, _, _ = FileExists(invalidAll)
	test.IsEqualBool(t, result, false)
	result, _, err = FileExists(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
}

func TestDeleteObject(t *testing.T) {
	result, _, err := FileExists(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
	result, err = DeleteObject(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
	result, _, err = FileExists(testFile)
	test.IsEqualBool(t, result, false)
	test.IsNil(t, err)
	result, err = DeleteObject(invalidFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
}
func TestLogOut(t *testing.T) {
	test.IsEqualBool(t, isCorrectLogin, true)
	LogOut()
	test.IsEqualBool(t, isCorrectLogin, false)
}

func TestGetDefaultBucketName(t *testing.T) {
	test.IsEqualString(t, GetDefaultBucketName(), awsConfig.Bucket)
}

func TestIsCorsCorrectlySet(t *testing.T) {
	// not implemented in s3mem
	if !isRealAwsServer {
		return
	}
	// TODO
}
