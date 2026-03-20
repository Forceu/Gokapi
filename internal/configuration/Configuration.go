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
	"io"
	"os"
	"strings"

	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/filesystem"
	"golang.org/x/crypto/argon2"
)

// parsedEnvironment is an object containing the environment variables
var parsedEnvironment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings models.Configuration

var usesHttps bool

// Exists returns true if configuration files are present
func Exists() bool {
	configPath, _, _, _ := environment.GetConfigPaths()
	exists, err := helper.FileExists(configPath)
	helper.Check(err)
	return exists
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
	parsedEnvironment = environment.New()
	// No check if file exists, as this was checked earlier
	settings, err := loadFromFile(parsedEnvironment.ConfigPath)
	helper.Check(err)
	serverSettings = settings
	usesHttps = strings.HasPrefix(strings.ToLower(serverSettings.ServerUrl), "https://")

	if configupgrade.DoUpgrade(&serverSettings, &parsedEnvironment) {
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
	logging.Init(serverSettings.DataDir)
}

// ConnectDatabase loads the database that is defined in the configuration
func ConnectDatabase() {
	dbConfig, err := database.ParseUrl(serverSettings.DatabaseUrl, false)
	helper.Check(err)
	database.Connect(dbConfig)
	database.Upgrade()
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
	file, err := os.OpenFile(parsedEnvironment.ConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
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
	parsedEnvironment = environment.New()
	helper.CreateDir(parsedEnvironment.ConfigDir)

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

// GetEnvironment returns a copy of the environment object
func GetEnvironment() environment.Environment {
	if !parsedEnvironment.IsParsed() {
		panic("Environment is not parsed yet")
	}
	return parsedEnvironment
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

// SetDeploymentPassword sets a new password. This should only be used for non-interactive deployment but is not enforced
func SetDeploymentPassword(newPassword string) {
	if len(newPassword) < parsedEnvironment.MinLengthPassword {
		fmt.Printf("Password needs to be at least %d characters long\n", parsedEnvironment.MinLengthPassword)
		os.Exit(1)
	}
	serverSettings.Authentication.SaltAdmin = helper.GenerateRandomString(30)
	err := database.EditSuperAdmin(serverSettings.Authentication.Username, HashPassword(newPassword, false, ""))
	if err != nil {
		fmt.Println("No super-admin user found, but database contains other users. Aborting.")
		os.Exit(1)
	}
	user, _ := database.GetSuperAdmin()
	database.DeleteAllSessionsByUser(user.Id)
	save()
	fmt.Println("New password has been set successfully for user " + serverSettings.Authentication.Username + ".")
	os.Exit(0)
}

// Deprecated: SHA1 is not secure, this is only used for migrating
// passwords from <v2.2.5 to the current version
// Will be removed soon.
func hashSha1(password, salt string) string {
	pwBytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(pwBytes)
	return hex.EncodeToString(hash.Sum(nil))
}

const (
	argonTime    = 2
	argonMemory  = 28 * 1024 // 28 MB
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword hashes a password with Argon2id.
// useOldHash is used for migrating from <v2.2.5 to the current version
// Will be removed soon.
// legacySalt is only used for migrating from <v2.2.5 to the current version
func HashPassword(password string, useOldHash bool, legacySalt string) string {
	if password == "" {
		return ""
	}
	pwBytes := []byte(password + legacySalt)
	if useOldHash {
		if legacySalt == "" {
			panic(errors.New("no salt provided for legacy hash"))
		}
		hash := sha1.New()
		hash.Write(pwBytes)
		return hex.EncodeToString(hash.Sum(nil))
	}
	// Argon2id: generate a fresh random salt, ignore the global salt
	randomSalt := []byte(helper.GenerateRandomString(argonSaltLen))
	hash := argon2.IDKey(
		[]byte(password),
		randomSalt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	return fmt.Sprintf("argon2id$%s$%s",
		hex.EncodeToString(randomSalt),
		hex.EncodeToString(hash),
	)
}

// VerifyPassword checks a plaintext password against a stored hash.
// If hash is still SHA1, it will check the sha1 hash and return the second parameter as true, to indicate
// that the hash was generated with the old hash function and requires rehashing
// Oherwise argon2 will be used and the second parameter will be false
func VerifyPassword(password, storedHash, legacySalt string) (bool, bool) {
	if len(storedHash) == 40 {
		hashedPassword := hashSha1(password, legacySalt)
		return helper.IsEqualStringConstantTime(hashedPassword, storedHash), true
	}

	parts := strings.Split(storedHash, "$")
	if len(parts) != 3 || parts[0] != "argon2id" {
		return false, false
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false, false
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)
	hashedPassword := hex.EncodeToString(hash)
	return helper.IsEqualStringConstantTime(hashedPassword, parts[2]), false
}

// End2EndReconfigParameters contains values on how to reset E2E, if requested
type End2EndReconfigParameters struct {
	DeleteEnd2EndEncryption bool
	DeleteEncryptedStorage  bool
}
