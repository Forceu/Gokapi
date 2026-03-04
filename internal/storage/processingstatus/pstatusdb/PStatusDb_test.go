package pstatusdb

import (
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
)

func TestSetStatus(t *testing.T) {
	startCleanupOnce.Do(func() {
		// Do nothing
	})
	const id = "testchunk"
	status, ok := getStatus(id)
	test.IsEqualBool(t, ok, false)
	test.IsEmpty(t, status.ChunkId)
	Set(models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: 2,
		FileId:        "testfile",
	})
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualString(t, status.FileId, "testfile")
	test.IsEqualInt(t, status.CurrentStatus, 2)
	Set(models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: 1,
	})
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 2)
	Set(models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: 3,
		FileId:        "testfile",
		ErrorMessage:  "test",
	})
	status, ok = getStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 3)
	test.IsEqualString(t, status.FileId, "testfile")
	test.IsEqualString(t, status.ErrorMessage, "test")
}

func TestGarbageCollection(t *testing.T) {
	Set(models.UploadStatus{
		ChunkId:       "toBeGarbaged",
		CurrentStatus: 2,
	})
	test.IsEqualInt(t, len(getAll()), 2)
	doGarbageCollection(false)
	test.IsEqualInt(t, len(getAll()), 2)
	status, ok := statusMap["toBeGarbaged"]
	test.IsEqualBool(t, ok, true)
	status.Creation = time.Now().Add(-30 * time.Hour).Unix()
	statusMap["toBeGarbaged"] = status
	test.IsEqualInt(t, len(getAll()), 2)
	doGarbageCollection(false)
	test.IsEqualInt(t, len(getAll()), 1)
}

func getStatus(id string) (models.UploadStatus, bool) {
	for _, status := range getAll() {
		if status.ChunkId == id {
			return status, true
		}
	}
	return models.UploadStatus{}, false
}
