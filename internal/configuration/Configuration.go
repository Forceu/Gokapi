package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	log "Gokapi/internal/logging"
	"Gokapi/internal/models"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"unicode/utf8"
)

// Min length of admin password in characters
const minLengthPassword = 6

// Min length of admin username in characters
const minLengthUsername = 3

const AuthenticationInternal = 0
const AuthenticationOAuth2 = 1
const AuthenticationHeader = 2
const AuthenticationDisabled = 3

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings Configuration

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 10

// For locking this object to prevent race conditions
var mutex sync.RWMutex

// Configuration is a struct that contains the global configuration
type Configuration struct {
	AuthenticationMethod int                              `json:"AuthenticationMethod"`
	Port                 string                           `json:"Port"`
	AdminName            string                           `json:"AdminName"`
	AdminPassword        string                           `json:"AdminPassword"`
	ServerUrl            string                           `json:"ServerUrl"`
	DefaultDownloads     int                              `json:"DefaultDownloads"`
	DefaultExpiry        int                              `json:"DefaultExpiry"`
	DefaultPassword      string                           `json:"DefaultPassword"`
	RedirectUrl          string                           `json:"RedirectUrl"`
	Sessions             map[string]models.Session        `json:"Sessions"`
	Files                map[string]models.File           `json:"Files"`
	Hotlinks             map[string]models.Hotlink        `json:"Hotlinks"`
	DownloadStatus       map[string]models.DownloadStatus `json:"DownloadStatus"`
	ApiKeys              map[string]models.ApiKey         `json:"ApiKeys"`
	ConfigVersion        int                              `json:"ConfigVersion"`
	SaltAdmin            string                           `json:"SaltAdmin"`
	SaltFiles            string                           `json:"SaltFiles"`
	LengthId             int                              `json:"LengthId"`
	DataDir              string                           `json:"DataDir"`
	MaxMemory            int                              `json:"MaxMemory"`
	UseSsl               bool                             `json:"UseSsl"`
	MaxFileSizeMB        int                              `json:"MaxFileSizeMB"`
	LoginHeaderKey       string                           `json:"LoginHeaderKey"`
	LoginHeaderUsers     []string                         `json:"LoginHeaderUsers"`
}

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
	defer file.Close()
	decoder := json.NewDecoder(file)
	serverSettings = Configuration{}
	err = decoder.Decode(&serverSettings)
	helper.Check(err)
	updateConfig()
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
func GetServerSettings() *Configuration {
	mutex.Lock()
	return &serverSettings
}

// GetServerSettingsReadOnly locks the settings for read-only access and returns a copy of the configuration
// ReleaseReadOnly needs to be called when finished with the operation!
func GetServerSettingsReadOnly() *Configuration {
	mutex.RLock()
	return &serverSettings
}

// ReleaseReadOnly unlocks the configuration opened for read-only access
func ReleaseReadOnly() {
	mutex.RUnlock()
}

// Upgrades the ServerSettings if saved with a previous version
func updateConfig() {
	// < v1.1.2
	if serverSettings.ConfigVersion < 3 {
		serverSettings.SaltAdmin = "eefwkjqweduiotbrkl##$2342brerlk2321"
		serverSettings.SaltFiles = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"
		serverSettings.LengthId = 15
		serverSettings.DataDir = Environment.DataDir
	}
	// < v1.1.3
	if serverSettings.ConfigVersion < 4 {
		serverSettings.Hotlinks = make(map[string]models.Hotlink)
	}
	// < v1.1.4
	if serverSettings.ConfigVersion < 5 {
		serverSettings.LengthId = 15
		serverSettings.DownloadStatus = make(map[string]models.DownloadStatus)
		for _, file := range serverSettings.Files {
			file.ContentType = "application/octet-stream"
			serverSettings.Files[file.Id] = file
		}
	}
	// < v1.2.0
	if serverSettings.ConfigVersion < 6 {
		serverSettings.ApiKeys = make(map[string]models.ApiKey)
	}
	// < v1.3.0
	if serverSettings.ConfigVersion < 7 {
		if Environment.UseSsl == environment.IsTrue {
			serverSettings.UseSsl = true
		}
	}
	// < v1.3.1
	if serverSettings.ConfigVersion < 8 {
		serverSettings.MaxFileSizeMB = Environment.MaxFileSize
	}
	// < v1.5.0
	if serverSettings.ConfigVersion < 10 {
		serverSettings.AuthenticationMethod = AuthenticationInternal
	}

	if serverSettings.ConfigVersion < CurrentConfigVersion {
		fmt.Println("Successfully upgraded database")
		serverSettings.ConfigVersion = CurrentConfigVersion
		serverSettings.LoginHeaderUsers = []string{}
		save()
	}
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

func LoadFromSetup(config Configuration) {
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
	serverSettings.AdminPassword = HashPassword(askForPassword(), false)
	// Log out all sessions
	serverSettings.Sessions = make(map[string]models.Session)
	save()
}

// HashPassword hashes a string with SHA256 and a salt
func HashPassword(password string, useFileSalt bool) string {
	if useFileSalt {
		return HashPasswordCustomSalt(password, serverSettings.SaltFiles)
	}
	return HashPasswordCustomSalt(password, serverSettings.SaltAdmin)
}

func HashPasswordCustomSalt(password, salt string) string {
	if password == "" {
		return ""
	}
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}

// GetLengthId returns the length of the file IDs to be generated
func GetLengthId() int {
	return serverSettings.LengthId
}

func IsLogoutAvailable() bool {
	return serverSettings.AuthenticationMethod == AuthenticationInternal
}

var osExit = os.Exit
