package models

import (
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
	"time"
)

func TestUser_GetReadableDate(t *testing.T) {
	u := User{
		Id:            50,
		Name:          "Admin",
		Permissions:   UserPermissionAll,
		UserLevel:     UserLevelSuperAdmin,
		LastOnline:    0,
		Password:      "1234",
		ResetPassword: false,
	}
	date := u.GetReadableDate()
	test.IsEqualString(t, date, "Never")
	u.LastOnline = time.Now().Unix() - 10
	date = u.GetReadableDate()
	test.IsEqualString(t, date, "Online")
	u.LastOnline = 1736276120

	lastTz := os.Getenv("TZ")
	err := os.Setenv("TZ", "Europe/Berlin")
	test.IsNil(t, err)
	date = u.GetReadableDate()
	test.IsEqualString(t, date, "2025-01-07 19:55")
	err = os.Setenv("TZ", lastTz)
	test.IsNil(t, err)
}

func TestUserPermAll(t *testing.T) {
	user := &User{}
	user.GrantPermission(UserPermissionAll)
	if !user.HasPermission(UserPermReplaceUploads) ||
		!user.HasPermission(UserPermListOtherUploads) ||
		!user.HasPermission(UserPermEditOtherUploads) ||
		!user.HasPermission(UserPermDeleteOtherUploads) ||
		!user.HasPermission(UserPermReplaceOtherUploads) ||
		!user.HasPermission(UserPermManageLogs) ||
		!user.HasPermission(UserPermManageApiKeys) ||
		!user.HasPermission(UserPermManageUsers) {
		t.Errorf("expected all permissions to be set")
	}
}

// Helper function to check only one permission is set
func checkOnlyUserPermissionSet(t *testing.T, user *User, perm UserPermission) {
	allPermissions := []struct {
		perm     UserPermission
		permName string
	}{
		{UserPermReplaceUploads, "UserPermReplaceUploads"},
		{UserPermListOtherUploads, "UserPermListOtherUploads"},
		{UserPermEditOtherUploads, "UserPermEditOtherUploads"},
		{UserPermDeleteOtherUploads, "UserPermDeleteOtherUploads"},
		{UserPermReplaceOtherUploads, "UserPermReplaceOtherUploads"},
		{UserPermManageLogs, "UserPermManageLogs"},
		{UserPermManageApiKeys, "UserPermManageApiKeys"},
		{UserPermManageUsers, "UserPermManageUsers"},
	}

	for _, p := range allPermissions {
		if p.perm == perm {
			if !user.HasPermission(p.perm) {
				t.Errorf("expected permission %s to be set", p.permName)
			}
		} else {
			if user.HasPermission(p.perm) {
				t.Errorf("expected permission %s not to be set", p.permName)
			}
		}
	}
}

func TestSetIndividualUserPermissions(t *testing.T) {
	user := &User{}

	// Test each individual permission
	permissions := []struct {
		perm     UserPermission
		permName string
	}{
		{UserPermReplaceUploads, "UserPermReplaceUploads"},
		{UserPermListOtherUploads, "UserPermListOtherUploads"},
		{UserPermEditOtherUploads, "UserPermEditOtherUploads"},
		{UserPermDeleteOtherUploads, "UserPermDeleteOtherUploads"},
		{UserPermReplaceOtherUploads, "UserPermReplaceOtherUploads"},
		{UserPermManageLogs, "UserPermManageLogs"},
		{UserPermManageApiKeys, "UserPermManageApiKeys"},
		{UserPermManageUsers, "UserPermManageUsers"},
	}

	for _, p := range permissions {
		user.Permissions = UserPermissionNone // reset permissions
		user.GrantPermission(p.perm)
		checkOnlyUserPermissionSet(t, user, p.perm)
	}
}

// Helper function to check combined permissions are set
func checkCombinedUserPermissions(t *testing.T, user *User, perms []UserPermission) {
	for _, perm := range perms {
		if !user.HasPermission(perm) {
			t.Errorf("expected permission %d to be set", perm)
		}
	}
}

func TestSetCombinedUserPermissions(t *testing.T) {
	user := &User{}
	allPermissions := []UserPermission{
		UserPermReplaceUploads,
		UserPermListOtherUploads,
		UserPermEditOtherUploads,
		UserPermDeleteOtherUploads,
		UserPermReplaceOtherUploads,
		UserPermManageLogs,
		UserPermManageApiKeys,
		UserPermManageUsers,
	}

	// Test setting permissions in combination
	for i := 0; i < len(allPermissions); i++ {
		user.Permissions = UserPermissionNone // reset permissions
		for j := 0; j <= i; j++ {
			user.GrantPermission(allPermissions[j])
		}
		checkCombinedUserPermissions(t, user, allPermissions[:i+1])
	}
}

func TestUser_GetReadableUserLevel(t *testing.T) {
	user := &User{
		UserLevel: UserLevelSuperAdmin,
	}
	test.IsEqualString(t, user.GetReadableUserLevel(), "Super Admin")
	user.UserLevel = UserLevelAdmin
	test.IsEqualString(t, user.GetReadableUserLevel(), "Admin")
	user.UserLevel = UserLevelUser
	test.IsEqualString(t, user.GetReadableUserLevel(), "User")
	user.UserLevel = 4
	test.IsEqualString(t, user.GetReadableUserLevel(), "Invalid")
}

