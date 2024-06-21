package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixUploadStatus = "us:"
)

// GetAllUploadStatus returns all UploadStatus values from the past 24 hours
func (p DatabaseProvider) GetAllUploadStatus() []models.UploadStatus {
	var result []models.UploadStatus
	for k, v := range getAllValuesWithPrefix(prefixUploadStatus) {
		status, err := redigo.Int(v, nil)
		helper.Check(err)
		result = append(result, models.UploadStatus{
			ChunkId:       k,
			CurrentStatus: status,
		})
	}
	return result
}

// GetUploadStatus returns a models.UploadStatus from the ID passed or false if the id is not valid
func (p DatabaseProvider) GetUploadStatus(id string) (models.UploadStatus, bool) {
	status, ok := getKeyInt(prefixUploadStatus + id)
	if !ok {
		return models.UploadStatus{}, false
	}
	result := models.UploadStatus{
		ChunkId:       id,
		CurrentStatus: status,
	}
	return result, true
}

// SaveUploadStatus stores the upload status of a new file for 24 hours
func (p DatabaseProvider) SaveUploadStatus(status models.UploadStatus) {
	setKey(prefixUploadStatus+status.ChunkId, status.CurrentStatus)
	setExpiryInSeconds(prefixUploadStatus+status.ChunkId, 24*60*60) // 24h
}
