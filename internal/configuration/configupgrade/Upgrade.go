package configupgrade

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/inhies/go-bytesize"
	"github.com/jinzhu/copier"
	"os"
	"strconv"
	"strings"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 13

// DoUpgrade checks if an old version is present and updates it to the current version if required
func DoUpgrade(settings *models.Configuration, env *environment.Environment) bool {
	if settings.ConfigVersion < CurrentConfigVersion {
		updateConfig(settings, env)
		fmt.Println("Successfully upgraded database")
		settings.ConfigVersion = CurrentConfigVersion
		return true
	}
	return false
}

// Upgrades the settings if saved with a previous version
func updateConfig(settings *models.Configuration, env *environment.Environment) {

	// < v1.5.0
	if settings.ConfigVersion < 11 {
		fmt.Println("Please update to version 1.5 before running this version,")
		osExit(1)
		return
	}
	// < v1.6.0
	if settings.ConfigVersion < 12 {
		keys := database.GetAllMetaDataIds()
		for _, key := range keys {
			raw, ok := database.GetRawKey("file:id:" + key)
			if !ok {
				panic("could not read raw key for upgrade")
			}
			file := legacyFileToCurrentFile(raw)
			database.SaveMetaData(file)
		}
	}
	// < v1.6.2
	if settings.ConfigVersion < 13 {
		data := database.GetAllMetadata()
		for _, file := range data {
			b, err := bytesize.Parse(file.Size)
			if err == nil {
				bytesFormatted := b.Format("%.0f", "b", false)
				bytesFormatted = strings.Replace(bytesFormatted, "B", "", 1)
				sizeBytes, err := strconv.ParseInt(bytesFormatted, 10, 64)
				if err == nil {
					file.SizeBytes = sizeBytes
					database.SaveMetaData(file)
				}
			}
		}
	}
}

func legacyFileToCurrentFile(input []byte) models.File {
	oldFile := legacyFile{}
	buf := bytes.NewBuffer(input)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&oldFile)
	helper.Check(err)
	result := models.File{}
	err = copier.Copy(&result, oldFile)
	helper.Check(err)
	result.SHA1 = oldFile.SHA256
	return result
}

type legacyFile struct {
	Id                           string                `json:"Id"`
	Name                         string                `json:"Name"`
	Size                         string                `json:"Size"`
	SHA256                       string                `json:"SHA256"`
	ExpireAt                     int64                 `json:"ExpireAt"`
	ExpireAtString               string                `json:"ExpireAtString"`
	DownloadsRemaining           int                   `json:"DownloadsRemaining"`
	DownloadCount                int                   `json:"DownloadCount"`
	PasswordHash                 string                `json:"PasswordHash"`
	HotlinkId                    string                `json:"HotlinkId"`
	ContentType                  string                `json:"ContentType"`
	AwsBucket                    string                `json:"AwsBucket"`
	Encryption                   models.EncryptionInfo `json:"Encryption"`
	UnlimitedDownloads           bool                  `json:"UnlimitedDownloads"`
	UnlimitedTime                bool                  `json:"UnlimitedTime"`
	RequiresClientSideDecryption bool                  `json:"RequiresClientSideDecryption"`
}

var osExit = os.Exit
