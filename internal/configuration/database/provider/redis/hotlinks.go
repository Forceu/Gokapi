package redis

import (
	"github.com/forceu/gokapi/internal/models"
)

const (
	prefixHotlinks = "hl:"
)

// GetHotlink returns the id of the file associated or false if not found
func (p DatabaseProvider) GetHotlink(id string) (string, bool) {
	return getKeyString(prefixHotlinks + id)
}

// GetAllHotlinks returns an array with all hotlink ids
func (p DatabaseProvider) GetAllHotlinks() []string {
	return getAllKeysWithPrefix(prefixHotlinks)
}

// SaveHotlink stores the hotlink associated with the file in the database
func (p DatabaseProvider) SaveHotlink(file models.File) {
	setKey(prefixHotlinks+file.HotlinkId, file.Id)
}

// DeleteHotlink deletes a hotlink with the given hotlink ID
func (p DatabaseProvider) DeleteHotlink(id string) {
	deleteKey(prefixHotlinks + id)
}
