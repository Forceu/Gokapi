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
	newStatus := statusMap[statusId]
	test.IsNotEmpty(t, newStatus.Id)
	test.IsEqualString(t, newStatus.Id, statusId)
	test.IsEqualString(t, newStatus.FileId, testFile.Id)
	test.IsEqualBool(t, newStatus.ExpireAt > time.Now().Unix(), true)
}

func TestSetComplete(t *testing.T) {
	newStatus := statusMap[statusId]
	test.IsNotEmpty(t, newStatus.Id)
	SetComplete(statusId)
	newStatus = statusMap[statusId]
	test.IsEmpty(t, newStatus.Id)
}

func TestIsCurrentlyDownloading(t *testing.T) {
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), false)
	statusIdFirst := SetDownload(testFile)
	firstStatus := statusMap[statusIdFirst]
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)
	statusIdSecond := SetDownload(testFile)
	secondStatus := statusMap[statusIdSecond]
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)

	firstStatus.ExpireAt = 0
	statusMap[firstStatus.Id] = firstStatus
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), true)
	secondStatus.ExpireAt = 0
	statusMap[secondStatus.Id] = secondStatus
	test.IsEqualBool(t, IsCurrentlyDownloading(testFile), false)

	statusId = SetDownload(testFile)
	test.IsEqualBool(t, IsCurrentlyDownloading(models.File{Id: "notDownloading"}), false)
}
func TestClean(t *testing.T) {
	test.IsEqualInt(t, len(statusMap), 3)
	Clean()
	test.IsEqualInt(t, len(statusMap), 1)
	newStatus := statusMap[statusId]
	newStatus.ExpireAt = 1
	statusMap[statusId] = newStatus
	test.IsEqualInt(t, len(statusMap), 1)
	Clean()
	test.IsEqualInt(t, len(statusMap), 0)
}

func TestDeleteAll(t *testing.T) {
	statusId = SetDownload(testFile)
	test.IsEqualBool(t, len(statusMap) != 0, true)
	DeleteAll()
	test.IsEqualInt(t, len(statusMap), 0)
}

func TestSetAllComplete(t *testing.T) {
	test.IsEqualInt(t, len(statusMap), 0)
	SetDownload(models.File{Id: "stillDownloading"})
	SetDownload(models.File{Id: "stillDownloading"})
	status1 := SetDownload(models.File{Id: "stillDownloading"})
	SetDownload(models.File{Id: "fileToBeDeleted"})
	SetDownload(models.File{Id: "fileToBeDeleted"})
	SetDownload(models.File{Id: "fileToBeDeleted"})
	status2 := SetDownload(models.File{Id: "fileToBeDeleted"})
	test.IsEqualInt(t, len(statusMap), 7)
	SetAllComplete("fileToBeDeleted")
	test.IsEqualInt(t, len(statusMap), 3)
	_, ok := statusMap[status1]
	test.IsEqualBool(t, ok, true)
	_, ok = statusMap[status2]
	test.IsEqualBool(t, ok, false)
}
