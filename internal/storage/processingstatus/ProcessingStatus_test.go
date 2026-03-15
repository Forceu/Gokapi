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

	// pstatusdb.Set is synchronous — state is committed before Set returns,
	// so no synchronisation is needed to read it back.
	// The goroutines spawned by Set (sse.PublishNewStatus and pstatusdb's GC)
	// are fire-and-forget and do not affect the assertions below.
	// Note: synctest.Test cannot be used here because pstatusdb.Set starts a
	// long-running GC goroutine (via sync.Once) that sleeps for 1 hour and
	// recurses, which permanently blocks the synctest bubble from resolving.
	Set(id, 2, models.File{Id: "testfile"}, 1, nil)
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualString(t, status.FileId, "testfile")
	test.IsEqualInt(t, status.CurrentStatus, 2)

	// Lower status must not overwrite a higher one
	Set(id, 1, models.File{}, 1, nil)
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 2)

	// Error status must be stored including the error message
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
