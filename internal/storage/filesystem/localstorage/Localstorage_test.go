package localstorage

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Mkdir("test/", 0777)
	os.Mkdir("test/data", 0777)
	exitVal := m.Run()
	os.RemoveAll("test")
	os.Exit(exitVal)
}

func getTestDriver(t *testing.T) *localStorageDriver {
	t.Helper()
	driver := GetDriver()
	result, ok := driver.(*localStorageDriver)
	test.IsEqualBool(t, ok, true)
	return result
}

func initDriver(t *testing.T, d *localStorageDriver) {
	ok := d.Init(Config{
		DataPath:   "test/data",
		FilePrefix: "123",
	})
	test.IsEqualBool(t, ok, true)
}

func TestGetDriver(t *testing.T) {
	getTestDriver(t)
}

func TestLocalStorageDriver_Init(t *testing.T) {
	driver := getTestDriver(t)
	ok := driver.Init(Config{
		DataPath:   "test",
		FilePrefix: "tpref",
	})
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, driver.getPath(), "test/")
	ok = driver.Init(Config{
		DataPath:   "test2/",
		FilePrefix: "",
	})
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, driver.getPath(), "test2/")
	defer test.ExpectPanic(t)
	driver.Init(struct {
		invalid string
	}{invalid: "true"})
}

func TestLocalStorageDriver_Init2(t *testing.T) {
	driver := getTestDriver(t)
	defer test.ExpectPanic(t)
	driver.Init(Config{
		DataPath:   "",
		FilePrefix: "tpref",
	})
}

func TestLocalStorageDriver_IsAvailable(t *testing.T) {
	driver := getTestDriver(t)
	test.IsEqualBool(t, driver.IsAvailable(), true)
}

func TestGetDataPath(t *testing.T) {
	driver := getTestDriver(t)
	initDriver(t, driver)
	test.IsEqualString(t, driver.getPath(), "test/data/")
	driver.dataPath = ""
	defer test.ExpectPanic(t)
	driver.getPath()
}
func TestLocalStorageDriver_MoveToFilesystem(t *testing.T) {
	driver := getTestDriver(t)
	initDriver(t, driver)
	metaData := models.File{
		SHA1: "testsha",
	}
	err := driver.MoveToFilesystem(nil, metaData)
	test.IsNotNil(t, err)
	err = os.WriteFile("test/testfile", []byte("This is a test"), 0777)
	test.IsNil(t, err)
	file, err := os.Open("test/testfile")
	test.IsNil(t, err)
	err = driver.MoveToFilesystem(file, models.File{})
	test.IsNotNil(t, err)
	test.FileExists(t, "test/testfile")
	file, err = os.Open("test/testfile")
	test.IsNil(t, err)
	err = driver.MoveToFilesystem(file, metaData)
	test.IsNil(t, err)
	test.FileDoesNotExist(t, "test/testfile")
	test.FileExists(t, "test/data/123testsha")

}

func TestLocalFile_Exists(t *testing.T) {
	driver := getTestDriver(t)
	initDriver(t, driver)
	test.FileExists(t, "test/data/123testsha")
	test.FileDoesNotExist(t, "test/data/testsha")
	file := driver.GetFile("testsha")
	test.IsEqualBool(t, file.Exists(), true)
	test.IsEqualString(t, file.GetName(), "123testsha")
}

func TestLocalStorageDriver_FileExists(t *testing.T) {
	driver := getTestDriver(t)
	initDriver(t, driver)
	test.FileExists(t, "test/data/123testsha")
	test.FileDoesNotExist(t, "test/data/testsha")
	exist, err := driver.FileExists("testsha")
	test.IsNil(t, err)
	test.IsEqualBool(t, exist, true)
	exist, err = driver.FileExists("123testsha")
	test.IsNil(t, err)
	test.IsEqualBool(t, exist, false)
}

func TestLocalStorageDriver_GetSystemName(t *testing.T) {
	driver := getTestDriver(t)
	test.IsEqualString(t, driver.GetSystemName(), "localstorage")
}
