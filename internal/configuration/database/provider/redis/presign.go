package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixPresign = "ps:"
)

// GetPresignedUrl returns the presigned url with the given ID or false if not a valid ID
func (p DatabaseProvider) GetPresignedUrl(id string) (models.Presign, bool) {
	hashmapEntry, ok := p.getHashMap(prefixPresign + id)
	if !ok {
		return models.Presign{}, false
	}
	var result models.Presign
	err := redigo.ScanStruct(hashmapEntry, &result)
	helper.Check(err)
	return result, true
}

// SavePresignedUrl saves the presigned url
func (p DatabaseProvider) SavePresignedUrl(presign models.Presign) {
	p.setHashMap(p.buildArgs(prefixPresign + presign.Id).AddFlat(presign))
	p.setExpiryAt(prefixPresign+presign.Id, presign.Expiry)
}

// DeletePresignedUrl deletes the presigned url with the given ID
func (p DatabaseProvider) DeletePresignedUrl(id string) {
	p.deleteKey(prefixPresign + id)
}
