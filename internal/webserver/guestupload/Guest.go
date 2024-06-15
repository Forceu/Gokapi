package guestupload

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

func NewUploadToken() string {
	newToken := models.UploadToken{
		Id:       helper.GenerateRandomString(30),
		LastUsed: 0,
	}
	database.SaveUploadToken(newToken)
	return newToken.Id
}

func DeleteUploadToken(id string) bool {
	if !IsValidUploadToken(id) {
		return false
	}
	database.DeleteUploadToken(id)
	return true
}

// IsValidUploadToken checks if the  provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func IsValidUploadToken(token string) bool {
	if token == "" {
		return false
	}
	savedToken, ok := database.GetUploadToken(token)
	if ok && savedToken.Id != "" {
		return true
	}
	return false
}
