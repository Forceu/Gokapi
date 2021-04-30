package downloadstatus

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"os"
	"testing"
	"time"
)

var testFile models.File
var statusId string

func TestMain(m *testing.M) {
	configuration.ServerSettings.DownloadStatus = make(map[string]models.DownloadStatus)
	testFile = models.File{
		Id:                 "test",
		Name:               "testName",
		Size:               "3 B",
		SHA256:             "123456",
		ExpireAt:           500,
		ExpireAtString:     "expire",
		DownloadsRemaining: 1,
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestNewDownloadStatus(t *testing.T) {
	status := newDownloadStatus(models.File{Id: "testId"})
	test.IsNotEmpty(t, status.Id)
	test.IsEqualString(t, status.FileId, "testId")
	test.IsEqualBool(t, status.ExpireAt > time.Now().Unix(), true)
}

func TestSetDownload(t *testing.T) {
	statusId = SetDownload(testFile)
	status := configuration.ServerSettings.DownloadStatus[statusId]
	test.IsNotEmpty(t, status.Id)
	test.IsEqualString(t, status.Id, statusId)
	test.IsEqualString(t, status.FileId, testFile.Id)
	test.IsEqualBool(t, status.ExpireAt > time.Now().Unix(), true)
}

func TestSetComplete(t *testing.T) {
	status := configuration.ServerSettings.DownloadStatus[statusId]
	test.IsNotEmpty(t, status.Id)
	SetComplete(statusId)
	status = configuration.ServerSettings.DownloadStatus[statusId]
	test.IsEmpty(t, status.Id)
}

func TestIsCurrentlyDownloading(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)
	test.IsEqualBool(t, IsCurrentlyDownloading(models.File{Id: "notDownloading"}), false)
}

func TestClean(t *testing.T) {
	test.IsEqualInt(t, len(configuration.ServerSettings.DownloadStatus), 1)
	Clean()
	test.IsEqualInt(t, len(configuration.ServerSettings.DownloadStatus), 1)
	status := configuration.ServerSettings.DownloadStatus[statusId]
	status.ExpireAt = 1
	configuration.ServerSettings.DownloadStatus[statusId] = status
	test.IsEqualInt(t, len(configuration.ServerSettings.DownloadStatus), 1)
	Clean()
	test.IsEqualInt(t, len(configuration.ServerSettings.DownloadStatus), 0)
}
