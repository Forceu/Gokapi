package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	log "github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/filesystem"
	"io"
	"os"
	"strings"
)

// Min length of admin password in characters
const minLengthPassword = 6

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings models.Configuration

var usesHttps bool

// Exists returns true if configuration files are present
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
	database.Init(Environment.FileDbPath)
	if configupgrade.DoUpgrade(&serverSettings, &Environment) {
		save()
	}
	envMaxMem := os.Getenv("GOKAPI_MAX_MEMORY_UPLOAD")
	if envMaxMem != "" {
		serverSettings.MaxMemory = Environment.MaxMemory
	}
	helper.CreateDir(serverSettings.DataDir)
	filesystem.Init(serverSettings.DataDir)
	log.Init(Environment.DataDir)
	usesHttps = strings.HasPrefix(strings.ToLower(serverSettings.ServerUrl), "https://")
}

// UsesHttps returns true if Gokapi URL is set to a secure URL
func UsesHttps() bool {
	return usesHttps
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

// LoadFromSetup creates a new configuration file after a user completed the setup. If cloudConfig is not nil, a new
// cloud config file is created. If it is nil an existing cloud config file will be deleted.
func LoadFromSetup(config models.Configuration, cloudConfig *cloudconfig.CloudConfig, isInitialSetup bool) {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)
	if !isInitialSetup {
		Load()
		database.DeleteAllSessions()
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
	Load()
}

// HashPassword hashes a string with SHA1 the file salt or admin user salt
func HashPassword(password string, useFileSalt bool) string {
	if useFileSalt {
		return HashPasswordCustomSalt(password, serverSettings.Authentication.SaltFiles)
	}
	return HashPasswordCustomSalt(password, serverSettings.Authentication.SaltAdmin)
}

// HashPasswordCustomSalt hashes a password with SHA1 and the provided salt
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
