package database

import (
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"net/url"
	"strings"
)

var db dbabstraction.Database

// Connect establishes a connection to the database and creates the table structure, if necessary
func Connect(config models.DbConnection) {
	var err error
	db, err = dbabstraction.GetNew(config)
	if err != nil {
		panic(err)
	}
}

// ParseUrl converts a database URL to a models.DbConnection struct
func ParseUrl(dbUrl string, mustExist bool) (models.DbConnection, error) {
	if dbUrl == "" {
		return models.DbConnection{}, errors.New("dbUrl is empty")
	}
	u, err := url.Parse(dbUrl)
	if err != nil {
		return models.DbConnection{}, fmt.Errorf("unsupported database URL - expected format is: type://username:password@server: %v", err)
	}
	result := models.DbConnection{}
	switch strings.ToLower(u.Scheme) {
	case "sqlite":
		result.Type = dbabstraction.TypeSqlite
		result.HostUrl = strings.TrimPrefix(dbUrl, "sqlite://")
		if mustExist && !helper.FileExists(result.HostUrl) {
			return models.DbConnection{}, fmt.Errorf("file %s does not exist\n", result.HostUrl)
		}
	case "redis":
		result.Type = dbabstraction.TypeRedis
		result.HostUrl = u.Host
	default:
		return models.DbConnection{}, fmt.Errorf("unsupported database type: %s\n", dbUrl)
	}

	query := u.Query()

	result.Username = u.User.Username()
	result.Password, _ = u.User.Password()
	result.RedisUseSsl = query.Has("ssl")
	result.RedisPrefix = query.Get("prefix")

	return result, nil
}

// Migrate copies a database to a new location
func Migrate(configOld, configNew models.DbConnection) {
	dbOld, err := dbabstraction.GetNew(configOld)
	helper.Check(err)
	dbNew, err := dbabstraction.GetNew(configNew)
	helper.Check(err)

	apiKeys := dbOld.GetAllApiKeys()
	for _, apiKey := range apiKeys {
		dbNew.SaveApiKey(apiKey)
	}
	users := dbOld.GetAllUsers()
	for _, user := range users {
		dbNew.SaveUser(user, false)
		dbNew.SaveEnd2EndInfo(dbOld.GetEnd2EndInfo(user.Id), user.Id)
	}
	files := dbOld.GetAllMetadata()
	for _, file := range files {
		dbNew.SaveMetaData(file)
		if file.HotlinkId != "" {
			dbNew.SaveHotlink(file)
		}
	}
	dbOld.Close()
	dbNew.Close()
}

// RunGarbageCollection runs the databases GC
func RunGarbageCollection() {
	db.RunGarbageCollection()
}

// Upgrade migrates the DB to a new Gokapi version, if required
func Upgrade() {
	dbVersion := db.GetDbVersion()
	expectedVersion := db.GetSchemaVersion()
	if dbVersion < expectedVersion {
		db.Upgrade(dbVersion)
		db.SetDbVersion(expectedVersion)
		fmt.Printf("Successfully upgraded database to version %d\n", expectedVersion)
	}
}

// Close the database connection
func Close() {
	db.Close()
}

// Api Key Section

