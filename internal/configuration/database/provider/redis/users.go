package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strconv"
	"time"
)

const (
	prefixUsers         = "users:"
	prefixUserIdCounter = "userid_max"
)

func dbToUser(input []any) (models.User, error) {
	var result models.User
	err := redigo.ScanStruct(input, &result)
	if err != nil {
		return models.User{}, err
	}
	return result, nil
}

// GetAllUsers returns a map with all users
func (p DatabaseProvider) GetAllUsers() []models.User {
	var result []models.User
	maps := p.getAllHashesWithPrefix(prefixUsers)
	for _, v := range maps {
		user, err := dbToUser(v)
		helper.Check(err)
		result = append(result, user)
	}
	return result
}

// GetUser returns a models.User if valid or false if the ID is not valid
func (p DatabaseProvider) GetUser(id int) (models.User, bool) {
	result, ok := p.getHashMap(prefixUsers + strconv.Itoa(id))
	if !ok {
		return models.User{}, false
	}
	user, err := dbToUser(result)
	helper.Check(err)
	return user, true
}

// SaveUser saves a user to the database. If isNewUser is true, a new Id will be generated
func (p DatabaseProvider) SaveUser(user models.User, isNewUser bool) {
	if isNewUser {
		id := p.getIncreasedInt(prefixUserIdCounter)
		user.Id = id
	} else {
		counter, _ := p.getKeyInt(prefixUserIdCounter)
		if counter < user.Id {
			p.setKey(prefixUserIdCounter, user.Id)
		}
	}
	p.setHashMap(p.buildArgs(prefixUsers + strconv.Itoa(user.Id)).AddFlat(user))
}

// UpdateUserLastOnline writes the last online time to the database
func (p DatabaseProvider) UpdateUserLastOnline(id int) {
	p.setHashmapField(prefixUsers+strconv.Itoa(id), "LastOnline", time.Now().Unix())
}

// DeleteUser deletes a user with the given ID
func (p DatabaseProvider) DeleteUser(id int) {
	p.deleteKey(prefixUsers + strconv.Itoa(id))
}
