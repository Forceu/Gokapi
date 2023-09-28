package guest

import (
	"time"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

// DeleteToken deletes the selected guest token
func DeleteToken(id string) bool {
	if !IsValidGuestToken(id, false) {
		return false
	}
	database.DeleteGuestToken(id)
	return true
}

// NewToken generates a new guest token
func NewToken() string {
	newToken := models.GuestToken{
		Id:            helper.GenerateRandomString(30),
		TimesUsed:     0,
		UnlimitedTime: true,
		LastUsed:      0,
	}
	database.SaveGuestToken(newToken, false)
	return newToken.Id
}

// IsValidGuestToken checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func IsValidGuestToken(tokenId string, modifyTime bool) bool {
	if tokenId == "" {
		return false
	}
	savedToken, ok := database.GetGuestToken(tokenId)
	if ok && savedToken.Id != "" {
		if modifyTime {
			savedToken.LastUsed = time.Now().Unix()
			database.SaveGuestToken(savedToken, true)
		}
		return true
	}
	return false
}
