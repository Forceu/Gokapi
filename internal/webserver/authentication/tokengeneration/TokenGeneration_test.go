package tokengeneration

import (
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
)

func TestGenerate(t *testing.T) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	defer testconfiguration.Delete()

	// Mock user with base permissions
	testUser := models.User{
		Id:   6644,
		Name: "TestUser",
	}

	t.Run("Generate basic token", func(t *testing.T) {
		// Requesting no special high-level permissions
		token, expiry, err := Generate(testUser, models.ApiPermEdit)
		test.IsNil(t, err)
		test.IsEqualBool(t, len(token) > 0, true)

		// Verify expiry is roughly 5 minutes from now
		now := time.Now().Unix()
		test.IsEqualBool(t, expiry > now && expiry <= now+(5*60), true)
	})

	t.Run("Fail on missing PERM_REPLACE", func(t *testing.T) {
		// User does not have replace permission in their model
		_, _, err := Generate(testUser, models.ApiPermReplace)
		test.IsEqualBool(t, err != nil, true)
		test.IsEqualString(t, err.Error(), "user does not have permission to generate a token with PERM_REPLACE")
	})

	t.Run("Fail on missing PERM_MANAGE_USERS", func(t *testing.T) {
		_, _, err := Generate(testUser, models.ApiPermManageUsers)
		test.IsEqualBool(t, err != nil, true)
		test.IsEqualString(t, err.Error(), "user does not have permission to generate a token with PERM_MANAGE_USERS")
	})
	t.Run("Fail on missing PERM_MANAGE_LOGS", func(t *testing.T) {
		_, _, err := Generate(testUser, models.ApiPermManageLogs)
		test.IsEqualBool(t, err != nil, true)
		test.IsEqualString(t, err.Error(), "user does not have permission to generate a token with PERM_MANAGE_LOGS")
	})

	t.Run("Success with elevated permissions", func(t *testing.T) {
		// Grant user the necessary permission
		privilegedUser := testUser
		privilegedUser.GrantPermission(models.UserPermManageUsers)

		token, _, err := Generate(privilegedUser, models.ApiPermManageUsers)
		test.IsNil(t, err)
		test.IsEqualBool(t, len(token) > 0, true)
	})
}

func TestContainsApiPermission(t *testing.T) {
	t.Run("Exact match", func(t *testing.T) {
		res := containsApiPermission(models.ApiPermEdit, models.ApiPermEdit)
		test.IsEqualBool(t, res, true)
	})

	t.Run("Subset match", func(t *testing.T) {
		requested := models.ApiPermEdit | models.ApiPermUpload
		res := containsApiPermission(requested, models.ApiPermEdit)
		test.IsEqualBool(t, res, true)
	})

	t.Run("No match", func(t *testing.T) {
		requested := models.ApiPermEdit
		res := containsApiPermission(requested, models.ApiPermUpload)
		test.IsEqualBool(t, res, false)
	})
}
