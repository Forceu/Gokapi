package downloadstatus

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
	"time"
)

var testFile models.File
var statusId string

func TestMain(m *testing.M) {
	testFile = models.File{
		Id:                 "test",
		Name:               "testName",
		Size:               "3 B",
		SHA1:               "123456",
		ExpireAt:           500,
		ExpireAtString:     "expire",
		DownloadsRemaining: 1,
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestNewDownloadStatus(t *testing.T) {
	newStatus := newDownloadStatus(models.File{Id: "testId"})
	test.IsNotEmpty(t, newStatus.Id)
	test.IsEqualString(t, newStatus.FileId, "testId")
	test.IsEqualBool(t, newStatus.ExpireAt > time.Now().Unix(), true)
}

func TestSetDownload(t *testing.T) {
	statusId = SetDownload(testFile)
	newStatus := status[statusId]
	test.IsNotEmpty(t, newStatus.Id)
	test.IsEqualString(t, newStatus.Id, statusId)
	test.IsEqualString(t, newStatus.FileId, testFile.Id)
	test.IsEqualBool(t, newStatus.ExpireAt > time.Now().Unix(), true)
}

func TestSetComplete(t *testing.T) {
	newStatus := status[statusId]
	test.IsNotEmpty(t, newStatus.Id)
	SetComplete(statusId)
	newStatus = status[statusId]
	test.IsEmpty(t, newStatus.Id)
}

func TestIsCurrentlyDownloading(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)
	test.IsEqualBool(t, IsCurrentlyDownloading(models.File{Id: "notDownloading"}), false)
}
func TestClean(t *testing.T) {
	test.IsEqualInt(t, len(status), 1)
	Clean()
	test.IsEqualInt(t, len(status), 1)
	newStatus := status[statusId]
	newStatus.ExpireAt = 1
	status[statusId] = newStatus
	test.IsEqualInt(t, len(status), 1)
	Clean()
	test.IsEqualInt(t, len(status), 0)
}

func TestGetAll(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualInt(t, len(GetAll()), len(status))
}

func TestDeleteAll(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualBool(t, len(GetAll()) != 0, true)
	DeleteAll()
	test.IsEqualInt(t, len(GetAll()), 0)

}
