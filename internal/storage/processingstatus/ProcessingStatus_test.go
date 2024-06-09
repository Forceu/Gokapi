package processingstatus

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestSetStatus(t *testing.T) {

	chunkID := "testChunkID"
	testCases := []struct {
		name          string
		initialStatus int
		newStatus     int
	}{
		{"SetNewStatus", -1, StatusHashingOrEncrypting},
		{"SetSameStatus", StatusUploading, StatusUploading},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the initial status for the chunk ID
			initialStatus := models.UploadStatus{
				ChunkId:       chunkID,
				CurrentStatus: tc.initialStatus,
			}
			database.SaveUploadStatus(initialStatus)

			// Set the new status
			Set(chunkID, tc.newStatus)

			// Wait for SSE event to be published
			time.Sleep(100 * time.Millisecond)

			// Retrieve the updated status from the database
			updatedStatus, _ := database.GetUploadStatus(chunkID)

			// Check if the status was updated
			test.IsEqualInt(t, tc.newStatus, updatedStatus.CurrentStatus)
		})
	}
}
