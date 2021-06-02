package history

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
	"time"
)

var testFile models.File
var statusId string

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	settings := configuration.GetServerSettings()
	settings.DownloadHistory = make(map[string]models.LogHistory)
	testFile = models.File{
		Id:                 "test",
		Name:               "testName",
		Size:               "3 B",
		SHA256:             "123456",
		ExpireAt:           500,
		ExpireAtString:     "expire",
		DownloadsRemaining: 1,
	}
	configuration.Release()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestnewDownloadHistory(t *testing.T) {
	r := httptest.NewRequest("GET", "/upload", nil)

	h := newDownloadHistory(models.File{Id: "testId"}, r)
	test.IsNotEmpty(t, h.Id)
	test.IsEqualString(t, h.FileId, "testId")
	test.IsEqualBool(t, h.DownloadDate < time.Now().Unix(), true)
}

func TestLogHistory(t *testing.T) {
	r := httptest.NewRequest("GET", "/upload", nil)

	statusId = LogHistory(testFile, r)
	settings := configuration.GetServerSettings()
	status := settings.DownloadStatus[statusId]
	configuration.Release()
	test.IsNotEmpty(t, status.Id)
	test.IsEqualString(t, status.Id, statusId)
	test.IsEqualString(t, status.FileId, testFile.Id)
	test.IsEqualString(t, status.DownloaderUA, r.UserAgent())
	test.IsEqualString(t, status.DownloaderIP, r.RemoteAddr)

	test.IsEqualBool(t, status.ExpireAt < time.Now().Unix(), true)
}

