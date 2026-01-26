package users

import (
	"errors"
	"testing"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
)

func TestCreate(t *testing.T) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	defer testconfiguration.Delete()

	t.Run("Username too short", func(t *testing.T) {
		_, err := Create("a")
		test.IsEqualBool(t, errors.Is(err, ErrorNameToShort), true)
	})

	t.Run("Successfully create user without default permissions", func(t *testing.T) {
		userName := "testuser1"
		user, err := Create(userName)

		test.IsNil(t, err)
		test.IsEqualString(t, user.Name, userName)
		test.IsEqualInt(t, int(user.UserLevel), int(models.UserLevelUser))

		// Check that guest upload permission was NOT granted
		test.IsEqualBool(t, user.HasPermission(models.UserPermGuestUploads), false)
	})

	t.Run("Duplicate user check", func(t *testing.T) {
		userName := "duplicate"
		_, err := Create(userName)
		test.IsNil(t, err)

		// Try creating the same user again
		_, err = Create(userName)
		test.IsEqualBool(t, errors.Is(err, ErrorUserExists), true)
	})
}
