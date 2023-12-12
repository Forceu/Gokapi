package models

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
)

// ApiPermNone means no permission granted
const ApiPermNone = 0

// ApiPermAllNoApiMod means all permission granted, except ApiPermApiMod
const ApiPermAllNoApiMod = 23

// ApiPermAll means all permission granted
const ApiPermAll = 31

// ApiKey contains data of a single api key
type ApiKey struct {
	Id             string `json:"Id"`
	FriendlyName   string `json:"FriendlyName"`
	LastUsedString string `json:"LastUsedString"`
	LastUsed       int64  `json:"LastUsed"`
	Permissions    uint8  `json:"Permissions"`
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