// GetAllApiKeys returns a map with all API keys
func GetAllApiKeys() map[string]models.ApiKey {
	return db.GetAllApiKeys()
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func GetApiKey(id string) (models.ApiKey, bool) {
	return db.GetApiKey(id)
}

// SaveApiKey saves the API key to the database
func SaveApiKey(apikey models.ApiKey) {
	db.SaveApiKey(apikey)
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func UpdateTimeApiKey(apikey models.ApiKey) {
	db.UpdateTimeApiKey(apikey)
}

// DeleteApiKey deletes an API key with the given ID
func DeleteApiKey(id string) {
	db.DeleteApiKey(id)
}

// GetSystemKey returns the latest UI API key
func GetSystemKey(userId int) (models.ApiKey, bool) {
	return db.GetSystemKey(userId)
}

// GetApiKeyByPublicKey returns an API key by using the public key
func GetApiKeyByPublicKey(publicKey string) (string, bool) {
	return db.GetApiKeyByPublicKey(publicKey)
}

// E2E Section

// SaveEnd2EndInfo stores the encrypted e2e info
func SaveEnd2EndInfo(info models.E2EInfoEncrypted, userId int) {
	info.AvailableFiles = nil
	db.SaveEnd2EndInfo(info, userId)
}

// GetEnd2EndInfo retrieves the encrypted e2e info
func GetEnd2EndInfo(userId int) models.E2EInfoEncrypted {
	info := db.GetEnd2EndInfo(userId)
	info.AvailableFiles = GetAllMetaDataIds()
	return info
}

// DeleteEnd2EndInfo resets the encrypted e2e info
func DeleteEnd2EndInfo(userId int) {
	db.DeleteEnd2EndInfo(userId)
}

// Hotlink Section

// GetHotlink returns the id of the file associated or false if not found
func GetHotlink(id string) (string, bool) {
	return db.GetHotlink(id)
}

// GetAllHotlinks returns an array with all hotlink ids
func GetAllHotlinks() []string {
	return db.GetAllHotlinks()
}

// SaveHotlink stores the hotlink associated with the file in the database
func SaveHotlink(file models.File) {
	db.SaveHotlink(file)
}

// DeleteHotlink deletes a hotlink with the given hotlink ID
func DeleteHotlink(id string) {
	db.DeleteHotlink(id)
}

// Metadata Section

// GetAllMetadata returns a map of all available files
func GetAllMetadata() map[string]models.File {
	return db.GetAllMetadata()
}

// GetAllMetaDataIds returns all Ids that contain metadata
func GetAllMetaDataIds() []string {
	return db.GetAllMetaDataIds()
}

// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
func GetMetaDataById(id string) (models.File, bool) {
	return db.GetMetaDataById(id)
}

// SaveMetaData stores the metadata of a file to the disk
func SaveMetaData(file models.File) {
	db.SaveMetaData(file)
}

// DeleteMetaData deletes information about a file
func DeleteMetaData(id string) {
	db.DeleteMetaData(id)
}

// IncreaseDownloadCount increases the download count of a file, preventing race conditions
func IncreaseDownloadCount(id string, decreaseRemainingDownloads bool) {
	db.IncreaseDownloadCount(id, decreaseRemainingDownloads)
}

// Session Section

// GetSession returns the session with the given ID or false if not a valid ID
func GetSession(id string) (models.Session, bool) {
	return db.GetSession(id)
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func SaveSession(id string, session models.Session) {
	db.SaveSession(id, session)
}

// DeleteSession deletes a session with the given ID
func DeleteSession(id string) {
	db.DeleteSession(id)
}

// DeleteAllSessions logs all users out
func DeleteAllSessions() {
	db.DeleteAllSessions()
}

// DeleteAllSessionsByUser logs the specific users out
func DeleteAllSessionsByUser(userId int) {
	db.DeleteAllSessionsByUser(userId)
}

// User Section

// GetAllUsers returns a map with all users
func GetAllUsers() []models.User {
	return db.GetAllUsers()
}

// GetUser returns a models.User if valid or false if the ID is not valid
func GetUser(id int) (models.User, bool) {
	return db.GetUser(id)
}

// GetUserByName returns a models.User if valid or false if the email is not valid
func GetUserByName(username string) (models.User, bool) {
	username = strings.ToLower(username)
	return db.GetUserByName(username)
}

// SaveUser saves a user to the database. If isNewUser is true, a new Id will be generated
func SaveUser(user models.User, isNewUser bool) {
	if user.Name == "" {
		panic("username cannot be empty")
	}
	user.Name = strings.ToLower(user.Name)
	db.SaveUser(user, isNewUser)
}

// UpdateUserLastOnline writes the last online time to the database
func UpdateUserLastOnline(id int) {
	db.UpdateUserLastOnline(id)
}

// DeleteUser deletes a user with the given ID
func DeleteUser(id int) {
	db.DeleteUser(id)
}

// GetSuperAdmin returns the models.User data for the super admin
func GetSuperAdmin() (models.User, bool) {
	users := db.GetAllUsers()
	for _, user := range users {
		if user.UserLevel == models.UserLevelSuperAdmin {
			return user, true
		}
	}
	return models.User{}, false
}

// EditSuperAdmin changes parameters of the super admin. If no user exists, a new superadmin will be created
// Returns an error if at least one user exists, but no superadmin
func EditSuperAdmin(username, passwordHash string) error {
	user, ok := GetSuperAdmin()
	if !ok {
		if len(GetAllUsers()) != 0 {
			return errors.New("at least one user exists, but no superadmin found")
		}
		newAdmin := models.User{
			Name:        username,
			Permissions: models.UserPermissionAll,
			UserLevel:   models.UserLevelSuperAdmin,
			Password:    passwordHash,
		}
		db.SaveUser(newAdmin, true)
		return nil
	}
	if username != "" {
		user.Name = username
	}
	if passwordHash != "" {
		user.Password = passwordHash
	}
	db.SaveUser(user, false)
	return nil
}
