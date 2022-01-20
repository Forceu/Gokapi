package configUpgrade

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"encoding/json"
	"fmt"
	"os"
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
		legacyConfig := loadLegacyConfig(env)
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
		// TODO
	}
}

func loadLegacyConfig(env *environment.Environment) configurationLegacy {
	file, err := os.Open(env.ConfigPath)
	defer file.Close()
	helper.Check(err)
	decoder := json.NewDecoder(file)

	result := configurationLegacy{}
	err = decoder.Decode(&result)
	helper.Check(err)
	return result
}

// configurationLegacy is a struct that contains missing values for the global configuration when loading  pre v1.5 format
type configurationLegacy struct {
	AdminName     string `json:"AdminName"`
	AdminPassword string `json:"AdminPassword"`
	SaltAdmin     string `json:"SaltAdmin"`
	SaltFiles     string `json:"SaltFiles"`
}
