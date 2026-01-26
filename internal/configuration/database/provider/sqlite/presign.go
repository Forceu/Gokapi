package sqlite

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaPresign struct {
	Id       string
	FileId   string
	Expiry   int64
	Filename string
}

// GetPresignedUrl returns the presigned url with the given ID or false if not a valid ID
func (p DatabaseProvider) GetPresignedUrl(id string) (models.Presign, bool) {
	var rowResult schemaPresign
	row := p.sqliteDb.QueryRow("SELECT * FROM Presign WHERE id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.FileId, &rowResult.Expiry, &rowResult.Filename)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Presign{}, false
		}
		helper.Check(err)
		return models.Presign{}, false
	}
	result := models.Presign{
		Id:       rowResult.Id,
		FileIds:  strings.Split(rowResult.FileId, ","),
		Expiry:   rowResult.Expiry,
		Filename: rowResult.Filename,
	}
	return result, true
}

// SavePresignedUrl saves the presigned url
func (p DatabaseProvider) SavePresignedUrl(presign models.Presign) {
	_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO Presign (id, fileIds, expiry, filename) VALUES (?, ?, ?, ?)",
		presign.Id, strings.Join(presign.FileIds, ","), presign.Expiry, presign.Filename)
	helper.Check(err)
}

// DeletePresignedUrl deletes the presigned url with the given ID
func (p DatabaseProvider) DeletePresignedUrl(id string) {
	_, err := p.sqliteDb.Exec("DELETE FROM Presign WHERE id = ?", id)
	helper.Check(err)
}

func (p DatabaseProvider) cleanPresignedUrls() {
	_, err := p.sqliteDb.Exec("DELETE FROM Presign WHERE expiry < ?", currentTime().Unix())
	helper.Check(err)
}
