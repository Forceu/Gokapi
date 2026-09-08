package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaApiKeys struct {
	Id              string
	FriendlyName    string
	LastUsed        int64
	Permissions     int
	Expiry          int64
	IsSystemKey     int
	UserId          int
	PublicId        string
	UploadRequestId string
}

// currentTime is used in order to modify the current time for testing purposes in unit tests
var currentTime = func() time.Time {
	return time.Now()
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)

	rows, err := p.sqlDb.Query("SELECT * FROM apikeys WHERE apikeys.expiry = 0 OR apikeys.expiry > $1", currentTime().Unix())
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaApiKeys{}
		err = rows.Scan(&rowData.Id, &rowData.FriendlyName, &rowData.LastUsed, &rowData.Permissions, &rowData.Expiry,
			&rowData.IsSystemKey, &rowData.UserId, &rowData.PublicId, &rowData.UploadRequestId)
		helper.Check(err)
		result[rowData.Id] = models.ApiKey{
			Id:              rowData.Id,
			PublicId:        rowData.PublicId,
			FriendlyName:    rowData.FriendlyName,
			LastUsed:        rowData.LastUsed,
			Permissions:     models.ApiPermission(rowData.Permissions),
			Expiry:          rowData.Expiry,
			IsSystemKey:     rowData.IsSystemKey == 1,
			UserId:          rowData.UserId,
			UploadRequestId: rowData.UploadRequestId,
		}
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	var rowResult schemaApiKeys
	row := p.sqlDb.QueryRow("SELECT * FROM apikeys WHERE id = $1", id)
	err := row.Scan(&rowResult.Id, &rowResult.FriendlyName, &rowResult.LastUsed, &rowResult.Permissions, &rowResult.Expiry,
		&rowResult.IsSystemKey, &rowResult.UserId, &rowResult.PublicId, &rowResult.UploadRequestId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ApiKey{}, false
		}
		helper.Check(err)
		return models.ApiKey{}, false
	}

	result := models.ApiKey{
		Id:              rowResult.Id,
		PublicId:        rowResult.PublicId,
		FriendlyName:    rowResult.FriendlyName,
		LastUsed:        rowResult.LastUsed,
		Permissions:     models.ApiPermission(rowResult.Permissions),
		Expiry:          rowResult.Expiry,
		IsSystemKey:     rowResult.IsSystemKey == 1,
		UserId:          rowResult.UserId,
		UploadRequestId: rowResult.UploadRequestId,
	}

	return result, true
}

// GetApiKeyByPublicKey returns an API key by using the public key
func (p DatabaseProvider) GetApiKeyByPublicKey(publicKey string) (string, bool) {
	var rowResult schemaApiKeys
	row := p.sqlDb.QueryRow("SELECT id FROM apikeys WHERE publicid = $1 LIMIT 1", publicKey)
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
	_, err := p.sqlDb.Exec(`INSERT INTO apikeys (id, friendlyname, lastused, permissions, expiry, issystemkey, userid, publicid, uploadrequestid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET friendlyname = $2, lastused = $3, permissions = $4, expiry = $5,
			issystemkey = $6, userid = $7, publicid = $8, uploadrequestid = $9`,
		apikey.Id, apikey.FriendlyName, apikey.LastUsed, apikey.Permissions, apikey.Expiry, isSystemKey, apikey.UserId, apikey.PublicId, apikey.UploadRequestId)
	helper.Check(err)
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	_, err := p.sqlDb.Exec("UPDATE apikeys SET lastused = $1 WHERE id = $2",
		apikey.LastUsed, apikey.Id)
	helper.Check(err)
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	_, err := p.sqlDb.Exec("DELETE FROM apikeys WHERE id = $1", id)
	helper.Check(err)
}

func (p DatabaseProvider) cleanApiKeys() {
	_, err := p.sqlDb.Exec("DELETE FROM apikeys WHERE apikeys.expiry > 0 AND apikeys.expiry < $1", currentTime().Unix())
	helper.Check(err)
}
