package sqlite

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"time"
)

type schemaApiKeys struct {
	Id           string
	FriendlyName string
	LastUsed     int64
	Permissions  int
	Expiry       int64
	IsSystemKey  int
	UserId       int
	PublicId     string
}

// currentTime is used in order to modify the current time for testing purposes in unit tests
var currentTime = func() time.Time {
	return time.Now()
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)

	rows, err := p.sqliteDb.Query("SELECT * FROM ApiKeys WHERE ApiKeys.Expiry == 0 OR ApiKeys.Expiry > ?", currentTime().Unix())
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaApiKeys{}
		err = rows.Scan(&rowData.Id, &rowData.FriendlyName, &rowData.LastUsed, &rowData.Permissions, &rowData.Expiry, &rowData.IsSystemKey, &rowData.UserId, &rowData.PublicId)
		helper.Check(err)
		result[rowData.Id] = models.ApiKey{
			Id:           rowData.Id,
			PublicId:     rowData.PublicId,
			FriendlyName: rowData.FriendlyName,
			LastUsed:     rowData.LastUsed,
			Permissions:  models.ApiPermission(rowData.Permissions),
			Expiry:       rowData.Expiry,
			IsSystemKey:  rowData.IsSystemKey == 1,
			UserId:       rowData.UserId,
		}
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := p.sqliteDb.QueryRow("SELECT * FROM ApiKeys WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions, &rowResult.Expiry, &rowResult.IsSystemKey, &rowResult.UserId, &rowResult.PublicId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ApiKey{}, false
		}
		helper.Check(err)
		return models.ApiKey{}, false
	}

	result := models.ApiKey{
		Id:           rowResult.Id,
		PublicId:     rowResult.PublicId,
		FriendlyName: rowResult.FriendlyName,
		LastUsed:     rowResult.LastUsed,
		Permissions:  models.ApiPermission(rowResult.Permissions),
		Expiry:       rowResult.Expiry,
		IsSystemKey:  rowResult.IsSystemKey == 1,
		UserId:       rowResult.UserId,
	}

	return result, true
}

// GetSystemKey returns the latest UI API key
func (p DatabaseProvider) GetSystemKey(userId int) (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := p.sqliteDb.QueryRow("SELECT * FROM ApiKeys WHERE IsSystemKey = 1 AND UserId = ? ORDER BY Expiry DESC LIMIT 1", userId)
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions, &rowResult.Expiry, &rowResult.IsSystemKey, &rowResult.UserId, &rowResult.PublicId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ApiKey{}, false
		}
		helper.Check(err)
		return models.ApiKey{}, false
	}

	result := models.ApiKey{
		Id:           rowResult.Id,
		PublicId:     rowResult.PublicId,
		FriendlyName: rowResult.FriendlyName,
		LastUsed:     rowResult.LastUsed,
		Permissions:  models.ApiPermission(rowResult.Permissions),
		Expiry:       rowResult.Expiry,
		IsSystemKey:  rowResult.IsSystemKey == 1,
		UserId:       rowResult.UserId,
	}
	return result, true
}

// GetApiKeyByPublicKey returns an API key by using the public key
func (p DatabaseProvider) GetApiKeyByPublicKey(publicKey string) (string, bool) {
	var rowResult schemaApiKeys
	row := p.sqliteDb.QueryRow("SELECT Id FROM ApiKeys WHERE PublicId = ? LIMIT 1", publicKey)
	err := row.Scan(&rowResult.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false
		}
		helper.Check(err)
		return "", false
	}
	return rowResult.Id, true
}

// SaveApiKey saves the API key to the database
func (p DatabaseProvider) SaveApiKey(apikey models.ApiKey) {
	isSystemKey := 0
	if apikey.IsSystemKey {
		isSystemKey = 1
	}
	_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO ApiKeys (Id, FriendlyName, LastUsed, Permissions, Expiry, IsSystemKey, UserId, PublicId) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		apikey.Id, apikey.FriendlyName, apikey.LastUsed, apikey.Permissions, apikey.Expiry, isSystemKey, apikey.UserId, apikey.PublicId)
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
