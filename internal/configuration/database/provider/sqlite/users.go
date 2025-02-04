package sqlite

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"time"
)

type schemaUser struct {
	Id            int
	Name          string
	Password      sql.NullString
	Permissions   models.UserPermission
	UserLevel     models.UserRank
	LastOnline    int64
	ResetPassword int
}

func (s schemaUser) ToUser() models.User {
	pw := ""
	if s.Password.Valid {
		pw = s.Password.String
	}
	return models.User{
		Id:            s.Id,
		Name:          s.Name,
		Permissions:   s.Permissions,
		UserLevel:     s.UserLevel,
		LastOnline:    s.LastOnline,
		Password:      pw,
		ResetPassword: s.ResetPassword == 1,
	}
}

// GetAllUsers returns a map with all users
func (p DatabaseProvider) GetAllUsers() []models.User {
	var result []models.User
	rows, err := p.sqliteDb.Query("SELECT * FROM Users ORDER BY Userlevel ASC, LastOnline DESC, Name ASC")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		row := schemaUser{}
		err = rows.Scan(&row.Id, &row.Name, &row.Password, &row.Permissions, &row.UserLevel, &row.LastOnline, &row.ResetPassword)
		helper.Check(err)
		result = append(result, row.ToUser())
	}
	return result
}

func (p DatabaseProvider) getUserWithConstraint(isName bool, searchValue any) (models.User, bool) {
	rowResult := schemaUser{}
	query := "SELECT * FROM Users WHERE Id = ?"
	if isName {
		query = "SELECT * FROM Users WHERE Name = ?"
	}
	row := p.sqliteDb.QueryRow(query, searchValue)
	err := row.Scan(&rowResult.Id, &rowResult.Name, &rowResult.Password, &rowResult.Permissions, &rowResult.UserLevel, &rowResult.LastOnline, &rowResult.ResetPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, false
		}
		helper.Check(err)
		return models.User{}, false
	}
	user := rowResult.ToUser()
	return user, true
}

// GetUser returns a models.User if valid or false if the ID is not valid
func (p DatabaseProvider) GetUser(id int) (models.User, bool) {
	return p.getUserWithConstraint(false, id)
}

// GetUserByName returns a models.User if valid or false if the name is not valid
func (p DatabaseProvider) GetUserByName(username string) (models.User, bool) {
	return p.getUserWithConstraint(true, username)
}

// SaveUser saves a user to the database. If isNewUser is true, a new Id will be generated
func (p DatabaseProvider) SaveUser(user models.User, isNewUser bool) {
	resetpw := 0
	if user.ResetPassword {
		resetpw = 1
	}
	if isNewUser {
		_, err := p.sqliteDb.Exec("INSERT INTO Users (Name, Password, Permissions, Userlevel, LastOnline, ResetPassword) VALUES  (?, ?, ?, ?, ?, ?)",
			user.Name, user.Password, user.Permissions, user.UserLevel, user.LastOnline, resetpw)
		helper.Check(err)
	} else {
		_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO Users (Id, Name, Password, Permissions, Userlevel, LastOnline, ResetPassword) VALUES  (?, ?, ?, ?, ?, ?, ?)",
			user.Id, user.Name, user.Password, user.Permissions, user.UserLevel, user.LastOnline, resetpw)
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
