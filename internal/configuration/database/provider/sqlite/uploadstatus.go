package sqlite

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
	CreationDate  int64
	FileId        sql.NullString
}

// GetAllUploadStatus returns all UploadStatus values from the past 24 hours
func (p DatabaseProvider) GetAllUploadStatus() []models.UploadStatus {
	var result = make([]models.UploadStatus, 0)
	rows, err := p.sqliteDb.Query("SELECT * FROM UploadStatus")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowResult := schemaUploadStatus{}
		err = rows.Scan(&rowResult.ChunkId, &rowResult.CurrentStatus, &rowResult.CreationDate, &rowResult.FileId)
		helper.Check(err)
		result = append(result, models.UploadStatus{
			ChunkId:       rowResult.ChunkId,
			CurrentStatus: rowResult.CurrentStatus,
			FileId:        rowResult.FileId.String,
		})
	}
	return result
}

// GetUploadStatus returns a models.UploadStatus from the ID passed or false if the id is not valid
func (p DatabaseProvider) GetUploadStatus(id string) (models.UploadStatus, bool) {
	result := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: 0,
	}

	var rowResult schemaUploadStatus
	row := p.sqliteDb.QueryRow("SELECT * FROM UploadStatus WHERE ChunkId = ?", id)
	err := row.Scan(&rowResult.ChunkId, &rowResult.CurrentStatus, &rowResult.CreationDate, &rowResult.FileId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UploadStatus{}, false
		}
		helper.Check(err)
		return models.UploadStatus{}, false
	}
	result.CurrentStatus = rowResult.CurrentStatus
	result.FileId = rowResult.FileId.String
	return result, true
}

// currentTime is used in order to modify the current time for testing purposes in unit tests
var currentTime = func() time.Time {
	return time.Now()
}

// SaveUploadStatus stores the upload status of a new file for 24 hours
func (p DatabaseProvider) SaveUploadStatus(status models.UploadStatus) {
	_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO UploadStatus (ChunkId, CurrentStatus, CreationDate, FileId) VALUES (?, ?, ?, ?)",
		status.ChunkId, status.CurrentStatus, currentTime().Unix(), status.FileId)
	helper.Check(err)
}

func (p DatabaseProvider) cleanUploadStatus() {
	_, err := p.sqliteDb.Exec("DELETE FROM UploadStatus WHERE CreationDate < ?", currentTime().Add(-time.Hour*24).Unix())
	helper.Check(err)
}
