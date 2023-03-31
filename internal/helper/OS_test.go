package helper

import (
	"errors"
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
)

func TestIsInArray(t *testing.T) {
	test.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "test2"), true)
	test.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "invalid"), false)
}

func TestFolderCreation(t *testing.T) {
	test.IsEqualBool(t, FolderExists("invalid"), false)
	test.FileDoesNotExist(t, "invalid/file")
	test.IsEqualBool(t, FileExists("invalid/file"), false)
	CreateDir("invalid")
	test.IsEqualBool(t, FolderExists("invalid"), true)
	err := os.WriteFile("invalid/file", []byte("test"), 0644)
	if err != nil {
		t.Error(err)
	}
	test.FileExists(t, "invalid/file")
	test.IsEqualBool(t, FileExists("invalid/file"), true)
	os.RemoveAll("invalid")
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
