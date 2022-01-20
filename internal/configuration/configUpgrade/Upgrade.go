package configUpgrade

import (
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 11

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
		os.Exit(1)
	}
	// < v1.3.0
	if settings.ConfigVersion < 7 {
		settings.UseSsl = false
	}
	// < v1.3.1
	if settings.ConfigVersion < 8 {
		settings.MaxFileSizeMB = env.MaxFileSize
	}
	// < v1.5.0-dev
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
	// < v1.5.0
	if settings.ConfigVersion < 11 {
		legacyConfig := loadLegacyConfigPreDb(env)
		dataStorage.SaveUploadDefaults(legacyConfig.DefaultDownloads, legacyConfig.DefaultExpiry, legacyConfig.DefaultPassword)

		for _, hotlink := range legacyConfig.Hotlinks {
			dataStorage.SaveHotlink(hotlink.Id, models.File{Id: hotlink.FileId})
		}
		for _, apikey := range legacyConfig.ApiKeys {
			dataStorage.SaveApiKey(apikey, false)
		}
		for _, file := range legacyConfig.Files {
			dataStorage.SaveMetaData(file)
		}
		for key, session := range legacyConfig.Sessions {
			dataStorage.SaveSession(key, session, 48*time.Hour)
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
