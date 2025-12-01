package users

import (
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
)

const minLengthUser = 2

var ErrorNameToShort = errors.New("username too short")
var ErrorUserExists = errors.New("user already exists")

func Create(name string) (models.User, error) {
	if len(name) < minLengthUser {
		return models.User{}, ErrorNameToShort
	}
	_, ok := database.GetUserByName(name)
	if ok {
		return models.User{}, ErrorUserExists
	}
	newUser := models.User{
		Name:      name,
		UserLevel: models.UserLevelUser,
	}
	if configuration.Get().AllowGuestUploadsByDefault {
		newUser.GrantPermission(models.UserPermGuestUploads)
	}
	database.SaveUser(newUser, true)
	newUser, ok = database.GetUserByName(name)
	if !ok {
		return models.User{}, errors.New("user could not be created")
	}
	return newUser, nil
}
