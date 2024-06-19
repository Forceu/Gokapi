package database

import (
	"database/sql"
	"errors"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaUploadTokens struct {
	Id             string
	FriendlyName   string
	LastUsed       int64
	LastUsedString string
	Permissions    int
}

// GetAllUploadTokens returns a map with all upload tokens
func GetAllUploadTokens() map[string]models.UploadToken {
	result := make(map[string]models.UploadToken)

	rows, err := sqliteDb.Query("SELECT * FROM UploadTokens")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaUploadTokens{}
		err = rows.Scan(&rowData.Id, &rowData.LastUsed, &rowData.LastUsedString)
		helper.Check(err)
		result[rowData.Id] = models.UploadToken{
			Id:             rowData.Id,
			LastUsed:       rowData.LastUsed,
			LastUsedString: rowData.LastUsedString,
		}
	}
	return result
}

// GetUploadToken returns a models.UploadToken if valid or false if the ID is not valid
func GetUploadToken(id string) (models.UploadToken, bool) {
	var rowResult schemaUploadTokens
	row := sqliteDb.QueryRow("SELECT * FROM UploadTokens WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.LastUsed, &rowResult.LastUsedString)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UploadToken{}, false
		}
		helper.Check(err)
		return models.UploadToken{}, false
	}

	result := models.UploadToken{
		Id:             rowResult.Id,
		LastUsed:       rowResult.LastUsed,
		LastUsedString: rowResult.LastUsedString,
	}

	return result, true
}

// SaveUploadToken saves the upload token to the database
func SaveUploadToken(uploadToken models.UploadToken) {
	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO UploadTokens (Id, LastUsed, LastUsedString) VALUES (?, ?, ?)",
		uploadToken.Id, uploadToken.LastUsed, uploadToken.LastUsedString)
	helper.Check(err)
}

// UpdateTimeUploadToken writes the content of LastUsage to the database
func UpdateTimeUploadToken(uploadToken models.UploadToken) {
	_, err := sqliteDb.Exec("UPDATE UploadTokens SET LastUsed = ?, LastUsedString = ? WHERE Id = ?",
		uploadToken.LastUsed, uploadToken.LastUsedString, uploadToken.Id)
	helper.Check(err)
}

// DeleteUploadToken deletes an upload token with the given ID
func DeleteUploadToken(id string) {
	_, err := sqliteDb.Exec("DELETE FROM UploadTokens WHERE Id = ?", id)
	helper.Check(err)
}
