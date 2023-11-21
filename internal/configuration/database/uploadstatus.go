package database

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"time"
)

type schemaUploadStatus struct {
	ChunkId       string
	CurrentStatus int
	LastUpdate    int64
	CreationDate  int64
}

// GetUploadStatus returns a models.UploadStatus from the ID passed or false if the id is not valid
func GetUploadStatus(id string) (models.UploadStatus, bool) {
	result := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: 0,
		LastUpdate:    0,
	}

	var rowResult schemaUploadStatus
	row := sqliteDb.QueryRow("SELECT * FROM UploadStatus WHERE ChunkId = ?", id)
	err := row.Scan(&rowResult.ChunkId, &rowResult.CurrentStatus, &rowResult.LastUpdate, &rowResult.CreationDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UploadStatus{}, false
		}
		helper.Check(err)
		return models.UploadStatus{}, false
	}
	result.CurrentStatus = rowResult.CurrentStatus
	result.LastUpdate = rowResult.LastUpdate
	return result, true
}

// currentTime is used in order to modify the current time for testing purposes in unit tests
var currentTime = func() time.Time {
	return time.Now()
}

// SaveUploadStatus stores the upload status of a new file for 24 hours
func SaveUploadStatus(status models.UploadStatus) {
	newData := schemaUploadStatus{
		ChunkId:       status.ChunkId,
		CurrentStatus: status.CurrentStatus,
		LastUpdate:    status.LastUpdate,
		CreationDate:  currentTime().Unix(),
	}

	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO UploadStatus (ChunkId, CurrentStatus, LastUpdate, CreationDate) VALUES (?, ?, ?, ?)",
		newData.ChunkId, newData.CurrentStatus, newData.LastUpdate, newData.CreationDate)
	helper.Check(err)
}

func cleanUploadStatus() {
	_, err := sqliteDb.Exec("DELETE FROM UploadStatus WHERE CreationDate < ?", currentTime().Add(-time.Hour*24).Unix())
	helper.Check(err)
}
