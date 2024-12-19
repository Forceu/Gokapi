package models

import "time"

type User struct {
	Id          int    `json:"id" redis:"id""`
	Name        string `json:"name" redis:"Name"`
	Email       string `json:"email" redis:"Email"`
	Permissions uint16 `json:"permissions" redis:"Permissions"`
	UserLevel   uint8  `json:"userLevel" redis:"UserLevel"`
	LastOnline  int64  `json:"lastOnline" redis:"LastOnline"`
	Password    string `redis:"Password"`
}

// GetReadableDate returns the date as YYYY-MM-DD HH:MM:SS
func (u *User) GetReadableDate() string {
	if u.LastOnline == 0 {
		return "Never"
	}
	return time.Unix(u.LastOnline, 0).Format("2006-01-02 15:04:05")
}

const UserLevelSuperAdmin = 0
const UserLevelAdmin = 1
const UserLevelUser = 2

const (
	UserPermissionListOtherUploads = 1 << iota
	UserPermissionEditOtherUploads
	UserPermissionReplaceUploads
	UserPermissionReplaceOtherUploads
	UserPermissionDeleteOtherUploads
	UserPermissionManageUsers
	UserPermissionManageApiKeys
	UserPermissionManageLogs
)
const UserPermissionNone = 0

// SetPermission grants one or more permissions
func (u *User) SetPermission(permission uint16) {
	u.Permissions |= permission
}

// RemovePermission revokes one or more permissions
func (u *User) RemovePermission(permission uint16) {
	u.Permissions &^= permission
}

// HasPermission returns true if the key has the permission(s)
func (u *User) HasPermission(permission uint16) bool {
	if permission == UserPermissionNone {
		return true
	}
	return (u.Permissions & permission) == permission
}
