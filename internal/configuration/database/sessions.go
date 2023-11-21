package database

import (
	"database/sql"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"time"
)

type schemaSessions struct {
	Id         string
	RenewAt    int64
	ValidUntil int64
}

// GetSession returns the session with the given ID or false if not a valid ID
func GetSession(id string) (models.Session, bool) {
	var rowResult schemaSessions
	row := sqliteDb.QueryRow("SELECT * FROM Sessions WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.RenewAt, &rowResult.ValidUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Session{}, false
		}
		helper.Check(err)
		return models.Session{}, false
	}
	result := models.Session{
		RenewAt:    rowResult.RenewAt,
		ValidUntil: rowResult.ValidUntil,
	}
	return result, true
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func SaveSession(id string, session models.Session) {
	newData := schemaSessions{
		Id:         id,
		RenewAt:    session.RenewAt,
		ValidUntil: session.ValidUntil,
	}

	_, err := sqliteDb.Exec("INSERT OR REPLACE INTO Sessions (Id, RenewAt, ValidUntil) VALUES (?, ?, ?)",
		newData.Id, newData.RenewAt, newData.ValidUntil)
	helper.Check(err)
}

// DeleteSession deletes a session with the given ID
func DeleteSession(id string) {
	_, err := sqliteDb.Exec("DELETE FROM Sessions WHERE Id = ?", id)
	helper.Check(err)
}

// DeleteAllSessions logs all users out
func DeleteAllSessions() {
	//goland:noinspection SqlWithoutWhere
	_, err := sqliteDb.Exec("DELETE FROM Sessions")
	helper.Check(err)
}

func cleanExpiredSessions() {
	_, err := sqliteDb.Exec("DELETE FROM Sessions WHERE Sessions.ValidUntil < ?", time.Now().Unix())
	helper.Check(err)
}
