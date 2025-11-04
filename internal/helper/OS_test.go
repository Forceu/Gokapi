package helper

import (
	"errors"
	"os"
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

func TestIsInArray(t *testing.T) {
	test.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "test2"), true)
	test.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "invalid"), false)
}

func TestFolderCreation(t *testing.T) {
	test.IsEqualBool(t, FolderExists("invalid"), false)
	test.FileDoesNotExist(t, "invalid/file")
	exists, err := FileExists("invalid/file")
	test.IsEqualBool(t, exists, false)
	test.IsNil(t, err)
	CreateDir("invalid")
	test.IsEqualBool(t, FolderExists("invalid"), true)
	err = os.WriteFile("invalid/file", []byte("test"), 0644)
	if err != nil {
		t.Error(err)
	}
	test.FileExists(t, "invalid/file")
	exists, err = FileExists("invalid/file")
	test.IsNil(t, err)
	test.IsEqualBool(t, exists, true)
	_ = os.RemoveAll("invalid")
}

func TestReadLine(t *testing.T) {
	original := test.StartMockInputStdin("test")
	output := ReadLine()
	test.StopMockInputStdin(original)
	test.IsEqualString(t, output, "test")
}

func TestReadPassword(t *testing.T) {
	original := test.StartMockInputStdin("testpw")
	output := ReadPassword()
	test.StopMockInputStdin(original)
	test.IsEqualString(t, output, "testpw")
}

func TestGetFileSize(t *testing.T) {
	os.WriteFile("testfile", []byte(""), 0777)
	file, err := os.OpenFile("testfile", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	size, _ := GetFileSize(file)
	test.IsEqualInt(t, int(size), 0)
	os.WriteFile("testfile", []byte("123"), 0777)
	size, _ = GetFileSize(file)
	test.IsEqualInt(t, int(size), 3)
	file, _ = os.OpenFile("invalid", os.O_RDONLY, 0644)
	size, _ = GetFileSize(file)
	test.IsEqualInt(t, int(size), 0)
	os.Remove("testfile")
}

func TestCheck(t *testing.T) {
	var err error
	Check(err)
	defer test.ExpectPanic(t)
	err = errors.New("test")
	Check(err)
}

func TestCheckIgnoreTimeout(t *testing.T) {
	CheckIgnoreTimeout(nil)
	CheckIgnoreTimeout(os.ErrDeadlineExceeded)
	defer test.ExpectPanic(t)
	CheckIgnoreTimeout(errors.New("other"))
}
