package dbabstraction

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database/provider/redis"
	"github.com/forceu/gokapi/internal/configuration/database/provider/sqlite"
	"github.com/forceu/gokapi/internal/models"
)

const (
	// TypeSqlite specifies to use an SQLite database
	TypeSqlite = iota
	// TypeRedis specifies to use a Redis database
	TypeRedis
)

// Database declares the required functions for a database connection
type Database interface {
	// GetType returns identifier of the underlying interface
	GetType() int

	// Upgrade migrates the DB to a new Gokapi version, if required
	Upgrade(currentDbVersion int)
	// RunGarbageCollection runs the databases GC
	RunGarbageCollection()
	// Close the database connection
	Close()

	// GetDbVersion gets the version number of the database
	GetDbVersion() int
	// SetDbVersion sets the version number of the database
	SetDbVersion(newVersion int)
	// GetSchemaVersion returns the version number, that the database should be if fully upgraded
	GetSchemaVersion() int

	// GetAllApiKeys returns a map with all API keys
	GetAllApiKeys() map[string]models.ApiKey
	// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
	GetApiKey(id string) (models.ApiKey, bool)
	// SaveApiKey saves the API key to the database
	SaveApiKey(apikey models.ApiKey)
	// UpdateTimeApiKey writes the content of LastUsage to the database
	UpdateTimeApiKey(apikey models.ApiKey)
	// DeleteApiKey deletes an API key with the given ID
	DeleteApiKey(id string)
	// GetSystemKey returns the latest UI API key
	GetSystemKey(userId int) (models.ApiKey, bool)
	// GetApiKeyByPublicKey returns an API key by using the public key
	GetApiKeyByPublicKey(publicKey string) (string, bool)

	// SaveEnd2EndInfo stores the encrypted e2e info
	SaveEnd2EndInfo(info models.E2EInfoEncrypted, userId int)
	// GetEnd2EndInfo retrieves the encrypted e2e info
	GetEnd2EndInfo(userId int) models.E2EInfoEncrypted
	// DeleteEnd2EndInfo resets the encrypted e2e info
	DeleteEnd2EndInfo(userId int)

	// GetHotlink returns the id of the file associated or false if not found
	GetHotlink(id string) (string, bool)
	// GetAllHotlinks returns an array with all hotlink ids
	GetAllHotlinks() []string
	// SaveHotlink stores the hotlink associated with the file in the database
	SaveHotlink(file models.File)
	// DeleteHotlink deletes a hotlink with the given hotlink ID
	DeleteHotlink(id string)

	// GetAllMetadata returns a map of all available files
	GetAllMetadata() map[string]models.File
	// GetAllMetaDataIds returns all Ids that contain metadata
	GetAllMetaDataIds() []string
	// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
	GetMetaDataById(id string) (models.File, bool)
	// SaveMetaData stores the metadata of a file to the disk
	SaveMetaData(file models.File)
	// DeleteMetaData deletes information about a file
	DeleteMetaData(id string)
	// IncreaseDownloadCount increases the download count of a file, preventing race conditions
	IncreaseDownloadCount(id string, decreaseRemainingDownloads bool)

	// GetSession returns the session with the given ID or false if not a valid ID
	GetSession(id string) (models.Session, bool)
	// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
	SaveSession(id string, session models.Session)
	// DeleteSession deletes a session with the given ID
	DeleteSession(id string)
	// DeleteAllSessions logs all users out
	DeleteAllSessions()
	// DeleteAllSessionsByUser logs the specific users out
	DeleteAllSessionsByUser(userId int)

	// GetAllUsers returns a map with all users
	GetAllUsers() []models.User
	// GetUser returns a models.User if valid or false if the ID is not valid
	GetUser(id int) (models.User, bool)
	// GetUserByName returns a models.User if valid or false if the username is not valid
	GetUserByName(email string) (models.User, bool)
	// SaveUser saves a user to the database. If isNewUser is true, a new Id will be generated
	SaveUser(user models.User, isNewUser bool)
	// UpdateUserLastOnline writes the last online time to the database
	UpdateUserLastOnline(id int)
	// DeleteUser deletes a user with the given ID
	DeleteUser(id int)
}

// GetNew connects to the given database and initialises it
func GetNew(config models.DbConnection) (Database, error) {
	switch config.Type {
	case TypeSqlite:
		return sqlite.New(config)
	case TypeRedis:
		return redis.New(config)
	default:
		return nil, fmt.Errorf("unsupported database: type %v", config.Type)
	}
}
