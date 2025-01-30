package models

import (
	"encoding/json"
	"github.com/forceu/gokapi/internal/helper"
	"time"
)

type UserPermission uint16

type User struct {
	Id            int            `json:"id" redis:"id"`
	Name          string         `json:"name" redis:"Name"`
	Permissions   UserPermission `json:"permissions" redis:"Permissions"`
	UserLevel     UserRank       `json:"userLevel" redis:"UserLevel"`
	LastOnline    int64          `json:"lastOnline" redis:"LastOnline"`
	Password      string         `json:"-" redis:"Password"`
	ResetPassword bool           `json:"resetPassword" redis:"ResetPassword"`
}

// GetReadableDate returns the date as YYYY-MM-DD HH:MM
func (u *User) GetReadableDate() string {
	if u.LastOnline == 0 {
		return "Never"
	}
	if time.Now().Unix()-u.LastOnline < 120 {
		return "Online"
	}
	return time.Unix(u.LastOnline, 0).Format("2006-01-02 15:04")
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

// ToJson returns the user as a JSon object
func (u *User) ToJson() string {
	result, err := json.Marshal(u)
	helper.Check(err)
	return string(result)
}

const UserLevelSuperAdmin UserRank = 0
const UserLevelAdmin UserRank = 1
const UserLevelUser UserRank = 2

type UserRank uint8

func (u *User) IsSuperAdmin() bool {
	return u.UserLevel == UserLevelSuperAdmin
}
func (u *User) IsSameUser(userId int) bool {
	return u.Id == userId
}

const (
	UserPermReplaceUploads UserPermission = 1 << iota
	UserPermListOtherUploads
	UserPermEditOtherUploads
	UserPermReplaceOtherUploads
	UserPermDeleteOtherUploads
	UserPermManageLogs
	UserPermManageApiKeys
	UserPermManageUsers
)
const UserPermissionNone UserPermission = 0
const UserPermissionAll UserPermission = 255

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
