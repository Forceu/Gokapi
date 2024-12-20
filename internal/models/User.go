package models

import "time"

type User struct {
	Id          int    `json:"id" redis:"id"`
	Name        string `json:"name" redis:"Name"`
	Email       string `json:"email" redis:"Email"`
	Permissions uint16 `json:"permissions" redis:"Permissions"`
	UserLevel   uint8  `json:"userLevel" redis:"UserLevel"`
	LastOnline  int64  `json:"lastOnline" redis:"LastOnline"`
	Password    string `redis:"Password"`
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

const UserLevelSuperAdmin = 0
const UserLevelAdmin = 1
const UserLevelUser = 2

const (
	UserPermissionReplaceUploads = 1 << iota
	UserPermissionListOtherUploads
	UserPermissionEditOtherUploads
	UserPermissionReplaceOtherUploads
	UserPermissionDeleteOtherUploads
	UserPermissionManageLogs
	UserPermissionManageApiKeys
	UserPermissionManageUsers
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

// HasPermissionReplace returns true if the user has the permission UserPermissionReplaceUploads
func (u *User) HasPermissionReplace() bool {
	return u.HasPermission(UserPermissionReplaceUploads)
}

// HasPermissionListOtherUploads returns true if the user has the permission UserPermissionListOtherUploads
func (u *User) HasPermissionListOtherUploads() bool {
	return u.HasPermission(UserPermissionListOtherUploads)
}

// HasPermissionEditOtherUploads returns true if the user has the permission UserPermissionEditOtherUploads
func (u *User) HasPermissionEditOtherUploads() bool {
	return u.HasPermission(UserPermissionEditOtherUploads)
}

// HasPermissionReplaceOtherUploads returns true if the user has the permission UserPermissionReplaceOtherUploads
func (u *User) HasPermissionReplaceOtherUploads() bool {
	return u.HasPermission(UserPermissionReplaceOtherUploads)
}

// HasPermissionDeleteOtherUploads returns true if the user has the permission UserPermissionDeleteOtherUploads
func (u *User) HasPermissionDeleteOtherUploads() bool {
	return u.HasPermission(UserPermissionDeleteOtherUploads)
}

// HasPermissionManageLogs returns true if the user has the permission UserPermissionManageLogs
func (u *User) HasPermissionManageLogs() bool {
	return u.HasPermission(UserPermissionManageLogs)
}

// HasPermissionManageApi returns true if the user has the permission UserPermissionManageApiKeys
func (u *User) HasPermissionManageApi() bool {
	return u.HasPermission(UserPermissionManageApiKeys)
}

// HasPermissionManageUsers returns true if the user has the permission UserPermissionManageUsers
func (u *User) HasPermissionManageUsers() bool {
	return u.HasPermission(UserPermissionManageUsers)
}
