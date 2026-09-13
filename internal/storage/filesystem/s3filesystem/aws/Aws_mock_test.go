//go:build !noaws && awsmock && test

package aws

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
)

func initMockCredentials(t *testing.T) {
	t.Helper()
	os.Setenv("GOKAPI_AWS_BUCKET", bucketName)
	os.Setenv("GOKAPI_AWS_REGION", region)
	os.Setenv("GOKAPI_AWS_KEY", accessId)
	os.Setenv("GOKAPI_AWS_KEY_SECRET", accessKey)
	ok := Init(models.AwsConfig{})
	test.IsEqualBool(t, ok, true)
}

func TestServeFile_ProxyDownload(t *testing.T) {
	initMockCredentials(t)
	awsConfig.ProxyDownload = true
	defer func() { awsConfig.ProxyDownload = false }()

	file := models.File{Id: "awsTest1234567890123", SHA1: "x341354656543213246465465465432456898794"}
	err := os.MkdirAll("data", 0777)
	test.IsNil(t, err)
	err = os.WriteFile("data/"+file.SHA1, []byte("proxy download content"), 0777)
	test.IsNil(t, err)
	defer os.Remove("data/" + file.SHA1)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	isBlocking, err := ServeFile(w, r, file, false, false)
	test.IsNil(t, err)
	test.IsEqualBool(t, isBlocking, true)
	test.ResponseBodyIs(t, w, "proxy download content")
}

func TestServeFile_ProxyDownload_MissingFixture(t *testing.T) {
	initMockCredentials(t)
	awsConfig.ProxyDownload = true
	defer func() { awsConfig.ProxyDownload = false }()

	// awsTest1234567890123 is registered as uploaded by Init(), but its on-disk fixture
	// is never written by this test - proxyDownload must surface that as an error rather
	// than panicking.
	file := models.File{Id: "awsTest1234567890123", SHA1: "x341354656543213246465465465432456898794"}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	_, err := ServeFile(w, r, file, false, false)
	test.IsNotNil(t, err)
}

func TestServeFile_RedirectDownload(t *testing.T) {
	initMockCredentials(t)
	awsConfig.ProxyDownload = false

	file := models.File{Id: "awsTest1234567890123", SHA1: "x341354656543213246465465465432456898794"}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	isBlocking, err := ServeFile(w, r, file, false, false)
	test.IsNil(t, err)
	test.IsEqualBool(t, isBlocking, false)
	test.ResponseBodyContains(t, w, "https://redirect.url")
}
