package models

import (
	"errors"
	"strings"
)

const (
	// ApiPermView is the permission for viewing metadata of all uploaded files PERM_VIEW
	ApiPermView ApiPermission = 1 << iota
	// ApiPermUpload is the permission for creating new files PERM_UPLOAD
	ApiPermUpload
	// ApiPermDelete is the permission for deleting files PERM_DELETE
	ApiPermDelete
	// ApiPermApiMod is the permission for adding / removing API key permissions PERM_API_MOD
	ApiPermApiMod
	// ApiPermEdit is the permission for editing parameters of uploaded files PERM_EDIT
	ApiPermEdit
	// ApiPermReplace is the permission for replacing the content of uploaded files PERM_REPLACE
	ApiPermReplace
	// ApiPermManageUsers is the permission for managing users PERM_MANAGE_USERS
	ApiPermManageUsers
	// ApiPermManageLogs is the permission required for managing the log file PERM_MANAGE_LOGS
	ApiPermManageLogs
	// ApiPermManageFileRequests is the permission required for creating and managing file requests PERM_MANAGE_FILE_REQUESTS
	ApiPermManageFileRequests
	// ApiPermDownload is the permission required for downloading stored files without increasing the counter PERM_DOWNLOAD
	ApiPermDownload
)

// ApiPermNone means no permission granted
const ApiPermNone ApiPermission = 0

// ApiPermDefault the default for new API keys that are created from the UI
const ApiPermDefault = ApiPermView + ApiPermUpload + ApiPermDelete + ApiPermEdit

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

func ApiPermissionFromString(permString string) (ApiPermission, error) {
	switch strings.ToUpper(permString) {
	case "PERM_VIEW":
		return ApiPermView, nil
	case "PERM_UPLOAD":
		return ApiPermUpload, nil
	case "PERM_DELETE":
		return ApiPermDelete, nil
	case "PERM_API_MOD":
		return ApiPermApiMod, nil
	case "PERM_EDIT":
		return ApiPermEdit, nil
	case "PERM_REPLACE":
		return ApiPermReplace, nil
	case "PERM_MANAGE_USERS":
		return ApiPermManageUsers, nil
	case "PERM_MANAGE_LOGS":
		return ApiPermManageLogs, nil
	case "PERM_MANAGE_FILE_REQUESTS":
		return ApiPermManageFileRequests, nil
	case "PERM_DOWNLOAD":
		return ApiPermDownload, nil
	default:
		return 0, errors.New("invalid permission")
	}
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

// HasPermissionDownload returns true if ApiPermDownload is granted
func (key *ApiKey) HasPermissionDownload() bool {
	return key.HasPermission(ApiPermDownload)
}

// ApiKeyOutput is the output used after a new key is created
type ApiKeyOutput struct {
	Result   string
	Id       string
	PublicId string
}
