package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const idE2EInfo = "e2einfo"

// SaveEnd2EndInfo stores the encrypted e2e info
func (p DatabaseProvider) SaveEnd2EndInfo(info models.E2EInfoEncrypted) {
	setHashMap(buildArgs(idE2EInfo).AddFlat(info))
}

// GetEnd2EndInfo retrieves the encrypted e2e info
func (p DatabaseProvider) GetEnd2EndInfo() models.E2EInfoEncrypted {
	result := models.E2EInfoEncrypted{}
	value, ok := getHashMap(idE2EInfo)
	if !ok {
		return models.E2EInfoEncrypted{}
	}
	err := redigo.ScanStruct(value, &result)
	helper.Check(err)
	return result
}

// DeleteEnd2EndInfo resets the encrypted e2e info
func (p DatabaseProvider) DeleteEnd2EndInfo() {
	deleteKey(idE2EInfo)
}