func TestUser_IsSuperAdmin(t *testing.T) {
	user := &User{
		UserLevel: UserLevelSuperAdmin,
	}
	test.IsEqualBool(t, user.IsSuperAdmin(), true)
	user.UserLevel = UserLevelAdmin
	test.IsEqualBool(t, user.IsSuperAdmin(), false)
	user.UserLevel = UserLevelUser
	test.IsEqualBool(t, user.IsSuperAdmin(), false)
	user.UserLevel = 4
	test.IsEqualBool(t, user.IsSuperAdmin(), false)
}

func TestUser_IsSameUser(t *testing.T) {
	user := &User{
		Id: 5,
	}
	test.IsEqualBool(t, user.IsSameUser(5), true)
	test.IsEqualBool(t, user.IsSameUser(0), false)
}

func TestSetUserPermission(t *testing.T) {
	user := &User{}
	user.GrantPermission(UserPermListOtherUploads)
	if !user.HasPermission(UserPermListOtherUploads) {
		t.Errorf("expected permission %d to be set", UserPermListOtherUploads)
	}
	if user.HasPermission(UserPermReplaceOtherUploads) {
		t.Errorf("expected permission %d to be not set", UserPermReplaceOtherUploads)
	}
}

func TestRemoveUserPermission(t *testing.T) {
	user := &User{}
	user.GrantPermission(UserPermManageUsers)
	if !user.HasPermission(UserPermManageUsers) {
		t.Errorf("expected permission %d to be set", UserPermManageUsers)
	}
	user.RemovePermission(UserPermManageUsers)
	if user.HasPermission(UserPermManageUsers) {
		t.Errorf("expected permission %d to be removed", UserPermManageUsers)
	}
}

func TestHasUserPermission(t *testing.T) {
	user := &User{}
	if !user.HasPermission(UserPermissionNone) {
		t.Errorf("expected ApiPermNone to always return true")
	}
	if user.HasPermission(UserPermManageUsers) {
		t.Errorf("expected permission %d not to be set", UserPermManageUsers)
	}
	user.GrantPermission(UserPermManageUsers)
	if !user.HasPermission(UserPermManageUsers) {
		t.Errorf("expected permission %d to be set", UserPermManageUsers)
	}
	if user.HasPermission(UserPermReplaceOtherUploads) {
		t.Errorf("expected permission %d not to be set", UserPermReplaceOtherUploads)
	}
}

func TestUser_HasPermissionReplace(t *testing.T) {
	user := &User{}
	if user.HasPermissionReplace() {
		t.Errorf("expected replace permission to be not set")
	}
	user.GrantPermission(UserPermReplaceUploads)
	if !user.HasPermissionReplace() {
		t.Errorf("expected replace permission to be set")
	}
}

func TestUser_HasPermissionListOtherUploads(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionListOtherUploads(), false)
	user.GrantPermission(UserPermListOtherUploads)
	test.IsEqualBool(t, user.HasPermissionListOtherUploads(), true)
}

func TestUser_HasPermissionEditOtherUploads(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionEditOtherUploads(), false)
	user.GrantPermission(UserPermEditOtherUploads)
	test.IsEqualBool(t, user.HasPermissionEditOtherUploads(), true)
}

func TestUser_HasPermissionDeleteOtherUploads(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionDeleteOtherUploads(), false)
	user.GrantPermission(UserPermDeleteOtherUploads)
	test.IsEqualBool(t, user.HasPermissionDeleteOtherUploads(), true)
}

func TestUser_HasPermissionReplaceOtherUploads(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionReplaceOtherUploads(), false)
	user.GrantPermission(UserPermReplaceOtherUploads)
	test.IsEqualBool(t, user.HasPermissionReplaceOtherUploads(), true)
}
func TestUser_HasPermissionManageLogs(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionManageLogs(), false)
	user.GrantPermission(UserPermManageLogs)
	test.IsEqualBool(t, user.HasPermissionManageLogs(), true)
}

func TestUser_HasPermissionManageApi(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionManageApi(), false)
	user.GrantPermission(UserPermManageApiKeys)
	test.IsEqualBool(t, user.HasPermissionManageApi(), true)
}
func TestUser_HasPermissionManageUsers(t *testing.T) {
	user := &User{}
	test.IsEqualBool(t, user.HasPermissionManageUsers(), false)
	user.GrantPermission(UserPermManageUsers)
	test.IsEqualBool(t, user.HasPermissionManageUsers(), true)
}

func TestUser_ToJson(t *testing.T) {
	user := &User{
		Id:            4,
		Name:          "Test User",
		Permissions:   UserPermissionAll,
		UserLevel:     UserLevelAdmin,
		LastOnline:    1337,
		Password:      "1234",
		ResetPassword: true,
	}
	test.IsEqualString(t, user.ToJson(), `{"id":4,"name":"Test User","permissions":255,"userLevel":1,"lastOnline":1337,"resetPassword":true}`)
}
