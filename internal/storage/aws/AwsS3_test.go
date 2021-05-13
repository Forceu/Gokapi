// +build awstest
// +build !awsmock

package aws

import (
	"Gokapi/internal/models"
	"Gokapi/internal/test"
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

func TestFileExists(t *testing.T) {
	result, err := FileExists(invalidFile)
	test.IsEqualBool(t, result, false)
	test.IsNil(t, err)
	result, err = FileExists(invalidBucket)
	test.IsEqualBool(t, result, false)
	test.IsNotNil(t, err)
	result, err = FileExists(invalidAll)
	test.IsEqualBool(t, result, false)
	test.IsNotNil(t, err)
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
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	os.Setenv("AWS_REGION", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	os.Setenv("AWS_ACCESS_KEY_ID", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), false)
	test.IsEqualBool(t, IsCredentialProvided(true), false)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "valid")
	test.IsEqualBool(t, IsCredentialProvided(false), true)
	test.IsEqualBool(t, IsCredentialProvided(true), true)
}
