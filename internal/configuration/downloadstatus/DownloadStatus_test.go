//go:build test
// +build test

package downloadstatus

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

var testFile models.File
var statusId string

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
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
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestSetDownload(t *testing.T) {
	statusId = SetDownload(testFile)
	savedStatus, ok := dataStorage.GetDownloadStatus(statusId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, savedStatus, testFile.Id)
}

func TestSetComplete(t *testing.T) {
	_, ok := dataStorage.GetDownloadStatus(statusId)
	test.IsEqualBool(t, ok, true)
	SetComplete(statusId)
	_, ok = dataStorage.GetDownloadStatus(statusId)
	test.IsEqualBool(t, ok, false)
}

func TestIsCurrentlyDownloading(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)
	test.IsEqualBool(t, IsCurrentlyDownloading(models.File{Id: "notDownloading"}), false)
}
