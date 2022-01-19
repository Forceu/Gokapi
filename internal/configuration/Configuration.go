package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"Gokapi/internal/configuration/cloudconfig"
	"Gokapi/internal/configuration/configUpgrade"
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	log "Gokapi/internal/logging"
	"Gokapi/internal/models"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// Min length of admin password in characters
const minLengthPassword = 6

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings models.Configuration

// For locking this object to prevent race conditions
var mutex sync.RWMutex

func Exists() bool {
	configPath, _, _, _ := environment.GetConfigPaths()
	return helper.FileExists(configPath)
}

// Load loads the configuration or creates the folder structure and a default configuration
func Load() {
	Environment = environment.New()
	// No check if file exists, as this was checked earlier
	file, err := os.Open(Environment.ConfigPath)
	helper.Check(err)
	decoder := json.NewDecoder(file)
	serverSettings = models.Configuration{}
	err = decoder.Decode(&serverSettings)
	helper.Check(err)
	file.Close()
	if configUpgrade.DoUpgrade(&serverSettings, &Environment) {
		save()
	}
	envMaxMem := os.Getenv("GOKAPI_MAX_MEMORY_UPLOAD")
	if envMaxMem != "" {
		serverSettings.MaxMemory = Environment.MaxMemory
	}
	helper.CreateDir(serverSettings.DataDir)
	dataStorage.Init(Environment.FileDbPath)
	loadUploadDefaults()
	log.Init(Environment.ConfigDir)
}

// Lock locks configuration to prevent race conditions (blocking)
func Lock() {
	mutex.Lock()
}

func loadUploadDefaults() {
	downloads, expiry, password := dataStorage.GetUploadDefaults()
	serverSettings.DefaultDownloads = downloads
	serverSettings.DefaultExpiry = expiry
	serverSettings.DefaultPassword = password
}

// ReleaseAndSave unlocks and saves the configuration
func ReleaseAndSave() {
	save()
	mutex.Unlock()
}

// Release unlocks the configuration
func Release() {
	mutex.Unlock()
}

// GetServerSettings locks the settings returns a pointer to the configuration for Read/Write access
// Release needs to be called when finished with the operation!
func GetServerSettings() *models.Configuration {
	mutex.Lock()
	return &serverSettings
}

// GetServerSettingsReadOnly locks the settings for read-only access and returns a copy of the configuration
// ReleaseReadOnly needs to be called when finished with the operation!
func GetServerSettingsReadOnly() *models.Configuration {
	mutex.RLock()
	return &serverSettings
}

// ReleaseReadOnly unlocks the configuration opened for read-only access
func ReleaseReadOnly() {
	mutex.RUnlock()
}

// Save the configuration as a json file
func save() {
	file, err := os.OpenFile(Environment.ConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&serverSettings)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}

func LoadFromSetup(config models.Configuration, cloudConfig *cloudconfig.CloudConfig, isInitialConfig bool) {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)
	if !isInitialConfig {
		Load()
		config.DefaultDownloads = serverSettings.DefaultDownloads
		config.DefaultExpiry = serverSettings.DefaultExpiry
		config.DefaultPassword = serverSettings.DefaultPassword
		config.Files = serverSettings.Files
		config.Hotlinks = serverSettings.Hotlinks
		config.ApiKeys = serverSettings.ApiKeys
	}

	serverSettings = config
	if cloudConfig != nil {
		err := cloudconfig.Write(*cloudConfig)
		if err != nil {
			fmt.Println("Error writing cloud configuration:", err)
			os.Exit(1)
		}
	} else {
		err := cloudconfig.Delete()
		if err != nil {
			fmt.Println("Error deleting cloud configuration:", err)
			os.Exit(1)
		}
	}
	save()
}

// GetLengthId returns the length of the file IDs to be generated
func GetLengthId() int {
	return serverSettings.LengthId
}

// HashPassword hashes a string with SHA256 and a salt
func HashPassword(password string, useFileSalt bool) string {
	if useFileSalt {
		return HashPasswordCustomSalt(password, serverSettings.Authentication.SaltFiles)
	}
	return HashPasswordCustomSalt(password, serverSettings.Authentication.SaltAdmin)
}

func HashPasswordCustomSalt(password, salt string) string {
	if password == "" {
		return ""
	}
	if salt == "" {
		panic(errors.New("no salt provided"))
	}
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}
