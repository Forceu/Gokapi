package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"Gokapi/internal/configuration/configUpgrade"
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
	"strconv"
	"sync"
	"unicode/utf8"
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
	helper.CreateDir(Environment.ConfigDir)
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
	serverSettings.MaxMemory = Environment.MaxMemory
	helper.CreateDir(serverSettings.DataDir)
	log.Init(Environment.ConfigDir)
}

// Lock locks configuration to prevent race conditions (blocking)
func Lock() {
	mutex.Lock()
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
		osExit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&serverSettings)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		osExit(1)
	}
}

func LoadFromSetup(config models.Configuration) {
	Environment = environment.New()
	serverSettings = config
	save()
}

// Asks for password or loads it from env and returns input as string if valid
func askForPassword() string {
	fmt.Print("Password: ")
	envPassword := Environment.AdminPassword
	if envPassword != "" {
		fmt.Println("*******************")
		if utf8.RuneCountInString(envPassword) < minLengthPassword {
			fmt.Println("\nPassword needs to be at least " + strconv.Itoa(minLengthPassword) + " characters long")
			osExit(1)
		}
		return envPassword
	}
	password1 := helper.ReadPassword()
	if utf8.RuneCountInString(password1) < minLengthPassword {
		fmt.Println("\nPassword needs to be at least " + strconv.Itoa(minLengthPassword) + " characters long")
		return askForPassword()
	}
	fmt.Print("\nPassword (repeat): ")
	password2 := helper.ReadPassword()
	if password1 != password2 {
		fmt.Println("\nPasswords dont match")
		return askForPassword()
	}
	fmt.Println()
	return password1
}

// DisplayPasswordReset shows a password prompt in the CLI and saves the new password
func DisplayPasswordReset() {
	serverSettings.Authentication.Password = HashPassword(askForPassword(), false)
	// Log out all sessions
	serverSettings.Sessions = make(map[string]models.Session)
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

var osExit = os.Exit
