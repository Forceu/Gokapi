package sqlite

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaUploadConfig struct {
	Id                 int64
	Downloads          int
	TimeExpiry         int
	Password           string
	UnlimitedDownloads int
	UnlimitedTime      int
}

// GetUploadDefaults returns the last used setting for amount of downloads allowed, last expiry in days and
// a password for the file
func (p DatabaseProvider) GetUploadDefaults() models.LastUploadValues {
	defaultValues := models.LastUploadValues{
		Downloads:         1,
		TimeExpiry:        14,
		Password:          "",
		UnlimitedDownload: false,
		UnlimitedTime:     false,
	}

	rowResult := schemaUploadConfig{}
	row := sqliteDb.QueryRow("SELECT * FROM UploadConfig WHERE id = 1")
	err := row.Scan(&rowResult.Id, &rowResult.Downloads, &rowResult.TimeExpiry, &rowResult.Password, &rowResult.UnlimitedDownloads, &rowResult.UnlimitedTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValues
		}
		helper.Check(err)
		return defaultValues
	}

	result := models.LastUploadValues{
		Downloads:         rowResult.Downloads,
		TimeExpiry:        rowResult.TimeExpiry,
		Password:          rowResult.Password,
		UnlimitedDownload: rowResult.UnlimitedDownloads == 1,
		UnlimitedTime:     rowResult.UnlimitedTime == 1,
	}
	return result
}

// SaveUploadDefaults saves the last used setting for an upload
func (p DatabaseProvider) SaveUploadDefaults(values models.LastUploadValues) {

	newData := schemaUploadConfig{
		Downloads:  values.Downloads,
		TimeExpiry: values.TimeExpiry,
		Password:   values.Password,
	}
	if values.UnlimitedDownload {
		newData.UnlimitedDownloads = 1
	}
	if values.UnlimitedTime {
		newData.UnlimitedTime = 1
	}

	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO UploadConfig (id, Downloads,TimeExpiry,Password,UnlimitedDownloads,UnlimitedTime) VALUES (1, ?, ?, ?, ?, ?)",
		newData.Downloads, newData.TimeExpiry, newData.Password, newData.UnlimitedDownloads, newData.UnlimitedTime)
	helper.Check(err)
}
