package helper

import (
	testconfiguration "Gokapi/internal/test"
	testconfiguration2 "Gokapi/internal/test/testconfiguration"
	"io/ioutil"
	"os"
	"testing"
)

func TestIsInArray(t *testing.T) {
	testconfiguration.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "test2"), true)
	testconfiguration.IsEqualBool(t, IsInArray([]string{"test", "test2", "test3"}, "invalid"), false)
}

func TestFolderCreation(t *testing.T) {
	testconfiguration.IsEqualBool(t, FolderExists("invalid"), false)
	testconfiguration.IsEqualBool(t, FileExists("invalid/file"), false)
	CreateDir("invalid")
	testconfiguration.IsEqualBool(t, FolderExists("invalid"), true)
	err := ioutil.WriteFile("invalid/file", []byte("test"), 0644)
	if err != nil {
		t.Error(err)
	}
	testconfiguration.IsEqualBool(t, FileExists("invalid/file"), true)
	os.RemoveAll("invalid")
}

func TestReadLine(t *testing.T) {
	original := testconfiguration2.StartMockInputStdin("test")
	output := ReadLine()
	testconfiguration2.StopMockInputStdin(original)
	testconfiguration.IsEqualString(t, output, "test")
}

func TestGetFileSize(t *testing.T) {
	os.WriteFile("testfile", []byte(""), 0777)
	file, err := os.OpenFile("testfile", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	size, _ := GetFileSize(file)
	testconfiguration.IsEqualInt(t, int(size), 0)
	os.WriteFile("testfile", []byte("123"), 0777)
	size, _ = GetFileSize(file)
	testconfiguration.IsEqualInt(t, int(size), 3)
	file, _ = os.OpenFile("invalid", os.O_RDONLY, 0644)
	size, _ = GetFileSize(file)
	testconfiguration.IsEqualInt(t, int(size), 0)
	os.Remove("testfile")
}
