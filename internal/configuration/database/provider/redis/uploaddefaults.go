package redis

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"strconv"
)

const (
	idUploadDefaults = "uploadDefaults"

	hashmapUploadDefDownloads     = "downloads"
	hashmapUploadDefExpiry        = "expiry"
	hashmapUploadDefPassword      = "pw"
	hashmapUploadDefUnlimitedDown = "unl_down"
	hashmapUploadDefUnlimitedTime = "unl_time"
)

func dbToUploadDefaults(input map[string]string) (models.LastUploadValues, error) {
	downloads, err := strconv.Atoi(input[hashmapUploadDefDownloads])
	if err != nil {
		return models.LastUploadValues{}, err
	}
	timeExpiry, err := strconv.Atoi(input[hashmapUploadDefExpiry])
	helper.Check(err)
	if err != nil {
		return models.LastUploadValues{}, err
	}
	result := models.LastUploadValues{
		Downloads:         downloads,
		TimeExpiry:        timeExpiry,
		Password:          input[hashmapUploadDefPassword],
		UnlimitedDownload: input[hashmapUploadDefUnlimitedDown] == "1",
		UnlimitedTime:     input[hashmapUploadDefUnlimitedTime] == "1",
	}
	return result, nil
}

func uploadDefaultsToDb(input models.LastUploadValues) map[string]string {
	unlimitedDown := "0"
	unlimitedTime := "0"

	if input.UnlimitedDownload {
		unlimitedDown = "1"
	}
	if input.UnlimitedTime {
		unlimitedTime = "1"
	}

	return map[string]string{
		hashmapUploadDefDownloads:     strconv.Itoa(input.Downloads),
		hashmapUploadDefExpiry:        strconv.Itoa(input.TimeExpiry),
		hashmapUploadDefPassword:      input.Password,
		hashmapUploadDefUnlimitedDown: unlimitedDown,
		hashmapUploadDefUnlimitedTime: unlimitedTime,
	}
}

// GetUploadDefaults returns the last used setting for amount of downloads allowed, last expiry in days and
// a password for the file
func (p DatabaseProvider) GetUploadDefaults() (models.LastUploadValues, bool) {

	values, ok := getHashMap(idUploadDefaults)
	if !ok {
		return models.LastUploadValues{}, false
	}

	result, err := dbToUploadDefaults(values)
	helper.Check(err)
	return result, true
}

// SaveUploadDefaults saves the last used setting for an upload
func (p DatabaseProvider) SaveUploadDefaults(values models.LastUploadValues) {
	setHashMap(idUploadDefaults, uploadDefaultsToDb(values))
}
