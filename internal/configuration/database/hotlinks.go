package database

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaHotlinks struct {
	Id     string
	FileId string
}

// GetHotlink returns the id of the file associated or false if not found
func GetHotlink(id string) (string, bool) {
	var rowResult schemaHotlinks
	row := sqliteDb.QueryRow("SELECT FileId FROM Hotlinks WHERE Id = ?", id)
	err := row.Scan(&rowResult.FileId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false
		}
		helper.Check(err)
		return "", false
	}
	return rowResult.FileId, true
}

// SaveHotlink stores the hotlink associated with the file in the database
func SaveHotlink(file models.File) {

	newData := schemaHotlinks{
		Id:     file.HotlinkId,
		FileId: file.Id,
	}

	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO Hotlinks (Id, FileId) VALUES (?, ?)",
		newData.Id, newData.FileId)
	helper.Check(err)
}

// DeleteHotlink deletes a hotlink with the given ID
func DeleteHotlink(id string) {
	_, err := sqliteDb.Exec("DELETE FROM Hotlinks WHERE Id = ?", id)
	helper.Check(err)
}
