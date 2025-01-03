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
	// ApiPermManageUsers is the permission for managing users
	ApiPermManageUsers
)

// ApiPermNone means no permission granted
const ApiPermNone = 0

// ApiPermAll means all permission granted
const ApiPermAll = 127

// ApiPermDefault means all permission granted, except ApiPermApiMod, ApiPermManageUsers and ApiPermReplace
// This is the default for new API keys that are created from the UI
const ApiPermDefault = ApiPermAll - ApiPermApiMod - ApiPermManageUsers - ApiPermReplace

// ApiKey contains data of a single api key
type ApiKey struct {
	Id           string `json:"Id" redis:"Id"`
	PublicId     string `json:"PublicId" redis:"PublicId"`
	FriendlyName string `json:"FriendlyName" redis:"FriendlyName"`
	LastUsed     int64  `json:"LastUsed" redis:"LastUsed"`
	Permissions  uint8  `json:"Permissions" redis:"Permissions"`
	Expiry       int64  `json:"Expiry" redis:"Expiry"` // Does not expire if 0
	IsSystemKey  bool   `json:"IsSystemKey" redis:"IsSystemKey"`
	UserId       int    `json:"UserId" redis:"UserId"`
}

// GetReadableDate returns the date as YYYY-MM-DD HH:MM:SS
func (key *ApiKey) GetReadableDate() string {
	if key.LastUsed == 0 {
		return "Never"
	}
	return time.Unix(key.LastUsed, 0).Format("2006-01-02 15:04:05")
}

// GetRedactedId returns a redacted version of the API key
func (key *ApiKey) GetRedactedId() string {
	return key.Id[0:2] + "**************************" + key.Id[28:]

}

// GrantPermission sets one or more permissions
func (key *ApiKey) GrantPermission(permission uint8) {
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

// HasPermissionManageUsers returns true if ApiPermManageUsers is granted
func (key *ApiKey) HasPermissionManageUsers() bool {
	return key.HasPermission(ApiPermManageUsers)
}

// ApiKeyOutput is the output that is used after a new key is created
type ApiKeyOutput struct {
	Result   string
	Id       string
	PublicId string
}
