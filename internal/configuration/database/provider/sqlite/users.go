package sqlite

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"time"
)

// GetAllUsers returns a map with all users
func (p DatabaseProvider) GetAllUsers() map[int]models.User {
	result := make(map[int]models.User)
	rows, err := p.sqliteDb.Query("SELECT * FROM Users")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		user := models.User{}
		err = rows.Scan(&user.Id, &user.Email, &user.Password, &user.Name, &user.Permissions, &user.UserLevel, &user.LastOnline)
		helper.Check(err)
		result[user.Id] = user
	}
	return result
}

// GetUser returns a models.User if valid or false if the ID is not valid
func (p DatabaseProvider) GetUser(id int) (models.User, bool) {
	var result models.User
	row := p.sqliteDb.QueryRow("SELECT * FROM Users WHERE Id = ?", id)
	err := row.Scan(&result.Id, &result.Email, &result.Password, &result.Name, &result.Permissions, &result.UserLevel, &result.LastOnline)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, false
		}
		helper.Check(err)
		return models.User{}, false
	}
	return result, true
}

// SaveUser saves a user to the database. If isNewUser is true, a new Id will be generated
func (p DatabaseProvider) SaveUser(user models.User, isNewUser bool) {
	if isNewUser {
		_, err := p.sqliteDb.Exec("INSERT INTO Users ( Name, Email, Password, Permissions, Userlevel) VALUES  (?, ?, ?, ?, ?)",
			user.Name, user.Email, user.Password, user.Permissions, user.UserLevel)
		helper.Check(err)
	} else {
		_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO Users (Id, Name, Email, Password, Permissions, Userlevel) VALUES  (?, ?, ?, ?, ?, ?)",
			user.Id, user.Name, user.Email, user.Password, user.Permissions, user.UserLevel)
		helper.Check(err)
	}
}

// UpdateUserLastOnline writes the last online time to the database
func (p DatabaseProvider) UpdateUserLastOnline(id int) {
	_, err := p.sqliteDb.Exec("UPDATE Users SET LastOnline=? WHERE Id = ?", time.Now().Unix(), id)
	helper.Check(err)
}

// DeleteUser deletes a user with the given ID
func (p DatabaseProvider) DeleteUser(id int) {
	_, err := p.sqliteDb.Exec("DELETE FROM Users WHERE Id = ?", id)
	helper.Check(err)
}
