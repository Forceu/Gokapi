package sqlite

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
	UserId     int
}

// GetSession returns the session with the given ID or false if not a valid ID
func (p DatabaseProvider) GetSession(id string) (models.Session, bool) {
	var rowResult schemaSessions
	row := p.sqliteDb.QueryRow("SELECT * FROM Sessions WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.RenewAt, &rowResult.ValidUntil, &rowResult.UserId)
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
		UserId:     rowResult.UserId,
	}
	return result, true
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func (p DatabaseProvider) SaveSession(id string, session models.Session) {
	newData := schemaSessions{
		Id:         id,
		RenewAt:    session.RenewAt,
		ValidUntil: session.ValidUntil,
		UserId:     session.UserId,
	}

	_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO Sessions (Id, RenewAt, ValidUntil, UserId) VALUES (?, ?, ?, ?)",
		newData.Id, newData.RenewAt, newData.ValidUntil, newData.UserId)
	helper.Check(err)
}

// DeleteSession deletes a session with the given ID
func (p DatabaseProvider) DeleteSession(id string) {
	_, err := p.sqliteDb.Exec("DELETE FROM Sessions WHERE Id = ?", id)
	helper.Check(err)
}

// DeleteAllSessions logs all users out
func (p DatabaseProvider) DeleteAllSessions() {
	//goland:noinspection SqlWithoutWhere
	_, err := p.sqliteDb.Exec("DELETE FROM Sessions")
	helper.Check(err)
}

// DeleteAllSessionsByUser logs the specific users out
func (p DatabaseProvider) DeleteAllSessionsByUser(userId int) {
	_, err := p.sqliteDb.Exec("DELETE FROM Sessions WHERE UserId = ?", userId)
	helper.Check(err)
}

func (p DatabaseProvider) cleanExpiredSessions() {
	_, err := p.sqliteDb.Exec("DELETE FROM Sessions WHERE Sessions.ValidUntil < ?", time.Now().Unix())
	helper.Check(err)
}
