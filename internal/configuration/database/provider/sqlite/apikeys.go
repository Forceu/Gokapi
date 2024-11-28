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
	Expiry       int64
	IsSystemKey  int
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)

	rows, err := p.sqliteDb.Query("SELECT * FROM ApiKeys WHERE ApiKeys.Expiry == 0 OR ApiKeys.Expiry > ?", currentTime().Unix())
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaApiKeys{}
		err = rows.Scan(&rowData.Id, &rowData.FriendlyName, &rowData.LastUsed, &rowData.Permissions, &rowData.Expiry, &rowData.IsSystemKey)
		helper.Check(err)
		result[rowData.Id] = models.ApiKey{
			Id:           rowData.Id,
			FriendlyName: rowData.FriendlyName,
			LastUsed:     rowData.LastUsed,
			Permissions:  uint8(rowData.Permissions),
			Expiry:       rowData.Expiry,
			IsSystemKey:  rowData.IsSystemKey == 1,
		}
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := p.sqliteDb.QueryRow("SELECT * FROM ApiKeys WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions, &rowResult.Expiry, &rowResult.IsSystemKey)
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
		Expiry:       rowResult.Expiry,
		IsSystemKey:  rowResult.IsSystemKey == 1,
	}

	return result, true
}

// GetSystemKey returns the latest UI API key
func (p DatabaseProvider) GetSystemKey() (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := p.sqliteDb.QueryRow("SELECT * FROM ApiKeys WHERE IsSystemKey = 1 ORDER BY Expiry DESC LIMIT 1")
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions, &rowResult.Expiry, &rowResult.IsSystemKey)
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
		Expiry:       rowResult.Expiry,
		IsSystemKey:  rowResult.IsSystemKey == 1,
	}
	return result, true
}

// SaveApiKey saves the API key to the database
func (p DatabaseProvider) SaveApiKey(apikey models.ApiKey) {
	isSystemKey := 0
	if apikey.IsSystemKey {
		isSystemKey = 1
	}
	_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO ApiKeys (Id, FriendlyName, LastUsed, Permissions, Expiry, IsSystemKey) VALUES (?, ?, ?, ?, ?, ?)",
		apikey.Id, apikey.FriendlyName, apikey.LastUsed, apikey.Permissions, apikey.Expiry, isSystemKey)
	helper.Check(err)
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	_, err := p.sqliteDb.Exec("UPDATE ApiKeys SET LastUsed = ? WHERE Id = ?",
		apikey.LastUsed, apikey.Id)
	helper.Check(err)
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	_, err := p.sqliteDb.Exec("DELETE FROM ApiKeys WHERE Id = ?", id)
	helper.Check(err)
}

func (p DatabaseProvider) cleanApiKeys() {
	_, err := p.sqliteDb.Exec("DELETE FROM ApiKeys WHERE ApiKeys.Expiry > 0 AND ApiKeys.Expiry < ?", currentTime().Unix())
	helper.Check(err)
}
