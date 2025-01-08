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
