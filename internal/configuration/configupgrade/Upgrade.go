package configupgrade

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 20

// DoUpgrade checks if an old version is present and updates it to the current version if required
func DoUpgrade(settings *models.Configuration, env *environment.Environment) bool {
	if settings.ConfigVersion < CurrentConfigVersion {
		updateConfig(settings, env)
		fmt.Println("Successfully upgraded configuration")
		settings.ConfigVersion = CurrentConfigVersion
		return true
	}
	return false
}

// Upgrades the settings if saved with a previous version
func updateConfig(settings *models.Configuration, env *environment.Environment) {

	// < v1.8.0
	if settings.ConfigVersion < 16 {
		fmt.Println("Please update to version 1.8 before running this version,")
		osExit(1)
		return
	}
	// < v1.8.2
	if settings.ConfigVersion < 18 {
		if len(settings.Authentication.OAuthUsers) > 0 {
			settings.Authentication.OAuthUserScope = "email"
		}
		settings.Authentication.OAuthRecheckInterval = 168
	}
	// < v1.8.5beta
	if settings.ConfigVersion < 19 {
		if settings.MaxMemory == 40 {
			settings.MaxMemory = 50
		}
		settings.ChunkSize = env.ChunkSizeMB
		settings.MaxParallelUploads = env.MaxParallelUploads
	}
	// < v1.8.5
	if settings.ConfigVersion < 20 {
		err := database.RawSqlite(`DROP TABLE UploadStatus; CREATE TABLE "UploadStatus" (
	"ChunkId"	TEXT NOT NULL UNIQUE,
	"CurrentStatus"	INTEGER NOT NULL,
	"CreationDate"	INTEGER NOT NULL,
	PRIMARY KEY("ChunkId")
) WITHOUT ROWID;`)
		helper.Check(err)
	}
}

var osExit = os.Exit
