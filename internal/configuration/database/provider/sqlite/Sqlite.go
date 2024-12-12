package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
	"path/filepath"
	// Required for sqlite driver
	_ "modernc.org/sqlite"
)

// DatabaseProvider contains the database instance
type DatabaseProvider struct {
	sqliteDb *sql.DB
}

// DatabaseSchemeVersion contains the version number to be expected from the current database. If lower, an upgrade will be performed
const DatabaseSchemeVersion = 6

// New returns an instance
func New(dbConfig models.DbConnection) (DatabaseProvider, error) {
	return DatabaseProvider{}.init(dbConfig)
}

// GetType returns 0, for being a Sqlite interface
func (p DatabaseProvider) GetType() int {
	return 0 // dbabstraction.Sqlite
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentDbVersion int) {
	// < v1.9.0
	if currentDbVersion < 2 {
		// Remove Column LastUpdate, deleting old data
		err := p.rawSqlite(`DROP TABLE UploadStatus; CREATE TABLE "UploadStatus" (
			"ChunkId"	TEXT NOT NULL UNIQUE,
			"CurrentStatus"	INTEGER NOT NULL,
			"CreationDate"	INTEGER NOT NULL,
			PRIMARY KEY("ChunkId")
		) WITHOUT ROWID;`)
		helper.Check(err)
	}
	// < v1.9.4
	// there might be an instance where the LastUsedString column still exists. This reads all
	// API keys, drops the table and then recreates it.
	if currentDbVersion < 5 {
		// Add Column LastUsedString, keeping old data
		apiKeys := p.getLegacyApiKeys()
		err := p.rawSqlite(`DROP TABLE "ApiKeys";
				CREATE TABLE "ApiKeys" (
				    "Id"	TEXT NOT NULL UNIQUE,
				    "FriendlyName"	TEXT NOT NULL,
				    "LastUsed"	INTEGER NOT NULL,
				    "Permissions"	INTEGER NOT NULL DEFAULT 0,
				    "Expiry"	INTEGER,
				    "IsSystemKey"	INTEGER,
				     PRIMARY KEY("Id")
				) WITHOUT ROWID;`)
		helper.Check(err)
		for _, apiKey := range apiKeys {
			p.SaveApiKey(apiKey)
		}
	}
	// < v1.9.6
	if currentDbVersion < 6 {
		// Add Column LastUsedString, keeping old data
		err := p.rawSqlite(`ALTER TABLE "UploadStatus"  ADD COLUMN "FileId" text;`)
		helper.Check(err)
	}
}

type legacySchemaApiKeys struct {
	Id           string
	FriendlyName string
	LastUsed     int64
	Permissions  int
}

// getLegacyApiKeys returns a map with all API keys with the legacy scheme
func (p DatabaseProvider) getLegacyApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)

	rows, err := p.sqliteDb.Query("SELECT Id,FriendlyName,LastUsed,Permissions FROM ApiKeys")
	helper.Check(err)
	defer rows.Close()
	for rows.Next() {
		rowData := legacySchemaApiKeys{}
		err = rows.Scan(&rowData.Id, &rowData.FriendlyName, &rowData.LastUsed, &rowData.Permissions)
		helper.Check(err)
		result[rowData.Id] = models.ApiKey{
			Id:           rowData.Id,
			FriendlyName: rowData.FriendlyName,
			LastUsed:     rowData.LastUsed,
			Permissions:  uint8(rowData.Permissions),
		}
	}
	return result
}

// GetDbVersion gets the version number of the database
func (p DatabaseProvider) GetDbVersion() int {
	var userVersion int
	row := p.sqliteDb.QueryRow("PRAGMA user_version;")
	err := row.Scan(&userVersion)
	helper.Check(err)
	return userVersion
}

// SetDbVersion sets the version number of the database
func (p DatabaseProvider) SetDbVersion(newVersion int) {
	_, err := p.sqliteDb.Exec(fmt.Sprintf("PRAGMA user_version = %d;", newVersion))
	helper.Check(err)
}

// GetSchemaVersion returns the version number, that the database should be if fully upgraded
func (p DatabaseProvider) GetSchemaVersion() int {
	return DatabaseSchemeVersion
}

