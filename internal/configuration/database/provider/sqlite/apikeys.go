package sqlite

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaApiKeys struct {
	Id           string
	FriendlyName string
	LastUsed     int64
	Permissions  int
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)

	rows, err := sqliteDb.Query("SELECT * FROM ApiKeys")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaApiKeys{}
		err = rows.Scan(&rowData.Id, &rowData.FriendlyName, &rowData.LastUsed, &rowData.Permissions)
		helper.Check(err)
		result[rowData.Id] = models.ApiKey{
			Id:           rowData.Id,
			FriendlyName: rowData.FriendlyName,
			LastUsed:     rowData.LastUsed,
			Permissions:  uint8(rowData.Permissions),
		}
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := sqliteDb.QueryRow("SELECT * FROM ApiKeys WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ApiKey{}, false
		}
		helper.Check(err)
		return models.ApiKey{}, false
	}

	result := models.ApiKey{
		Id:           rowResult.Id,
		FriendlyName: rowResult.FriendlyName,
		LastUsed:     rowResult.LastUsed,
		Permissions:  uint8(rowResult.Permissions),
	}

	return result, true
}

// SaveApiKey saves the API key to the database
func (p DatabaseProvider) SaveApiKey(apikey models.ApiKey) {
	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO ApiKeys (Id, FriendlyName, LastUsed, Permissions) VALUES (?, ?, ?, ?)",
		apikey.Id, apikey.FriendlyName, apikey.LastUsed, apikey.Permissions)
	helper.Check(err)
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	_, err := sqliteDb.Exec("UPDATE ApiKeys SET LastUsed = ? WHERE Id = ?",
		apikey.LastUsed, apikey.Id)
	helper.Check(err)
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	_, err := sqliteDb.Exec("DELETE FROM ApiKeys WHERE Id = ?", id)
	helper.Check(err)
}
