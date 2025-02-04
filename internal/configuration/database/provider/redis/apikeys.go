package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strings"
)

const (
	prefixApiKeys = "apikey:"
)

func dbToApiKey(id string, input []any) (models.ApiKey, error) {
	var result models.ApiKey
	err := redigo.ScanStruct(input, &result)
	result.Id = strings.Replace(id, prefixApiKeys, "", 1)
	return result, err
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)
	maps := p.getAllHashesWithPrefix(prefixApiKeys)
	for k, v := range maps {
		apiKey, err := dbToApiKey(k, v)
		helper.Check(err)
		result[apiKey.Id] = apiKey
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	result, ok := p.getHashMap(prefixApiKeys + id)
	if !ok {
		return models.ApiKey{}, false
	}
	apikey, err := dbToApiKey(id, result)
	helper.Check(err)
	return apikey, true
}

// GetSystemKey returns the latest UI API key
func (p DatabaseProvider) GetSystemKey(userId int) (models.ApiKey, bool) {
	keys := p.GetAllApiKeys()
	foundKey := ""
	var latestExpiry int64
	for _, key := range keys {
		if !key.IsSystemKey {
			continue
		}
		if key.UserId != userId {
			continue
		}
		if key.Expiry > latestExpiry {
			foundKey = key.Id
			latestExpiry = key.Expiry
		}
	}
	if foundKey == "" {
		return models.ApiKey{}, false
	}
	return keys[foundKey], true
}

// GetApiKeyByPublicKey returns an API key by using the public key
func (p DatabaseProvider) GetApiKeyByPublicKey(publicKey string) (string, bool) {
	keys := p.GetAllApiKeys()
	for _, key := range keys {
		if key.PublicId == publicKey {
			return key.Id, true
		}
	}
	return "", false
}

// SaveApiKey saves the API key to the database
func (p DatabaseProvider) SaveApiKey(apikey models.ApiKey) {
	p.setHashMap(p.buildArgs(prefixApiKeys + apikey.Id).AddFlat(apikey))
	if apikey.Expiry != 0 {
		p.setExpiryAt(prefixApiKeys+apikey.Id, apikey.Expiry)
	}
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	p.SaveApiKey(apikey)
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	p.deleteKey(prefixApiKeys + id)
}
