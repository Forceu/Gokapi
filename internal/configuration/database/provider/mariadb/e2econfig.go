package mariadb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaE2EConfig struct {
	Id     int64
	Config []byte
	UserId int
}

// SaveEnd2EndInfo stores the encrypted e2e info
func (p DatabaseProvider) SaveEnd2EndInfo(info models.E2EInfoEncrypted, userId int) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(info)
	helper.Check(err)

	_, err = p.sqlDb.Exec("INSERT INTO E2EConfig (Config, UserId) VALUES (?, ?) ON DUPLICATE KEY UPDATE Config = ?",
		buf.Bytes(), userId, buf.Bytes())
	helper.Check(err)
}

// GetEnd2EndInfo retrieves the encrypted e2e info
func (p DatabaseProvider) GetEnd2EndInfo(userId int) models.E2EInfoEncrypted {
	result := models.E2EInfoEncrypted{}
	rowResult := schemaE2EConfig{}

	row := p.sqlDb.QueryRow("SELECT Config FROM E2EConfig WHERE UserId = ?", userId)
	err := row.Scan(&rowResult.Config)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result
		}
		helper.Check(err)
		return result
	}

	buf := bytes.NewBuffer(rowResult.Config)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&result)
	helper.Check(err)
	return result
}

// DeleteEnd2EndInfo resets the encrypted e2e info
func (p DatabaseProvider) DeleteEnd2EndInfo(userId int) {
	_, err := p.sqlDb.Exec("DELETE FROM E2EConfig WHERE UserId = ?", userId)
	helper.Check(err)
}
