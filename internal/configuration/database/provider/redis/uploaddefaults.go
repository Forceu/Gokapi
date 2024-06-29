package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	idUploadDefaults = "uploadDefaults"
)

// GetUploadDefaults returns the last used setting for amount of downloads allowed, last expiry in days and
// a password for the file
func (p DatabaseProvider) GetUploadDefaults() (models.LastUploadValues, bool) {
	var result models.LastUploadValues
	values, ok := p.getHashMap(idUploadDefaults)
	if !ok {
		return models.LastUploadValues{}, false
	}

	err := redigo.ScanStruct(values, &result)
	helper.Check(err)
	return result, true
}

// SaveUploadDefaults saves the last used setting for an upload
func (p DatabaseProvider) SaveUploadDefaults(values models.LastUploadValues) {
	p.setHashMap(p.buildArgs(idUploadDefaults).AddFlat(values))
}
