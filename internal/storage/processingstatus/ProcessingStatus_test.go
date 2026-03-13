package processingstatus

import (
	"errors"
	"testing"

	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/processingstatus/pstatusdb"
	"github.com/forceu/gokapi/internal/test"
)

func TestSetStatus(t *testing.T) {
	const id = "testchunk"
	status, ok := getStatus(id)
	test.IsEqualBool(t, ok, false)
	test.IsEmpty(t, status.ChunkId)
	Set(id, 2, models.File{Id: "testfile"}, 1, nil)
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualString(t, status.FileId, "testfile")
	test.IsEqualInt(t, status.CurrentStatus, 2)
	Set(id, 1, models.File{}, 1, nil)
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 2)
	Set(id, 3, models.File{Id: "testfile"}, 1, errors.New("test"))
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 3)
	test.IsEqualString(t, status.FileId, "testfile")
	test.IsEqualString(t, status.ErrorMessage, "test")
}

func getStatus(id string) (models.UploadStatus, bool) {
	for _, status := range pstatusdb.GetAllForUser(1) {
		if status.ChunkId == id {
			return status, true
		}
	}
	return models.UploadStatus{}, false
}
