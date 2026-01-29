package tokengeneration

import (
	"errors"
	"time"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/api"
)

func containsApiPermission(requestedPermissions models.ApiPermission, containsPermission models.ApiPermission) bool {
	return containsPermission&requestedPermissions == containsPermission
}

// Generate a temporary API key for the given user
func Generate(user models.User, permission models.ApiPermission) (string, int64, error) {
	if containsApiPermission(permission, models.ApiPermReplace) && !user.HasPermissionReplace() {
		return "", 0, errors.New("user does not have permission to generate a token with PERM_REPLACE")
	}
	if containsApiPermission(permission, models.ApiPermManageUsers) && !user.HasPermissionManageUsers() {
		return "", 0, errors.New("user does not have permission to generate a token with PERM_MANAGE_USERS")
	}
	if containsApiPermission(permission, models.ApiPermManageLogs) && !user.HasPermissionManageLogs() {
		return "", 0, errors.New("user does not have permission to generate a token with PERM_MANAGE_LOGS")
	}

	key := models.ApiKey{
		Id:           helper.GenerateRandomString(api.LengthApiKey),
		PublicId:     helper.GenerateRandomString(api.LengthPublicId),
		FriendlyName: "Temporary Token",
		Permissions:  permission,
		IsSystemKey:  true,
		UserId:       user.Id,
		Expiry:       time.Now().Add(time.Minute * 5).Unix(),
	}
	database.SaveApiKey(key)
	return key.Id, key.Expiry, nil
}