// Init connects to the database and creates the table structure, if necessary
func (p DatabaseProvider) init(dbConfig models.DbConnection) (DatabaseProvider, error) {
	if dbConfig.HostUrl == "" {
		return DatabaseProvider{}, errors.New("empty database url was provided")
	}
	if p.sqliteDb == nil {
		cleanPath := filepath.Clean(dbConfig.HostUrl)
		dataDir := filepath.Dir(cleanPath)
		var err error
		if !helper.FolderExists(dataDir) {
			err = os.MkdirAll(dataDir, 0700)
			if err != nil {
				return DatabaseProvider{}, err
			}
		}
		p.sqliteDb, err = sql.Open("sqlite", cleanPath+"?_pragma=busy_timeout=10000&_pragma=journal_mode=WAL")
		if err != nil {
			return DatabaseProvider{}, err
		}
		p.sqliteDb.SetMaxOpenConns(10000)
		p.sqliteDb.SetMaxIdleConns(10000)

		if !helper.FileExists(dbConfig.HostUrl) {
			return p, p.createNewDatabase()
		}
		err = p.sqliteDb.Ping()
		return p, err
	}
	return p, nil
}

// Close the database connection
func (p DatabaseProvider) Close() {
	if p.sqliteDb != nil {
		err := p.sqliteDb.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	p.sqliteDb = nil
}

// RunGarbageCollection runs the databases GC
func (p DatabaseProvider) RunGarbageCollection() {
	p.cleanExpiredSessions()
	p.cleanUploadStatus()
	p.cleanApiKeys()
}

func (p DatabaseProvider) createNewDatabase() error {
	sqlStmt := `CREATE TABLE "ApiKeys" (
			"Id"	TEXT NOT NULL UNIQUE,
			"FriendlyName"	TEXT NOT NULL,
			"LastUsed"	INTEGER NOT NULL,
			"Permissions"	INTEGER NOT NULL DEFAULT 0,
			"Expiry"	INTEGER,
			"IsSystemKey"	INTEGER,
			PRIMARY KEY("Id")
		) WITHOUT ROWID;
		CREATE TABLE "E2EConfig" (
			"id"	INTEGER NOT NULL UNIQUE,
			"Config"	BLOB NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT)
		);
		CREATE TABLE "FileMetaData" (
			"Id"	TEXT NOT NULL UNIQUE,
			"Name"	TEXT NOT NULL,
			"Size"	TEXT NOT NULL,
			"SHA1"	TEXT NOT NULL,
			"ExpireAt"	INTEGER NOT NULL,
			"SizeBytes"	INTEGER NOT NULL,
			"ExpireAtString"	TEXT NOT NULL,
			"DownloadsRemaining"	INTEGER NOT NULL,
			"DownloadCount"	INTEGER NOT NULL,
			"PasswordHash"	TEXT NOT NULL,
			"HotlinkId"	TEXT NOT NULL,
			"ContentType"	TEXT NOT NULL,
			"AwsBucket"	TEXT NOT NULL,
			"Encryption"	BLOB NOT NULL,
			"UnlimitedDownloads"	INTEGER NOT NULL,
			"UnlimitedTime"	INTEGER NOT NULL,
			PRIMARY KEY("Id")
		);
		CREATE TABLE "Hotlinks" (
			"Id"	TEXT NOT NULL UNIQUE,
			"FileId"	TEXT NOT NULL UNIQUE,
			PRIMARY KEY("Id")
		) WITHOUT ROWID;
		CREATE TABLE "Sessions" (
			"Id"	TEXT NOT NULL UNIQUE,
			"RenewAt"	INTEGER NOT NULL,
			"ValidUntil"	INTEGER NOT NULL,
			PRIMARY KEY("Id")
		) WITHOUT ROWID;
		CREATE TABLE "UploadStatus" (
			"ChunkId"	TEXT NOT NULL UNIQUE,
			"CurrentStatus"	INTEGER NOT NULL,
			"CreationDate"	INTEGER NOT NULL,
			PRIMARY KEY("ChunkId")
		) WITHOUT ROWID;
`
	err := p.rawSqlite(sqlStmt)
	if err != nil {
		return err
	}
	p.SetDbVersion(DatabaseSchemeVersion)
	return nil
}

// rawSqlite runs a raw SQL statement. Should only be used for upgrading
func (p DatabaseProvider) rawSqlite(statement string) error {
	if p.sqliteDb == nil {
		panic("Sqlite not initialised")
	}
	_, err := p.sqliteDb.Exec(statement)
	return err
}
