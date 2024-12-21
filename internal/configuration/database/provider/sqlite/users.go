package sqlite

import (
	"cmp"
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"slices"
	"time"
)

// GetAllUsers returns a map with all users
func (p DatabaseProvider) GetAllUsers() []models.User {
	var password sql.NullString
	var result []models.User
	rows, err := p.sqliteDb.Query("SELECT * FROM Users ORDER BY Userlevel ASC, LastOnline DESC, Email ASC")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		user := models.User{}
		err = rows.Scan(&user.Id, &user.Email, &password, &user.Name, &user.Permissions, &user.UserLevel, &user.LastOnline)
		helper.Check(err)
		if password.Valid {
			user.Password = password.String
		}
		result = append(result, user)
	}
	return orderUsers(result)
}

func orderUsers(users []models.User) []models.User {
	slices.SortFunc(users, func(a, b models.User) int {
		return cmp.Or(
			cmp.Compare(a.UserLevel, b.UserLevel),
			cmp.Compare(b.LastOnline, a.LastOnline),
			cmp.Compare(a.Email, b.Email),
		)
	})
	return users
}

// GetUser returns a models.User if valid or false if the ID is not valid
func (p DatabaseProvider) GetUser(id int) (models.User, bool) {
	var result models.User
	var password sql.NullString
	row := p.sqliteDb.QueryRow("SELECT * FROM Users WHERE Id = ?", id)
	err := row.Scan(&result.Id, &result.Email, &password, &result.Name, &result.Permissions, &result.UserLevel, &result.LastOnline)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, false
		}
		helper.Check(err)
		return models.User{}, false
	}
	if password.Valid {
		result.Password = password.String
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
	timeNow := time.Now().Unix()
	// To reduce database writes, the entry is only updated, if the last timestamp is more than 30 seconds old
	_, err := p.sqliteDb.Exec("UPDATE Users SET LastOnline= ? WHERE Id = ? AND (? - LastOnline > 30)", timeNow, id, timeNow)
	helper.Check(err)
}

// DeleteUser deletes a user with the given ID
func (p DatabaseProvider) DeleteUser(id int) {
	_, err := p.sqliteDb.Exec("DELETE FROM Users WHERE Id = ?", id)
	helper.Check(err)
}
