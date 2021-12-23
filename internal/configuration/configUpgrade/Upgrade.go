package configUpgrade

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"encoding/json"
	"fmt"
	"os"
)

func DoUpgrade(settings *models.Configuration, env *environment.Environment, currentVersion int) bool {
	if settings.ConfigVersion < currentVersion {
		updateConfig(settings, env)
		fmt.Println("Successfully upgraded database")
		settings.ConfigVersion = currentVersion
		return true
	}
	return false
}

// Upgrades the settings if saved with a previous version
func updateConfig(settings *models.Configuration, env *environment.Environment) {
	// < v1.1.2
	if settings.ConfigVersion < 3 {
		settings.Authentication.SaltAdmin = "eefwkjqweduiotbrkl##$2342brerlk2321"
		settings.Authentication.SaltFiles = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"
		settings.LengthId = 15
		settings.DataDir = env.DataDir
	}
	// < v1.1.3
	if settings.ConfigVersion < 4 {
		settings.Hotlinks = make(map[string]models.Hotlink)
	}
	// < v1.1.4
	if settings.ConfigVersion < 5 {
		settings.LengthId = 15
		settings.DownloadStatus = make(map[string]models.DownloadStatus)
		for _, file := range settings.Files {
			file.ContentType = "application/octet-stream"
			settings.Files[file.Id] = file
		}
	}
	// < v1.2.0
	if settings.ConfigVersion < 6 {
		settings.ApiKeys = make(map[string]models.ApiKey)
	}
	// < v1.3.0
	if settings.ConfigVersion < 7 {
		if env.UseSsl == environment.IsTrue {
			settings.UseSsl = true
		}
	}
	// < v1.3.1
	if settings.ConfigVersion < 8 {
		settings.MaxFileSizeMB = env.MaxFileSize
	}
	// < v1.5.0
	if settings.ConfigVersion < 10 {
		settings.Authentication.Method = models.AuthenticationInternal
		settings.Authentication.HeaderUsers = []string{}
		settings.Authentication.OauthUsers = []string{}
		legacyConfig := loadLegacyConfig(env)
		settings.Authentication.Username = legacyConfig.AdminName
		settings.Authentication.Password = legacyConfig.AdminPassword
		settings.Authentication.SaltAdmin = legacyConfig.SaltAdmin
		settings.Authentication.SaltFiles = legacyConfig.SaltFiles
	}
}

func loadLegacyConfig(env *environment.Environment) configurationLegacy {
	file, err := os.Open(env.ConfigPath)
	helper.Check(err)
	decoder := json.NewDecoder(file)
	file.Close()

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
