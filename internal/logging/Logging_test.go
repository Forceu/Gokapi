package logging

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestGetIpAddress(t *testing.T) {
	Init("test")
	r := httptest.NewRequest("GET", "/test", nil)
	test.IsEqualString(t, GetIpAddress(r), "192.0.2.1")
	r = httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	test.IsEqualString(t, GetIpAddress(r), "127.0.0.1")
	r.RemoteAddr = "invalid"
	test.IsEqualString(t, GetIpAddress(r), "invalid")
	r.Header.Add("X-REAL-IP", "invalid")
	test.IsEqualString(t, GetIpAddress(r), "invalid")
	r.Header.Add("X-FORWARDED-FOR", "invalid")
	test.IsEqualString(t, GetIpAddress(r), "invalid")
	r.RemoteAddr = "127.0.0.1"
	r.Header.Del("X-REAL-IP")
	r.Header.Del("X-FORWARDED-FOR")
	test.IsEqualString(t, GetIpAddress(r), "127.0.0.1")
	r.Header.Add("X-REAL-IP", "1.1.1.1")
	test.IsEqualString(t, GetIpAddress(r), "1.1.1.1")
	r.Header.Add("X-FORWARDED-FOR", "1.1.1.1, 2.2.2.2")
	test.IsEqualString(t, GetIpAddress(r), "2.2.2.2")
	useCloudflare = true
	r.Header.Add("CF-Connecting-IP", "3.3.3.3")
	test.IsEqualString(t, GetIpAddress(r), "3.3.3.3")
}

func TestInit(t *testing.T) {
	Init("test")
	test.IsEqualString(t, logPath, "test/log.txt")
}

func TestAddString(t *testing.T) {
	test.FileDoesNotExist(t, "test/log.txt")
	createLogEntry(categoryInfo, "Hello", true)
	test.FileExists(t, "test/log.txt")
	content, _ := os.ReadFile("test/log.txt")
	test.IsEqualBool(t, strings.Contains(string(content), "UTC   [info] Hello"), true)
}

func TestAddDownload(t *testing.T) {
	file := models.File{
		Id:   "testId",
		Name: "testName",
	}
	r := httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "127.0.0.1"
	r.Header.Set("User-Agent", "testAgent")
	r.Header.Add("X-REAL-IP", "1.1.1.1")
	LogDownload(file, r, true)
	// Need sleep, as LogDownload() is non-blocking
	time.Sleep(500 * time.Millisecond)
	content, _ := os.ReadFile("test/log.txt")
	test.IsEqualBool(t, strings.Contains(string(content), "UTC   [download] testName, IP 1.1.1.1, ID testId, Useragent testAgent"), true)
	r.Header.Add("X-REAL-IP", "2.2.2.2")
	LogDownload(file, r, false)
	// Need sleep, as LogDownload() is non-blocking
	time.Sleep(500 * time.Millisecond)
	content, _ = os.ReadFile("test/log.txt")
	test.IsEqualBool(t, strings.Contains(string(content), "2.2.2.2"), false)
}

func TestLogInvalidDownloadPassword(t *testing.T) {
	for _, saveIp := range []bool{false, true} {
		id := "audit-without-ip"
		if saveIp {
			id = "audit-with-ip"
		}
		r := httptest.NewRequest("POST", "/d?id="+id, strings.NewReader("password=never-log-this-secret"))
		r.RemoteAddr = "192.0.2.42:1234"
		r.Header.Set("X-Forwarded-For", "198.51.100.99")
		r.Header.Set("User-Agent", "TestAgent\r\n[auth] injected")
		LogInvalidDownloadPassword(id, r, saveIp)
		// Log entries are written asynchronously, as with successful downloads.
		deadline := time.Now().Add(2 * time.Second)
		var entry string
		for time.Now().Before(deadline) {
			content, _ := os.ReadFile(logPath)
			for _, line := range strings.Split(string(content), "\n") {
				if strings.Contains(line, "Invalid download password for file ID "+id) {
					entry = line
				}
			}
			if entry != "" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if entry == "" {
			t.Fatal("missing failed download password audit entry")
		}
		test.IsEqualBool(t, strings.Contains(entry, "[auth] Invalid download password"), true)
		test.IsEqualBool(t, strings.Contains(entry, "192.0.2.42"), saveIp)
		test.IsEqualBool(t, strings.Contains(entry, "198.51.100.99"), false)
		test.IsEqualBool(t, strings.Contains(entry, "never-log-this-secret"), false)
		test.IsEqualBool(t, strings.Contains(entry, "Useragent TestAgentauth injected"), true)
	}
}
