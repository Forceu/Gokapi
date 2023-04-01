package s3filesystem

import (
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func getTestDriver(t *testing.T) *s3StorageDriver {
	t.Helper()
	driver := GetDriver()
	result, ok := driver.(*s3StorageDriver)
	test.IsEqualBool(t, ok, true)
	return result
}

func TestGetDriver(t *testing.T) {
	getTestDriver(t)
}

func TestS3StorageDriver_Init(t *testing.T) {
	driver := getTestDriver(t)
	defer test.ExpectPanic(t)
	driver.Init("test")
	defer test.ExpectPanic(t)
	driver.Init(Config{Bucket: ""})
}
func TestS3StorageDriver_Init2(t *testing.T) {
	driver := getTestDriver(t)
	defer test.ExpectPanic(t)
	driver.Init(Config{Bucket: ""})
}

func TestS3StorageDriver_Init3(t *testing.T) {
	driver := getTestDriver(t)
	ok := driver.Init(Config{Bucket: "test"})
	test.IsEqualBool(t, ok, false)
	test.IsEqualString(t, driver.Bucket, "test")
	test.IsEqualBool(t, driver.IsAvailable(), false) // TODO
}

func TestS3StorageDriver_GetSystemName(t *testing.T) {
	driver := getTestDriver(t)
	test.IsEqualString(t, driver.GetSystemName(), "awss3")
}

func TestAwsFile_GetName(t *testing.T) {
	driver := getTestDriver(t)
	driver.Init(Config{Bucket: "test"})
	file := driver.GetFile("testfile")
	test.IsEqualString(t, file.GetName(), "testfile")
}
