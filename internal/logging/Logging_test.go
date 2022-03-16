package logging

import (
	"fmt"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestGetIpAddress(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	test.IsEqualString(t, getIpAddress(r), "192.0.2.1")
	r = httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	test.IsEqualString(t, getIpAddress(r), "127.0.0.1")
	r.RemoteAddr = "invalid"
	test.IsEqualString(t, getIpAddress(r), "Unknown IP")
	r.Header.Add("X-REAL-IP", "1.1.1.1")
	test.IsEqualString(t, getIpAddress(r), "1.1.1.1")
	r.Header.Add("X-FORWARDED-FOR", "1.1.1.2")
	test.IsEqualString(t, getIpAddress(r), "1.1.1.2")
}

func TestInit(t *testing.T) {
	Init("test")
	test.IsEqualString(t, logPath, "test/log.txt")
}

func TestAddString(t *testing.T) {
	test.FileDoesNotExist(t, "test/log.txt")
	AddString("Hello")
	// Need sleep, as AddString() is non-blocking
	time.Sleep(500 * time.Millisecond)
	test.FileExists(t, "test/log.txt")
	content, _ := ioutil.ReadFile("test/log.txt")
	test.IsEqualBool(t, strings.Contains(string(content), "UTC   Hello"), true)
}

func TestAddDownload(t *testing.T) {
	file := models.File{
		Id:   "testId",
		Name: "testName",
	}
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("User-Agent", "testAgent")
	r.Header.Add("X-REAL-IP", "1.1.1.1")
	AddDownload(&file, r)
	// Need sleep, as AddDownload() is non-blocking
	time.Sleep(500 * time.Millisecond)
	content, _ := ioutil.ReadFile("test/log.txt")
	fmt.Println(string(content))
	test.IsEqualBool(t, strings.Contains(string(content), "UTC   Download: Filename testName, IP 1.1.1.1, ID testId, Useragent testAgent"), true)
}
