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
	"Gokapi/internal/webserver/downloadstatus"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Min length of admin password in characters
const minLengthPassword = 6

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings models.Configuration

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
	dataStorage.Init(Environment.FileDbPath)
	if configUpgrade.DoUpgrade(&serverSettings, &Environment) {
		save()
	}
	envMaxMem := os.Getenv("GOKAPI_MAX_MEMORY_UPLOAD")
	if envMaxMem != "" {
		serverSettings.MaxMemory = Environment.MaxMemory
	}
	helper.CreateDir(serverSettings.DataDir)
	downloadstatus.Init()
	serverSettings.Encryption = true // TODO
	log.Init(Environment.DataDir)
}

// Get returns a pointer to the server configuration
func Get() *models.Configuration {
	return &serverSettings
}

// Save the configuration as a json file
func save() {
	file, err := os.OpenFile(Environment.ConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(serverSettings.ToJson()))
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}

func LoadFromSetup(config models.Configuration, cloudConfig *cloudconfig.CloudConfig, isInitialSetup bool) {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)
	if !isInitialSetup {
		Load()
		dataStorage.DeleteAllSessions()
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
