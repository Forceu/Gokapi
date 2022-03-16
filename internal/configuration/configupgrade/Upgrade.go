package configupgrade

import (
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
	"time"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 11

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

	// < v1.2.0
	if settings.ConfigVersion < 6 {
		fmt.Println("Please update to version 1.2 before running this version,")
		osExit(1)
		return
	}
	// < v1.3.0
	if settings.ConfigVersion < 7 {
		settings.UseSsl = false
	}
	// < v1.3.1
	if settings.ConfigVersion < 8 {
		settings.MaxFileSizeMB = env.MaxFileSize
	}
	// < v1.5.0-dev1
	if settings.ConfigVersion < 10 {
		settings.Authentication.Method = 0 // authentication.AuthenticationInternal
		settings.Authentication.HeaderUsers = []string{}
		settings.Authentication.OauthUsers = []string{}
		legacyConfig := loadLegacyConfigPreAuth(env)
		settings.Authentication.Username = legacyConfig.AdminName
		settings.Authentication.Password = legacyConfig.AdminPassword
		if legacyConfig.SaltAdmin != "" {
			settings.Authentication.SaltAdmin = legacyConfig.SaltAdmin
		}
		if legacyConfig.SaltFiles != "" {
			settings.Authentication.SaltFiles = legacyConfig.SaltFiles
		}
	}
	// < v1.5.0-dev2
	if settings.ConfigVersion < 11 {
		legacyConfig := loadLegacyConfigPreDb(env)
		uploadValues := models.LastUploadValues{
			Downloads:  legacyConfig.DefaultDownloads,
			TimeExpiry: legacyConfig.DefaultExpiry,
			Password:   legacyConfig.DefaultPassword,
		}
		database.SaveUploadDefaults(uploadValues)

		for _, hotlink := range legacyConfig.Hotlinks {
			database.SaveHotlink(models.File{Id: hotlink.FileId, HotlinkId: hotlink.Id})
		}
		for _, apikey := range legacyConfig.ApiKeys {
			database.SaveApiKey(apikey, false)
		}
		for _, file := range legacyConfig.Files {
			database.SaveMetaData(file)
		}
		for key, session := range legacyConfig.Sessions {
			database.SaveSession(key, session, 48*time.Hour)
		}
	}
}

func loadLegacyConfigPreAuth(env *environment.Environment) configurationLegacyPreAuth {
	file, err := os.Open(env.ConfigPath)
	defer file.Close()
	helper.Check(err)
	decoder := json.NewDecoder(file)

	result := configurationLegacyPreAuth{}
	err = decoder.Decode(&result)
	helper.Check(err)
	return result
}

func loadLegacyConfigPreDb(env *environment.Environment) configurationLegacyPreDb {
	file, err := os.Open(env.ConfigPath)
	defer file.Close()
	helper.Check(err)
	decoder := json.NewDecoder(file)

	result := configurationLegacyPreDb{}
	err = decoder.Decode(&result)
	helper.Check(err)
	return result
}

// configurationLegacyPreAuth is a struct that contains missing values for the global configuration when loading  pre v1.5-dev format
type configurationLegacyPreAuth struct {
	AdminName     string `json:"AdminName"`
	AdminPassword string `json:"AdminPassword"`
	SaltAdmin     string `json:"SaltAdmin"`
	SaltFiles     string `json:"SaltFiles"`
}

// configurationLegacyPreAuth is a struct that contains missing values for the global configuration when loading  pre v1.5 format
type configurationLegacyPreDb struct {
	DefaultDownloads int                       `json:"DefaultDownloads"`
	DefaultExpiry    int                       `json:"DefaultExpiry"`
	DefaultPassword  string                    `json:"DefaultPassword"`
	Files            map[string]models.File    `json:"Files"`
	Hotlinks         map[string]Hotlink        `json:"Hotlinks"`
	ApiKeys          map[string]models.ApiKey  `json:"ApiKeys"`
	Sessions         map[string]models.Session `json:"Sessions"`
}

// Hotlink is a legacy struct containing hotlink ids
type Hotlink struct {
	Id     string `json:"Id"`
	FileId string `json:"FileId"`
}

var osExit = os.Exit
