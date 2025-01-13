package sqlite

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

type schemaMetaData struct {
	Id                 string
	Name               string
	Size               string
	SHA1               string
	ExpireAt           int64
	SizeBytes          int64
	ExpireAtString     string
	DownloadsRemaining int
	DownloadCount      int
	PasswordHash       string
	HotlinkId          string
	ContentType        string
	AwsBucket          string
	Encryption         []byte
	UnlimitedDownloads int
	UnlimitedTime      int
	UserId             int
}

func (rowData schemaMetaData) ToFileModel() (models.File, error) {
	result := models.File{
		Id:                 rowData.Id,
		Name:               rowData.Name,
		Size:               rowData.Size,
		SHA1:               rowData.SHA1,
		ExpireAt:           rowData.ExpireAt,
		SizeBytes:          rowData.SizeBytes,
		ExpireAtString:     rowData.ExpireAtString,
		DownloadsRemaining: rowData.DownloadsRemaining,
		DownloadCount:      rowData.DownloadCount,
		PasswordHash:       rowData.PasswordHash,
		HotlinkId:          rowData.HotlinkId,
		ContentType:        rowData.ContentType,
		AwsBucket:          rowData.AwsBucket,
		Encryption:         models.EncryptionInfo{},
		UnlimitedDownloads: rowData.UnlimitedDownloads == 1,
		UnlimitedTime:      rowData.UnlimitedTime == 1,
		UserId:             rowData.UserId,
	}

	buf := bytes.NewBuffer(rowData.Encryption)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&result.Encryption)
	return result, err
}

// GetAllMetadata returns a map of all available files
func (p DatabaseProvider) GetAllMetadata() map[string]models.File {
	result := make(map[string]models.File)
	rows, err := p.sqliteDb.Query("SELECT * FROM FileMetaData")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaMetaData{}
		err = rows.Scan(&rowData.Id, &rowData.Name, &rowData.Size, &rowData.SHA1, &rowData.ExpireAt, &rowData.SizeBytes,
			&rowData.ExpireAtString, &rowData.DownloadsRemaining, &rowData.DownloadCount, &rowData.PasswordHash,
			&rowData.HotlinkId, &rowData.ContentType, &rowData.AwsBucket, &rowData.Encryption,
			&rowData.UnlimitedDownloads, &rowData.UnlimitedTime, &rowData.UserId)
		helper.Check(err)
		var metaData models.File
		metaData, err = rowData.ToFileModel()
		helper.Check(err)
		result[metaData.Id] = metaData
	}
	return result
}

// GetAllMetaDataIds returns all Ids that contain metadata
func (p DatabaseProvider) GetAllMetaDataIds() []string {
	keys := make([]string, 0)
	rows, err := p.sqliteDb.Query("SELECT Id FROM FileMetaData")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := schemaMetaData{}
		err = rows.Scan(&rowData.Id)
		helper.Check(err)
		keys = append(keys, rowData.Id)
	}
	return keys
}

// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
func (p DatabaseProvider) GetMetaDataById(id string) (models.File, bool) {
	result := models.File{}
	rowData := schemaMetaData{}

	row := p.sqliteDb.QueryRow("SELECT * FROM FileMetaData WHERE Id = ?", id)
	err := row.Scan(&rowData.Id, &rowData.Name, &rowData.Size, &rowData.SHA1, &rowData.ExpireAt, &rowData.SizeBytes,
		&rowData.ExpireAtString, &rowData.DownloadsRemaining, &rowData.DownloadCount, &rowData.PasswordHash,
		&rowData.HotlinkId, &rowData.ContentType, &rowData.AwsBucket, &rowData.Encryption,
		&rowData.UnlimitedDownloads, &rowData.UnlimitedTime, &rowData.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, false
		}
		helper.Check(err)
		return result, false
	}
	result, err = rowData.ToFileModel()
	helper.Check(err)
	return result, true
}

// SaveMetaData stores the metadata of a file to the disk
func (p DatabaseProvider) SaveMetaData(file models.File) {
	newData := schemaMetaData{
		Id:                 file.Id,
		Name:               file.Name,
		Size:               file.Size,
		SHA1:               file.SHA1,
		ExpireAt:           file.ExpireAt,
		SizeBytes:          file.SizeBytes,
		ExpireAtString:     file.ExpireAtString,
		DownloadsRemaining: file.DownloadsRemaining,
		DownloadCount:      file.DownloadCount,
		PasswordHash:       file.PasswordHash,
		HotlinkId:          file.HotlinkId,
		ContentType:        file.ContentType,
		AwsBucket:          file.AwsBucket,
		UserId:             file.UserId,
	}

	if file.UnlimitedDownloads {
		newData.UnlimitedDownloads = 1
	}
	if file.UnlimitedTime {
		newData.UnlimitedTime = 1
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(file.Encryption)
	helper.Check(err)
	newData.Encryption = buf.Bytes()

	_, err = p.sqliteDb.Exec(`INSERT OR REPLACE INTO FileMetaData (Id, Name, Size, SHA1, ExpireAt, SizeBytes, ExpireAtString, 
                                   DownloadsRemaining, DownloadCount, PasswordHash, HotlinkId, ContentType, AwsBucket, Encryption,
                                   UnlimitedDownloads, UnlimitedTime, UserId) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newData.Id, newData.Name, newData.Size, newData.SHA1, newData.ExpireAt, newData.SizeBytes, newData.ExpireAtString,
		newData.DownloadsRemaining, newData.DownloadCount, newData.PasswordHash, newData.HotlinkId, newData.ContentType,
		newData.AwsBucket, newData.Encryption, newData.UnlimitedDownloads, newData.UnlimitedTime, newData.UserId)
	helper.Check(err)
}

// IncreaseDownloadCount increases the download count of a file, preventing race conditions
func (p DatabaseProvider) IncreaseDownloadCount(id string, decreaseRemainingDownloads bool) {
	if decreaseRemainingDownloads {
		_, err := p.sqliteDb.Exec(`UPDATE FileMetaData SET DownloadCount = DownloadCount + 1,
                        DownloadsRemaining = DownloadsRemaining - 1 WHERE id = ?`, id)
		helper.Check(err)
	} else {
		_, err := p.sqliteDb.Exec(`UPDATE FileMetaData SET DownloadCount = DownloadCount + 1 WHERE id = ?`, id)
		helper.Check(err)
	}
}

// DeleteMetaData deletes information about a file
func (p DatabaseProvider) DeleteMetaData(id string) {
	_, err := p.sqliteDb.Exec("DELETE FROM FileMetaData WHERE Id = ?", id)
	helper.Check(err)
}
