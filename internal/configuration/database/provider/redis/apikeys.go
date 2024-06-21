package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixApiKeys = "apikey:"
)

func dbToApiKey(id string, input []any) (models.ApiKey, error) {
	var result models.ApiKey
	err := redigo.ScanStruct(input, &result)
	result.Id = id
	return result, err
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)
	maps := getAllHashesWithPrefix(prefixApiKeys)
	for _, m := range maps {
		apiKey, err := dbToApiKey(m.Key, m.Values)
		helper.Check(err)
		result[apiKey.Id] = apiKey
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func (p DatabaseProvider) GetApiKey(id string) (models.ApiKey, bool) {
	result, ok := getHashMap(prefixApiKeys + id)
	if !ok {
		return models.ApiKey{}, false
	}
	apikey, err := dbToApiKey(id, result)
	helper.Check(err)
	return apikey, true
}

// SaveApiKey saves the API key to the database
func (p DatabaseProvider) SaveApiKey(apikey models.ApiKey) {
	setHashMapArgs(buildArgs(prefixApiKeys + apikey.Id).AddFlat(apikey))
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	p.SaveApiKey(apikey)
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	deleteKey(prefixApiKeys + id)
}
