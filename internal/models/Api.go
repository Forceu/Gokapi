package models

import "time"

const (
	// ApiPermView is the permission for viewing metadata of all uploaded files
	ApiPermView = 1 << iota
	// ApiPermUpload is the permission for creating new files
	ApiPermUpload
	// ApiPermDelete is the permission for deleting files
	ApiPermDelete
	// ApiPermApiMod is the permission for adding / removing API key permissions
	ApiPermApiMod
	// ApiPermEdit is the permission for editing parameters of uploaded files
	ApiPermEdit
	// ApiPermReplace is the permission for replacing the content of uploaded files
	ApiPermReplace
)

// ApiPermNone means no permission granted
const ApiPermNone = 0

// ApiPermAll means all permission granted
const ApiPermAll = 63

// ApiPermAllNoApiMod means all permission granted, except ApiPermApiMod
// This is the default for new API keys that are created from the UI
const ApiPermAllNoApiMod = ApiPermAll - ApiPermApiMod - ApiPermReplace

// ApiKey contains data of a single api key
type ApiKey struct {
	Id           string `json:"Id" redis:"Id"`
	FriendlyName string `json:"FriendlyName" redis:"FriendlyName"`
	LastUsed     int64  `json:"LastUsed" redis:"LastUsed"`
	Permissions  uint8  `json:"Permissions" redis:"Permissions"`
	Expiry       int64  `json:"Expiry" redis:"Expiry"` // Does not expire if 0
	IsSystemKey  bool   `json:"IsSystemKey" redis:"IsSystemKey"`
}

// GetReadableDate returns the date as YYYY-MM-DD HH:MM:SS
func (key *ApiKey) GetReadableDate() string {
	if key.LastUsed == 0 {
		return "Never"
	}
	return time.Unix(key.LastUsed, 0).Format("2006-01-02 15:04:05")
}

// SetPermission grants one or more permissions
func (key *ApiKey) SetPermission(permission uint8) {
	key.Permissions |= permission
}

// RemovePermission revokes one or more permissions
func (key *ApiKey) RemovePermission(permission uint8) {
	key.Permissions &^= permission
}

// HasPermission returns true if the key has the permission(s)
func (key *ApiKey) HasPermission(permission uint8) bool {
	if permission == ApiPermNone {
		return true
	}
	return (key.Permissions & permission) == permission
}

// HasPermissionView returns true if ApiPermView is granted
func (key *ApiKey) HasPermissionView() bool {
	return key.HasPermission(ApiPermView)
}

// HasPermissionUpload returns true if ApiPermUpload is granted
func (key *ApiKey) HasPermissionUpload() bool {
	return key.HasPermission(ApiPermUpload)
}

// HasPermissionDelete returns true if ApiPermDelete is granted
func (key *ApiKey) HasPermissionDelete() bool {
	return key.HasPermission(ApiPermDelete)
}

// HasPermissionApiMod returns true if ApiPermApiMod is granted
func (key *ApiKey) HasPermissionApiMod() bool {
	return key.HasPermission(ApiPermApiMod)
}

// HasPermissionEdit returns true if ApiPermEdit is granted
func (key *ApiKey) HasPermissionEdit() bool {
	return key.HasPermission(ApiPermEdit)
}

// HasPermissionReplace returns true if ApiPermReplace is granted
func (key *ApiKey) HasPermissionReplace() bool {
	return key.HasPermission(ApiPermReplace)
}

// ApiKeyOutput is the output that is used after a new key is created
type ApiKeyOutput struct {
	Result string
	Id     string
}
