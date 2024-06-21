package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixMetaData = "fmeta:"
)

// GetAllMetadata returns a map of all available files
func (p DatabaseProvider) GetAllMetadata() map[string]models.File {
	result := make(map[string]models.File)
	hashes := getAllHashesWithPrefix(prefixMetaData)
	for _, hash := range hashes {
		file := models.File{}
		err := redigo.ScanStruct(hash.Values, &file)
		helper.Check(err)
		result[file.Id] = file
	}
	return result
}

// GetAllMetaDataIds returns all Ids that contain metadata
func (p DatabaseProvider) GetAllMetaDataIds() []string {
	return getAllKeynamesWithPrefix(prefixMetaData)
}

// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
func (p DatabaseProvider) GetMetaDataById(id string) (models.File, bool) {
	values, ok := getHashMap(prefixMetaData + id)
	if !ok {
		return models.File{}, false
	}
	result := models.File{}
	err := redigo.ScanStruct(values, &result)
	helper.Check(err)
	return result, true
}

// SaveMetaData stores the metadata of a file to the disk
func (p DatabaseProvider) SaveMetaData(file models.File) {
	setHashMapArgs(buildArgs(prefixMetaData + file.Id).AddFlat(file))
}

// DeleteMetaData deletes information about a file
func (p DatabaseProvider) DeleteMetaData(id string) {
	deleteKey(prefixMetaData + id)
}
