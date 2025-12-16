package models

import (
	"time"
)

const (
	// ApiPermView is the permission for viewing metadata of all uploaded files
	ApiPermView ApiPermission = 1 << iota
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
	// ApiPermManageLogs is the permission required for managing the log file
	ApiPermManageLogs
	// ApiPermManageFileRequests is the permission required for creating and managing file requests
	ApiPermManageFileRequests
)

// ApiPermNone means no permission granted
const ApiPermNone ApiPermission = 0

// ApiPermAll means all permission granted
const ApiPermAll ApiPermission = 511

// ApiPermDefault means all permission granted, except ApiPermApiMod, ApiPermManageUsers, ApiPermManageLogs and ApiPermReplace
// This is the default for new API keys that are created from the UI
const ApiPermDefault = ApiPermAll - ApiPermApiMod - ApiPermManageUsers - ApiPermReplace - ApiPermManageLogs - ApiPermManageFileRequests

// ApiKey contains data of a single api key
type ApiKey struct {
	Id              string        `json:"Id" redis:"Id"`
	PublicId        string        `json:"PublicId" redis:"PublicId"`
	FriendlyName    string        `json:"FriendlyName" redis:"FriendlyName"`
	LastUsed        int64         `json:"LastUsed" redis:"LastUsed"`
	Permissions     ApiPermission `json:"Permissions" redis:"Permissions"`
	Expiry          int64         `json:"Expiry" redis:"Expiry"` // Does not expire if 0
	IsSystemKey     bool          `json:"IsSystemKey" redis:"IsSystemKey"`
	UserId          int           `json:"UserId" redis:"UserId"`
	UploadRequestId int           `json:"UploadRequestId" redis:"UploadRequestId"`
}

// ApiPermission contains zero or more permissions as an uint16 format
type ApiPermission uint16

// GetReadableDate returns the date as YYYY-MM-DD HH:MM:SS
func (key *ApiKey) GetReadableDate() string {
	if key.LastUsed == 0 {
		return "Never"
	}
	return time.Unix(key.LastUsed, 0).Format("2006-01-02 15:04:05")
}

// GetRedactedId returns a redacted version of the API key
func (key *ApiKey) GetRedactedId() string {
	return key.Id[0:2] + "**************************" + key.Id[len(key.Id)-2:]
}

// GrantPermission sets one or more permissions
func (key *ApiKey) GrantPermission(permission ApiPermission) {
	key.Permissions |= permission
}

// RemovePermission revokes one or more permissions
func (key *ApiKey) RemovePermission(permission ApiPermission) {
	key.Permissions &^= permission
}

// HasPermission returns true if the key has the permission(s)
func (key *ApiKey) HasPermission(permission ApiPermission) bool {
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

// HasPermissionManageLogs returns true if ApiPermManageLogs is granted
func (key *ApiKey) HasPermissionManageLogs() bool {
	return key.HasPermission(ApiPermManageLogs)
}

// HasPermissionManageFileRequests returns true if ApiPermManageFileRequests is granted
func (key *ApiKey) HasPermissionManageFileRequests() bool {
	return key.HasPermission(ApiPermManageFileRequests)
}

// ApiKeyOutput is the output used after a new key is created
type ApiKeyOutput struct {
	Result   string
	Id       string
	PublicId string
}
