package models

import (
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

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
func TestHasPermissionManageLogs(t *testing.T) {
	key := &ApiKey{}
	key.GrantPermission(ApiPermManageLogs)
	if !key.HasPermissionManageLogs() {
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
		{ApiPermManageLogs, "ApiPermManageLogs"},
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
		{ApiPermManageLogs, "ApiPermManageLogs"},
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
		ApiPermManageLogs,
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

func TestApiPermissionFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPerm  ApiPermission
		wantError bool
	}{
		{
			name:     "PERM_VIEW",
			input:    "PERM_VIEW",
			wantPerm: ApiPermView,
		},
		{
			name:     "PERM_UPLOAD lowercase",
			input:    "perm_upload",
			wantPerm: ApiPermUpload,
		},
		{
			name:     "PERM_DELETE mixed case",
			input:    "Perm_Delete",
			wantPerm: ApiPermDelete,
		},
		{
			name:     "PERM_API_MOD",
			input:    "PERM_API_MOD",
			wantPerm: ApiPermApiMod,
		},
		{
			name:     "PERM_EDIT",
			input:    "PERM_EDIT",
			wantPerm: ApiPermEdit,
		},
		{
			name:     "PERM_REPLACE",
			input:    "PERM_REPLACE",
			wantPerm: ApiPermReplace,
		},
		{
			name:     "PERM_MANAGE_USERS",
			input:    "PERM_MANAGE_USERS",
			wantPerm: ApiPermManageUsers,
		},
		{
			name:     "PERM_MANAGE_LOGS",
			input:    "PERM_MANAGE_LOGS",
			wantPerm: ApiPermManageLogs,
		},
		{
			name:      "invalid permission",
			input:     "PERM_UNKNOWN",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := ApiPermissionFromString(tt.input)

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotPerm != tt.wantPerm {
				t.Fatalf("expected %v, got %v", tt.wantPerm, gotPerm)
			}
		})
	}
}
