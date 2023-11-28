package configupgrade

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/configuration/database/legacydb"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 16

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

	// < v1.7.0
	if settings.ConfigVersion < 13 {
		fmt.Println("Please update to version 1.7 before running this version,")
		osExit(1)
		return
	}
	// < v1.7.2
	if settings.ConfigVersion < 14 {
		settings.PublicName = "Gokapi"
	}
	// < v1.8.0beta1
	if settings.ConfigVersion < 15 {
		fmt.Println("Migrating to SQLite...")
		migrateToSqlite(env)
		fmt.Println("Migration complete. You will need to login again.")
		fmt.Println("It should be safe to delete the folder " + env.LegacyDbPath)
	}
	// < v1.8.0beta2
	if settings.ConfigVersion < 16 {
		exists, err := database.ColumnExists("ApiKeys", "Permissions")
		helper.Check(err)
		if !exists {
			err = database.RawSqlite("ALTER TABLE ApiKeys ADD	Permissions	INTEGER NOT NULL DEFAULT 0;")
			helper.Check(err)
		}
		apikeys := database.GetAllApiKeys()
		for _, apikey := range apikeys {
			apikey.Permissions = models.ApiPermAllNoApiMod
			database.SaveApiKey(apikey)
		}
	}
}

// migrateToSqlite copies the content of the old bitcask database to a new sqlite database
// Sessions and Uploadchunks will not be migrated.
func migrateToSqlite(env *environment.Environment) {
	legacydb.Init(env.LegacyDbPath)

	apikeys := legacydb.GetAllApiKeys()
	for _, apikey := range apikeys {
		database.SaveApiKey(apikey)
	}

	e2econfig := legacydb.GetEnd2EndInfo()
	database.SaveEnd2EndInfo(e2econfig)

	files := legacydb.GetAllMetadata()
	for _, file := range files {
		database.SaveMetaData(file)
		if file.HotlinkId != "" {
			database.SaveHotlink(file)
		}
	}

	uploadConfig := legacydb.GetUploadDefaults()
	database.SaveUploadDefaults(uploadConfig)

	legacydb.Close()
}

var osExit = os.Exit
