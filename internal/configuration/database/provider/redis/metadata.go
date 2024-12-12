package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strings"
)

const (
	prefixMetaData = "fmeta:"
)

// GetAllMetadata returns a map of all available files
func (p DatabaseProvider) GetAllMetadata() map[string]models.File {
	result := make(map[string]models.File)
	maps := p.getAllHashesWithPrefix(prefixMetaData)
	for k, v := range maps {
		file, err := newDbToMetadata(k, v)
		helper.Check(err)
		result[file.Id] = file
	}
	return result
}

func newDbToMetadata(id string, input []any) (models.File, error) {
	var result models.File
	err := redigo.ScanStruct(input, &result)
	if err != nil {
		return models.File{}, err
	}
	result.Id = strings.Replace(id, prefixMetaData, "", 1)
	err = result.RedisToFile()
	return result, err
}

// GetAllMetaDataIds returns all Ids that contain metadata
func (p DatabaseProvider) GetAllMetaDataIds() []string {
	result := make([]string, 0)
	for _, key := range p.getAllKeysWithPrefix(prefixMetaData) {
		result = append(result, strings.Replace(key, prefixMetaData, "", 1))
	}
	return result
}

// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
func (p DatabaseProvider) GetMetaDataById(id string) (models.File, bool) {
	result, ok := p.getHashMap(prefixMetaData + id)
	if !ok {
		return models.File{}, false
	}
	file, err := newDbToMetadata(id, result)
	helper.Check(err)
	return file, true
}

// SaveMetaData stores the metadata of a file to the disk
func (p DatabaseProvider) SaveMetaData(file models.File) {
	err := file.FileToRedis()
	helper.Check(err)
	p.setHashMap(p.buildArgs(prefixMetaData + file.Id).AddFlat(file))
}

// DeleteMetaData deletes information about a file
func (p DatabaseProvider) DeleteMetaData(id string) {
	p.deleteKey(prefixMetaData + id)
}

// IncreaseDownloadCount increases the download count of a file, preventing race conditions
func (p DatabaseProvider) IncreaseDownloadCount(id string, decreaseRemainingDownloads bool) {
	if decreaseRemainingDownloads {
		p.decreaseHashmapIntField(prefixMetaData+id, "DownloadsRemaining")
	}
	p.increaseHashmapIntField(prefixMetaData+id, "DownloadCount")
}
