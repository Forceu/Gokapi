package models

import (
	"encoding/json"

	"github.com/forceu/gokapi/internal/helper"
)

// UserPermission contains zero or more permissions as uint16
type UserPermission uint16

// User contains information about the Gokapi user
type User struct {
	Id            int            `json:"id" redis:"id"`
	Name          string         `json:"name" redis:"Name"`
	Permissions   UserPermission `json:"permissions" redis:"Permissions"`
	UserLevel     UserRank       `json:"userLevel" redis:"UserLevel"`
	LastOnline    int64          `json:"lastOnline" redis:"LastOnline"`
	Password      string         `json:"-" redis:"Password"`
	ResetPassword bool           `json:"resetPassword" redis:"ResetPassword"`
}

// GetReadableUserLevel returns the userlevel as a group name
func (u *User) GetReadableUserLevel() string {
	switch u.UserLevel {
	case UserLevelSuperAdmin:
		return "Super Admin"
	case UserLevelAdmin:
		return "Admin"
	case UserLevelUser:
		return "User"
	default:
		return "Invalid"
	}
}

// ToJson returns the user as a JSON object
func (u *User) ToJson() string {
	result, err := json.Marshal(u)
	helper.Check(err)
	return string(result)
}

// UserLevelSuperAdmin indicates that this is the single user with the most permissions
const UserLevelSuperAdmin UserRank = 0

// UserLevelAdmin indicates that this user has by default all permissions (unless they affect the super-admin)
const UserLevelAdmin UserRank = 1

// UserLevelUser indicates that this user has only basic permissions by default
const UserLevelUser UserRank = 2

// UserRank indicates the rank assigned to the user
type UserRank uint8

// IsSuperAdmin returns true if the user has the Rank UserLevelSuperAdmin
func (u *User) IsSuperAdmin() bool {
	return u.UserLevel == UserLevelSuperAdmin
}

// IsSameUser returns true, if the user has the same ID
func (u *User) IsSameUser(userId int) bool {
	return u.Id == userId
}

const (
	// UserPermReplaceUploads allows replacing uploads
	UserPermReplaceUploads UserPermission = 1 << iota
	// UserPermListOtherUploads allows also listing uploads by other users
	UserPermListOtherUploads
	// UserPermEditOtherUploads allows editing of uploads by other users
	UserPermEditOtherUploads
	// UserPermReplaceOtherUploads allows replacing of uploads by other users
	UserPermReplaceOtherUploads
	// UserPermDeleteOtherUploads allows deleting uploads by other users
	UserPermDeleteOtherUploads
	// UserPermManageLogs allows viewing and deleting logs
	UserPermManageLogs
	// UserPermManageApiKeys allows editing and deleting of API keys by other users
	UserPermManageApiKeys
	// UserPermManageUsers allows creating and editing of users, including granting and revoking permissions
	UserPermManageUsers
	// UserPermGuestUploads allows creating file requests
	UserPermGuestUploads
)

// UserPermissionNone means that the user has no permissions
const UserPermissionNone UserPermission = 0

// UserPermissionAll means that the user has all permissions
const UserPermissionAll UserPermission = 511

// GrantPermission grants one or more permissions
func (u *User) GrantPermission(permission UserPermission) {
	u.Permissions |= permission
}

// RemovePermission revokes one or more permissions
func (u *User) RemovePermission(permission UserPermission) {
	u.Permissions &^= permission
}

// HasPermission returns true if the key has the permission(s)
func (u *User) HasPermission(permission UserPermission) bool {
	if permission == UserPermissionNone {
		return true
	}
	return (u.Permissions & permission) == permission
}

// HasPermissionReplace returns true if the user has the permission UserPermReplaceUploads
func (u *User) HasPermissionReplace() bool {
	return u.HasPermission(UserPermReplaceUploads)
}

// HasPermissionListOtherUploads returns true if the user has the permission UserPermListOtherUploads
func (u *User) HasPermissionListOtherUploads() bool {
	return u.HasPermission(UserPermListOtherUploads)
}

// HasPermissionEditOtherUploads returns true if the user has the permission UserPermEditOtherUploads
func (u *User) HasPermissionEditOtherUploads() bool {
	return u.HasPermission(UserPermEditOtherUploads)
}

// HasPermissionReplaceOtherUploads returns true if the user has the permission UserPermReplaceOtherUploads
func (u *User) HasPermissionReplaceOtherUploads() bool {
	return u.HasPermission(UserPermReplaceOtherUploads)
}

// HasPermissionDeleteOtherUploads returns true if the user has the permission UserPermDeleteOtherUploads
func (u *User) HasPermissionDeleteOtherUploads() bool {
	return u.HasPermission(UserPermDeleteOtherUploads)
}

// HasPermissionManageLogs returns true if the user has the permission UserPermManageLogs
func (u *User) HasPermissionManageLogs() bool {
	return u.HasPermission(UserPermManageLogs)
}

// HasPermissionManageApi returns true if the user has the permission UserPermManageApiKeys
func (u *User) HasPermissionManageApi() bool {
	return u.HasPermission(UserPermManageApiKeys)
}

// HasPermissionManageUsers returns true if the user has the permission UserPermManageUsers
func (u *User) HasPermissionManageUsers() bool {
	return u.HasPermission(UserPermManageUsers)
}

// HasPermissionCreateFileRequests returns true if the user has the permission UserPermGuestUploads
func (u *User) HasPermissionCreateFileRequests() bool {
	return u.HasPermission(UserPermGuestUploads)
}
