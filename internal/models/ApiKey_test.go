package models

import (
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
)

func TestApiKey_GetReadableDate(t *testing.T) {
	key := &ApiKey{}
	test.IsEqualString(t, key.GetReadableDate(), "Never")
	key.LastUsed = 1736276120
	lastTz := os.Getenv("TZ")
	err := os.Setenv("TZ", "Europe/Berlin")
	test.IsNil(t, err)
	test.IsEqualString(t, key.GetReadableDate(), "2025-01-07 19:55:20")
	err = os.Setenv("TZ", lastTz)
	test.IsNil(t, err)
}

func TestApiKey_GetRedactedId(t *testing.T) {
	key := &ApiKey{Id: "eivahB9imahj3fiquoh6DieNgeeThe"}
	test.IsEqualString(t, key.GetRedactedId(), "ei**************************he")
}

func TestSetPermission(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermView)
	if !key.HasPermission(ApiPermView) {
		t.Errorf("expected permission %d to be set", ApiPermView)
	}
	if key.HasPermission(ApiPermEdit) {
		t.Errorf("expected permission %d to be not set", ApiPermEdit)
	}
}

func TestRemovePermission(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermView)
	if !key.HasPermission(ApiPermView) {
		t.Errorf("expected permission %d to be set", ApiPermView)
	}
	key.RemovePermission(ApiPermView)
	if key.HasPermission(ApiPermView) {
		t.Errorf("expected permission %d to be removed", ApiPermView)
	}
}

func TestHasPermission(t *testing.T) {
	key := &ApiKey{}
	if !key.HasPermission(ApiPermNone) {
		t.Errorf("expected ApiPermNone to always return true")
	}
	if key.HasPermission(ApiPermUpload) {
		t.Errorf("expected permission %d not to be set", ApiPermUpload)
	}
	key.GrantPermission(ApiPermUpload)
	if !key.HasPermission(ApiPermUpload) {
		t.Errorf("expected permission %d to be set", ApiPermUpload)
	}
	if key.HasPermission(ApiPermDelete) {
		t.Errorf("expected permission %d not to be set", ApiPermDelete)
	}
}

func TestHasPermissionView(t *testing.T) {
	key := &ApiKey{}
	if key.HasPermissionView() {
		t.Errorf("expected view permission to be not set")
	}
	key.GrantPermission(ApiPermView)
	if !key.HasPermissionView() {
		t.Errorf("expected view permission to be set")
	}
}

func TestHasPermissionUpload(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermUpload)
	if !key.HasPermissionUpload() {
		t.Errorf("expected upload permission to be set")
	}
}

func TestHasPermissionDelete(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermDelete)
	if !key.HasPermissionDelete() {
		t.Errorf("expected delete permission to be set")
	}
}

func TestHasPermissionApiMod(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermApiMod)
	if !key.HasPermissionApiMod() {
		t.Errorf("expected ApiMod permission to be set")
	}
}

func TestHasPermissionEdit(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermEdit)
	if !key.HasPermissionEdit() {
		t.Errorf("expected edit permission to be set")
	}
}

func TestHasPermissionReplace(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermReplace)
	if !key.HasPermissionReplace() {
		t.Errorf("expected edit permission to be set")
	}
}
func TestHasPermissionManageUsers(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermManageUsers)
	if !key.HasPermissionManageUsers() {
		t.Errorf("expected edit permission to be set")
	}
}

func TestApiPermAllNoApiMod(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermDefault)
	if !key.HasPermission(ApiPermView) || !key.HasPermission(ApiPermUpload) || !key.HasPermission(ApiPermDelete) || !key.HasPermission(ApiPermEdit) {
		t.Errorf("expected all permissions except ApiMod to be set")
	}
	if key.HasPermission(ApiPermApiMod) {
		t.Errorf("expected ApiMod permission not to be set")
	}
}

func TestApiPermAll(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermAll)
	if !key.HasPermission(ApiPermView) ||
		!key.HasPermission(ApiPermUpload) ||
		!key.HasPermission(ApiPermDelete) ||
		!key.HasPermission(ApiPermApiMod) ||
		!key.HasPermission(ApiPermEdit) ||
		!key.HasPermission(ApiPermReplace) ||
		!key.HasPermission(ApiPermManageUsers) {
		t.Errorf("expected all permissions to be set")
	}
}

// Helper function to check only one permission is set
func checkOnlyPermissionSet(t *testing.T, key *ApiKey, perm ApiPermission) {
	allPermissions := []struct {
		perm     ApiPermission
		permName string
	}{
		{ApiPermView, "ApiPermView"},
		{ApiPermUpload, "ApiPermUpload"},
		{ApiPermDelete, "ApiPermDelete"},
		{ApiPermApiMod, "ApiPermApiMod"},
		{ApiPermEdit, "ApiPermEdit"},
		{ApiPermReplace, "ApiPermReplace"},
		{ApiPermManageUsers, "ApiPermManageUsers"},
	}

	for _, p := range allPermissions {
		if p.perm == perm {
			if !key.HasPermission(p.perm) {
				t.Errorf("expected permission %s to be set", p.permName)
			}
		} else {
			if key.HasPermission(p.perm) {
				t.Errorf("expected permission %s not to be set", p.permName)
			}
		}
	}
}

func TestSetIndividualPermissions(t *testing.T) {
	key := &ApiKey{}

	// Test each individual permission
	permissions := []struct {
		perm     ApiPermission
		permName string
	}{
		{ApiPermView, "ApiPermView"},
		{ApiPermUpload, "ApiPermUpload"},
		{ApiPermDelete, "ApiPermDelete"},
		{ApiPermApiMod, "ApiPermApiMod"},
		{ApiPermEdit, "ApiPermEdit"},
		{ApiPermReplace, "ApiPermReplace"},
		{ApiPermManageUsers, "ApiPermManageUsers"},
	}

	for _, p := range permissions {
		key.Permissions = ApiPermNone // reset permissions
		key.GrantPermission(p.perm)
		checkOnlyPermissionSet(t, key, p.perm)
	}
}

// Helper function to check combined permissions are set
func checkCombinedPermissions(t *testing.T, key *ApiKey, perms []ApiPermission) {
	for _, perm := range perms {
		if !key.HasPermission(perm) {
			t.Errorf("expected permission %d to be set", perm)
		}
	}
}

func TestSetCombinedPermissions(t *testing.T) {
	key := &ApiKey{}
	allPermissions := []ApiPermission{
		ApiPermView,
		ApiPermUpload,
		ApiPermDelete,
		ApiPermApiMod,
		ApiPermEdit,
		ApiPermReplace,
		ApiPermManageUsers,
	}

	// Test setting permissions in combination
	for i := 0; i < len(allPermissions); i++ {
		key.Permissions = ApiPermNone // reset permissions
		for j := 0; j <= i; j++ {
			key.GrantPermission(allPermissions[j])
		}
		checkCombinedPermissions(t, key, allPermissions[:i+1])
	}
}
