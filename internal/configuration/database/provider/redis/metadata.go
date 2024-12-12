package redis

import (
	"bytes"
	"encoding/gob"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strings"
)

const (
	prefixMetaData = "fmeta:"
)

func dbToMetaData(input []byte) models.File {
	var result models.File
	buf := bytes.NewBuffer(input)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&result)
	helper.Check(err)
	return result
}

// GetAllMetadata returns a map of all available files
func (p DatabaseProvider) GetAllMetadata() map[string]models.File {
	result := make(map[string]models.File)
	allMetaData := p.getAllValuesWithPrefix(prefixMetaData)
	for _, metaData := range allMetaData {
		content, err := redigo.Bytes(metaData, nil)
		helper.Check(err)
		file := dbToMetaData(content)
		result[file.Id] = file
	}
	return result
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
	input, ok := p.getKeyBytes(prefixMetaData + id)
	if !ok {
		return models.File{}, false
	}
	return dbToMetaData(input), true
}

// SaveMetaData stores the metadata of a file to the disk
func (p DatabaseProvider) SaveMetaData(file models.File) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(file)
	helper.Check(err)
	p.setKey(prefixMetaData+file.Id, buf.Bytes())
}

// DeleteMetaData deletes information about a file
func (p DatabaseProvider) DeleteMetaData(id string) {
	p.deleteKey(prefixMetaData + id)
}

// IncreaseDownloadCount increases the download count of a file, preventing race conditions
func (p DatabaseProvider) IncreaseDownloadCount(id string, decreaseRemainingDownloads bool) {
	if decreaseRemainingDownloads {
	} else {
	}
	// TODO
}
