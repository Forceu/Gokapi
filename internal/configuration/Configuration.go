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
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/filesystem"
	"io"
	"os"
	"strings"
)

// MinLengthPassword is the required length of admin password in characters
const MinLengthPassword = 8

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

// loadFromFile parses the given file and adds salts, if they are invalid
func loadFromFile(path string) (models.Configuration, error) {
	file, err := os.Open(path)
	if err != nil {
		return models.Configuration{}, err
	}
	decoder := json.NewDecoder(file)
	settings := models.Configuration{}
	err = decoder.Decode(&settings)
	if err != nil {
		return models.Configuration{}, err
	}
	err = file.Close()
	if err != nil {
		return models.Configuration{}, err
	}
	if len(settings.Authentication.SaltFiles) < 20 {
		settings.Authentication.SaltFiles = helper.GenerateRandomString(30)
		fmt.Println("Warning: Salt for file hash invalid, generating new salt")
	}
	if len(settings.Authentication.SaltAdmin) < 20 {
		settings.Authentication.SaltAdmin = helper.GenerateRandomString(30)
		if settings.Authentication.Method == 0 { // == authentication.Internal, but would create import cycle
			fmt.Println("Warning: Salt for admin password invalid, generating new salt. You will need to reset the admin password.")
		}
	}
	return settings, nil
}

// Load loads the configuration or creates the folder structure and a default configuration
func Load() {
	Environment = environment.New()
	// No check if file exists, as this was checked earlier
	settings, err := loadFromFile(Environment.ConfigPath)
	helper.Check(err)
	serverSettings = settings
	usesHttps = strings.HasPrefix(strings.ToLower(serverSettings.ServerUrl), "https://")

	if configupgrade.DoUpgrade(&serverSettings, &Environment) {
		save()
	}
	if serverSettings.PublicName == "" {
		serverSettings.PublicName = "Gokapi"
	}
	if serverSettings.MaxParallelUploads == 0 {
		serverSettings.MaxParallelUploads = 4
	}
	if serverSettings.ChunkSize == 0 {
		serverSettings.ChunkSize = 45
	}
	helper.CreateDir(serverSettings.DataDir)
	filesystem.Init(serverSettings.DataDir)
	logging.Init(Environment.DataDir)
}

// ConnectDatabase loads the database that is defined in the configuration
func ConnectDatabase() {
	dbConfig, err := database.ParseUrl(serverSettings.DatabaseUrl, false)
	helper.Check(err)
	database.Connect(dbConfig)
	database.Upgrade()
}

// MigrateToV2 is used to migrate the previous admin user to the DB
func MigrateToV2(authPassword string, allowedUsers []string) {
	fmt.Println("Migrating v1 user, metadata and API keys to v2 scheme...")
	var adminName = "admin@gokapi"
	if serverSettings.Authentication.Method != models.AuthenticationDisabled &&
		serverSettings.Authentication.Username != "" {
		adminName = serverSettings.Authentication.Username
	}

	newAdmin := models.User{
		Name:        adminName,
		Permissions: models.UserPermissionAll,
		UserLevel:   models.UserLevelSuperAdmin,
		Password:    authPassword,
	}
	database.SaveUser(newAdmin, true)
	adminUser, ok := database.GetUserByName(adminName)
	if !ok {
		fmt.Println("ERROR: Could not retrieve new admin user after saving")
		os.Exit(1)
	}
	fmt.Println("Created admin user " + adminUser.Name)

	for _, user := range allowedUsers {
		newUser := models.User{
			Name:        user,
			Permissions: models.UserPermissionNone,
			UserLevel:   models.UserLevelUser,
		}
		database.SaveUser(newUser, true)
		fmt.Println("Created admin user ", user)
	}

	for _, apiKey := range database.GetAllApiKeys() {
		apiKey.UserId = adminUser.Id
		apiKey.PublicId = helper.GenerateRandomString(35)
		database.SaveApiKey(apiKey)
	}

	e2eConfig := database.GetEnd2EndInfo(0)
	database.DeleteEnd2EndInfo(0)
	database.SaveEnd2EndInfo(e2eConfig, adminUser.Id)

	for _, file := range database.GetAllMetadata() {
		file.UserId = adminUser.Id
		database.SaveMetaData(file)
	}
	database.DeleteAllSessions()
	logging.UpgradeToV2()
	fmt.Println("Migration complete")
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
		fmt.Println("Error writing configuration:", err)
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
func LoadFromSetup(config models.Configuration, cloudConfig *cloudconfig.CloudConfig, e2eConfig End2EndReconfigParameters, passwordHash string) {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)

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
	ConnectDatabase()
	err := database.EditSuperAdmin(serverSettings.Authentication.Username, passwordHash)
	if err != nil {
		fmt.Println("Could not edit superadmin, as none was found, but other users were present.")
		os.Exit(1)
	}
	database.DeleteAllSessions()
	if e2eConfig.DeleteEnd2EndEncryption {
		for _, user := range database.GetAllUsers() {
			database.DeleteEnd2EndInfo(user.Id)
		}
	}
	if e2eConfig.DeleteEncryptedStorage {
		deleteAllEncryptedStorage()
	}
}

func deleteAllEncryptedStorage() {
	files := database.GetAllMetadata()
	for _, file := range files {
		if file.Encryption.IsEncrypted {
			file.UnlimitedTime = false
			file.ExpireAt = 0
			database.SaveMetaData(file)
		}
	}
}

// SetDeploymentPassword sets a new password. This should only be used for non-interactive deployment, but is not enforced
func SetDeploymentPassword(newPassword string) {
	if len(newPassword) < MinLengthPassword {
		fmt.Printf("Password needs to be at least %d characters long\n", MinLengthPassword)
		os.Exit(1)
	}
	serverSettings.Authentication.SaltAdmin = helper.GenerateRandomString(30)
	err := database.EditSuperAdmin(serverSettings.Authentication.Username, hashUserPassword(newPassword))
	if err != nil {
		fmt.Println("No super-admin user found, but database contains other users. Aborting.")
		os.Exit(1)
	}
	save()
	fmt.Println("New password has been set successfully for user " + serverSettings.Authentication.Username + ".")
	os.Exit(0)
}

// HashPassword hashes a string with SHA1 the file salt or admin user salt
func HashPassword(password string, useFileSalt bool) string {
	if useFileSalt {
		return hashFilePassword(password)
	}
	return hashUserPassword(password)
}

func hashFilePassword(password string) string {
	return HashPasswordCustomSalt(password, serverSettings.Authentication.SaltFiles)
}

func hashUserPassword(password string) string {
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
	pwBytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(pwBytes)
	return hex.EncodeToString(hash.Sum(nil))
}

// End2EndReconfigParameters contains values on how to reset E2E, if requested
type End2EndReconfigParameters struct {
	DeleteEnd2EndEncryption bool
	DeleteEncryptedStorage  bool
}
