package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"strconv"
)

const (
	prefixApiKeys               = "apikey:"
	hashmapApiKeyFriendlyName   = "fn:"
	hashmapApiKeyLastUsed       = "lu:"
	hashmapApiKeyLastUsedString = "lus:"
	hashmapApiKeyPermissions    = "perm:"
)

func dbToApiKey(id string, input map[string]string) (models.ApiKey, error) {
	lastUsed, err := strconv.ParseInt(input[hashmapApiKeyLastUsed], 10, 64)
	if err != nil {
		return models.ApiKey{}, err
	}
	permissions, err := strconv.ParseInt(input[hashmapApiKeyPermissions], 10, 8)
	if err != nil {
		return models.ApiKey{}, err
	}
	return models.ApiKey{
		Id:             id,
		FriendlyName:   input[hashmapApiKeyFriendlyName],
		LastUsedString: input[hashmapApiKeyLastUsedString],
		LastUsed:       lastUsed,
		Permissions:    uint8(permissions),
	}, nil
}

func apiKeyToDb(input models.ApiKey) map[string]string {
	return map[string]string{
		hashmapApiKeyFriendlyName:   input.FriendlyName,
		hashmapApiKeyLastUsed:       strconv.FormatInt(input.LastUsed, 10),
		hashmapApiKeyLastUsedString: input.LastUsedString,
		hashmapApiKeyPermissions:    strconv.Itoa(int(input.Permissions)),
	}
}

// GetAllApiKeys returns a map with all API keys
func (p DatabaseProvider) GetAllApiKeys() map[string]models.ApiKey {
	var result map[string]models.ApiKey
	maps := getAllHashesWithPrefix(prefixApiKeys)
	for _, m := range maps {
		apiKey, err := dbToApiKey(m.Key, m.Hash)
		helper.Check(err)
		result[m.Key] = apiKey
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
	setHashMap(prefixApiKeys+apikey.Id, apiKeyToDb(apikey))
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func (p DatabaseProvider) UpdateTimeApiKey(apikey models.ApiKey) {
	setHashMap(prefixApiKeys+apikey.Id, apiKeyToDb(apikey))
}

// DeleteApiKey deletes an API key with the given ID
func (p DatabaseProvider) DeleteApiKey(id string) {
	deleteKey(prefixApiKeys + id)
}
