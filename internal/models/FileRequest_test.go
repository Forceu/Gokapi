package models

import (
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/test"
)

func TestFileRequest_PopulateAndHelpers(t *testing.T) {
	now := time.Now().Unix()

	files := map[string]File{
		"file1": {
			Id:              "file1",
			UploadRequestId: "req1",
			SizeBytes:       1000,
			UploadDate:      now - 100,
		},
		"file2": {
			Id:              "file2",
			UploadRequestId: "req1",
			SizeBytes:       2000,
			UploadDate:      now,
		},
		"file3": {
			Id:              "file3",
			UploadRequestId: "other",
			SizeBytes:       9999,
			UploadDate:      now,
		},
	}

	fr := &FileRequest{
		Id:       "req1",
		MaxFiles: 5,
		MaxSize:  10,
	}

	fr.Populate(files, 8)
	test.IsEqualInt(t, fr.UploadedFiles, 2)
	test.IsEqualInt(t, fr.MaxFiles, 5)
	test.IsEqualInt(t, fr.CombinedMaxSize, 8)
	test.IsEqualInt(t, fr.FilesRemaining(), 3)

	test.IsEqualInt64(t, fr.TotalFileSize, int64(3000))
	test.IsEqualInt64(t, fr.LastUpload, now)
	test.IsEqualInt(t, len(fr.FileIdList), 2)

	test.IsNotEqualString(t, fr.GetReadableDateLastUpdate(), "None")
	test.IsNotEqualString(t, fr.GetFilesAsString(), "")

	fr = &FileRequest{
		Id:            "req2",
		UploadedFiles: 5,
		MaxFiles:      2,
		TotalFileSize: 102400,
	}
	test.IsEqualInt(t, fr.FilesRemaining(), 0)
	test.IsEqualString(t, fr.GetReadableDateLastUpdate(), "None")
	test.IsEqualString(t, fr.GetReadableTotalSize(), "100.0 kB")

}

func TestFileRequest_UnlimitedFlags(t *testing.T) {
	fr := &FileRequest{
		MaxFiles: 0,
		MaxSize:  0,
		Expiry:   0,
	}

	test.IsEqualBool(t, fr.IsUnlimitedFiles(), true)
	test.IsEqualBool(t, fr.IsUnlimitedSize(), true)
	test.IsEqualBool(t, fr.IsUnlimitedTime(), true)
	test.IsEqualBool(t, !fr.HasRestrictions(), true)
}

func TestFileRequest_IsExpired(t *testing.T) {
	fr := &FileRequest{
		Expiry: time.Now().Unix() - 10,
	}

	test.IsEqualBool(t, fr.IsExpired(), true)
}
