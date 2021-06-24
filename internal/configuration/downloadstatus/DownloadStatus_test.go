// +build test

package downloadstatus

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
	settings.DownloadStatus = make(map[string]models.DownloadStatus)
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

func TestNewDownloadStatus(t *testing.T) {
	status := newDownloadStatus(models.File{Id: "testId"})
	test.IsNotEmpty(t, status.Id)
	test.IsEqualString(t, status.FileId, "testId")
	test.IsEqualBool(t, status.ExpireAt > time.Now().Unix(), true)
}

func TestSetDownload(t *testing.T) {
	statusId = SetDownload(testFile)
	settings := configuration.GetServerSettings()
	status := settings.DownloadStatus[statusId]
	configuration.Release()
	test.IsNotEmpty(t, status.Id)
	test.IsEqualString(t, status.Id, statusId)
	test.IsEqualString(t, status.FileId, testFile.Id)
	test.IsEqualBool(t, status.ExpireAt > time.Now().Unix(), true)
}

func TestSetComplete(t *testing.T) {
	settings := configuration.GetServerSettings()
	status := settings.DownloadStatus[statusId]
	configuration.Release()
	test.IsNotEmpty(t, status.Id)
	SetComplete(statusId)
	status = settings.DownloadStatus[statusId]
	test.IsEmpty(t, status.Id)
}

func TestIsCurrentlyDownloading(t *testing.T) {
	statusId = SetDownload(testFile)
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile, settings), true)
	test.IsEqualBool(t, IsCurrentlyDownloading(models.File{Id: "notDownloading"}, settings), false)
}

func TestClean(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualInt(t, len(settings.DownloadStatus), 1)
	Clean()
	test.IsEqualInt(t, len(settings.DownloadStatus), 1)
	status := settings.DownloadStatus[statusId]
	status.ExpireAt = 1
	settings.DownloadStatus[statusId] = status
	test.IsEqualInt(t, len(settings.DownloadStatus), 1)
	Clean()
	test.IsEqualInt(t, len(settings.DownloadStatus), 0)
}
