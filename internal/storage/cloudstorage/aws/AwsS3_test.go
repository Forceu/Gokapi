// +build awstest
// +build !awsmock

package aws

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
)

var testFile, invalidFile, invalidBucket, invalidAll models.File

func TestMain(m *testing.M) {
	testFile.AwsBucket = "gokapi-test"
	testFile.SHA256 = "testfile"
	invalidFile.AwsBucket = "gokapi-test"
	invalidFile.SHA256 = "invalid"
	invalidBucket.AwsBucket = "invalid"
	invalidBucket.SHA256 = "testfile"
	invalidAll.AwsBucket = "invalid"
	invalidAll.SHA256 = "invalid"
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	Init()
	// For testing Backblaze, as the bucket name in the dev account is gokapi instead of gokapi-test
	if os.Getenv("GOKAPI_AWS_ENDPOINT") != "" {
		testFile.AwsBucket = "gokapi"
		invalidFile.AwsBucket = "gokapi"
	}
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

func TestRedirectToDownload(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download", nil)
	err := RedirectToDownload(w, r, testFile)
	test.IsNil(t, err)
	fmt.Println(w.Body.String())
	test.ResponseBodyContains(t, w, "<a href=\"https://")
	test.IsEqualInt(t, w.Code, 307)
}

func TestFileExists(t *testing.T) {
	result, err := FileExists(invalidFile)
	test.IsEqualBool(t, result, false)
	test.IsNil(t, err)
	result, _ = FileExists(invalidBucket)
	test.IsEqualBool(t, result, false)
	result, _ = FileExists(invalidAll)
	test.IsEqualBool(t, result, false)
	result, err = FileExists(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
}

func TestDeleteObject(t *testing.T) {
	result, err := FileExists(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
	result, err = DeleteObject(testFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
	result, err = FileExists(testFile)
	test.IsEqualBool(t, result, false)
	test.IsNil(t, err)
	result, err = DeleteObject(invalidFile)
	test.IsEqualBool(t, result, true)
	test.IsNil(t, err)
}

func TestIsCredentialProvided(t *testing.T) {
	os.Unsetenv("GOKAPI_AWS_REGION")
	os.Unsetenv("GOKAPI_AWS_KEY")
	os.Unsetenv("GOKAPI_KEY_SECRET")
	os.Unsetenv("GOKAPI_AWS_BUCKET")
	environmentHolder = environment.Environment{}
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	environmentHolder = environment.Environment{}
	os.Setenv("GOKAPI_AWS_REGION", "valid")
	environmentHolder = environment.Environment{}
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	environmentHolder = environment.Environment{}
	os.Setenv("GOKAPI_AWS_KEY", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	test.IsEqualBool(t, IsCredentialProvided(true), false)
	environmentHolder = environment.Environment{}
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	test.IsEqualBool(t, IsCredentialProvided(true), false)
	environmentHolder = environment.Environment{}
	os.Setenv("GOKAPI_AWS_BUCKET", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), true)
	test.IsEqualBool(t, IsCredentialProvided(true), true)
}
