package sqlite

import (
	"database/sql"
	"errors"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaFileRequests struct {
	Id       int
	Name     string
	UserId   int
	Expiry   int64
	MaxFiles int
	MaxSize  int
	Creation int64
	ApiKey   string
}

// GetFileRequest returns the FileRequest or false if not found
func (p DatabaseProvider) GetFileRequest(id int) (models.FileRequest, bool) {
	var rowResult schemaFileRequests
	row := p.sqliteDb.QueryRow("SELECT * FROM UploadRequests WHERE Id = ?", id)
	err := row.Scan(&rowResult.Id, &rowResult.Name, &rowResult.UserId, &rowResult.Expiry,
		&rowResult.MaxFiles, &rowResult.MaxSize, &rowResult.Creation, &rowResult.ApiKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.FileRequest{}, false
		}
		helper.Check(err)
		return models.FileRequest{}, false
	}
	result := models.FileRequest{
		Id:           rowResult.Id,
		Name:         rowResult.Name,
		UserId:       rowResult.UserId,
		MaxFiles:     rowResult.MaxFiles,
		MaxSize:      rowResult.MaxSize,
		Expiry:       rowResult.Expiry,
		CreationDate: rowResult.Creation,
		ApiKey:       rowResult.ApiKey,
	}
	return result, true
}

// GetAllFileRequests returns an array with all file requests, ordered by creation date
func (p DatabaseProvider) GetAllFileRequests() []models.FileRequest {
	result := make([]models.FileRequest, 0)
	rows, err := p.sqliteDb.Query("SELECT * FROM UploadRequests ORDER BY Creation DESC, Name")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaFileRequests{}
		err = rows.Scan(&rowData.Id, &rowData.Name, &rowData.UserId, &rowData.Expiry, &rowData.MaxFiles,
			&rowData.MaxSize, &rowData.Creation, &rowData.ApiKey)
		helper.Check(err)
		result = append(result, models.FileRequest{
			Id:           rowData.Id,
			Name:         rowData.Name,
			UserId:       rowData.UserId,
			MaxFiles:     rowData.MaxFiles,
			MaxSize:      rowData.MaxSize,
			Expiry:       rowData.Expiry,
			CreationDate: rowData.Creation,
			ApiKey:       rowData.ApiKey,
		})
	}
	return result
}

// SaveFileRequest stores the hotlink associated with the file in the database
// Returns the ID of the new request
func (p DatabaseProvider) SaveFileRequest(request models.FileRequest) int {
	newData := schemaFileRequests{
		Id:       request.Id,
		Name:     request.Name,
		UserId:   request.UserId,
		MaxFiles: request.MaxFiles,
		MaxSize:  request.MaxSize,
		Expiry:   request.Expiry,
		Creation: request.CreationDate,
		ApiKey:   request.ApiKey,
	}

	// If ID is not 0, then an existing file request is being saved and needs to be
	// replaced in the database
	if newData.Id != 0 {
		_, err := p.sqliteDb.Exec("INSERT OR REPLACE INTO UploadRequests (id, name, userid, expiry, maxFiles, maxSize, creation, apiKey) VALUES  (?, ?, ?, ?, ?, ?, ?, ?)",
			newData.Id, newData.Name, newData.UserId, newData.Expiry, newData.MaxFiles, newData.MaxSize, newData.Creation, newData.ApiKey)
		helper.Check(err)
		return newData.Id
	}
	res, err := p.sqliteDb.Exec("INSERT INTO UploadRequests (name, userid, expiry, maxFiles, maxSize, creation, apiKey) VALUES  (?, ?, ?, ?, ?, ?, ?)",
		newData.Name, newData.UserId, newData.Expiry, newData.MaxFiles, newData.MaxSize, newData.Creation, newData.ApiKey)
	helper.Check(err)
	id, err := res.LastInsertId()
	helper.Check(err)
	return int(id)
}

// DeleteFileRequest deletes a file request with the given ID
func (p DatabaseProvider) DeleteFileRequest(request models.FileRequest) {
	if request.Id == 0 {
		return
	}
	_, err := p.sqliteDb.Exec("DELETE FROM UploadRequests WHERE Id = ?", request.Id)
	helper.Check(err)
}
