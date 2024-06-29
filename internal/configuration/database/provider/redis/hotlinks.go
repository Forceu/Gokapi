package redis

import (
	"github.com/forceu/gokapi/internal/models"
	"strings"
)

const (
	prefixHotlinks = "hl:"
)

// GetHotlink returns the id of the file associated or false if not found
func (p DatabaseProvider) GetHotlink(id string) (string, bool) {
	return p.getKeyString(prefixHotlinks + id)
}

// GetAllHotlinks returns an array with all hotlink ids
func (p DatabaseProvider) GetAllHotlinks() []string {
	result := make([]string, 0)
	for _, key := range p.getAllKeysWithPrefix(prefixHotlinks) {
		result = append(result, strings.Replace(key, prefixHotlinks, "", 1))
	}
	return result
}

// SaveHotlink stores the hotlink associated with the file in the database
func (p DatabaseProvider) SaveHotlink(file models.File) {
	p.setKey(prefixHotlinks+file.HotlinkId, file.Id)
}

// DeleteHotlink deletes a hotlink with the given hotlink ID
func (p DatabaseProvider) DeleteHotlink(id string) {
	p.deleteKey(prefixHotlinks + id)
}
