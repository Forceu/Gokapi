package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forceu/gokapi/internal/environment"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	// Required for the sqlite driver
	_ "modernc.org/sqlite"
)

// DatabaseProvider contains the database instance
type DatabaseProvider struct {
	sqliteDb *sql.DB
}

// DatabaseSchemeVersion contains the version number to be expected from the current database. If lower, an upgrade will be performed
const DatabaseSchemeVersion = 12

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
	// < v2.0.0
	if currentDbVersion < 10 {
		fmt.Println("Error: Gokapi runs >=v2.0.0, but Database is <v2.0.0")
		osExit(1)
		return
	}
	// pre local TZ
	if currentDbVersion < 11 {
		err := p.rawSqlite("ALTER TABLE FileMetaData DROP COLUMN ExpireAtString;")
		helper.Check(err)
	}
	// pre upload requests
	if currentDbVersion < 12 {

		err := p.rawSqlite(`ALTER TABLE FileMetaData ADD COLUMN "UploadRequestId" INTEGER NOT NULL DEFAULT 0;
									 ALTER TABLE ApiKeys ADD COLUMN "UploadRequestId" INTEGER NOT NULL DEFAULT 0;
									 CREATE TABLE "UploadRequests" (
										"id"	INTEGER NOT NULL UNIQUE,
										"name"	TEXT NOT NULL,
										"userid"	INTEGER NOT NULL,
										"expiry"	INTEGER NOT NULL,
										"maxFiles"	INTEGER NOT NULL,
										"maxSize"	INTEGER NOT NULL,
										"creation"	INTEGER NOT NULL,
										PRIMARY KEY("id" AUTOINCREMENT)
									 );
									CREATE TABLE "Presign" (
										"id"	TEXT NOT NULL UNIQUE,
										"fileIds"	TEXT NOT NULL,
										"expiry"	INTEGER NOT NULL,
										"filename"	TEXT NOT NULL,
										PRIMARY KEY("id")
									);`)
		helper.Check(err)
		if environment.New().PermRequestGrantedByDefault {
			for _, user := range p.GetAllUsers() {
				user.GrantPermission(models.UserPermGuestUploads)
				p.SaveUser(user, false)
			}
		}
		for _, apiKey := range p.GetAllApiKeys() {
			if apiKey.IsSystemKey {
				p.DeleteApiKey(apiKey.Id)
			}
		}
	}
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

// GetSchemaVersion returns the version number, which the database should be at if fully upgraded
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
		p.sqliteDb.SetMaxOpenConns(5)
		p.sqliteDb.SetMaxIdleConns(5)

		exists, err := helper.FileExists(dbConfig.HostUrl)
		helper.Check(err)
		if !exists {
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
	p.cleanApiKeys()
	p.cleanPresignedUrls()
}

func (p DatabaseProvider) createNewDatabase() error {
	sqlStmt := `CREATE TABLE "ApiKeys" (
			"Id"	TEXT NOT NULL UNIQUE,
			"FriendlyName"	TEXT NOT NULL,
			"LastUsed"	INTEGER NOT NULL,
			"Permissions"	INTEGER NOT NULL DEFAULT 0,
			"Expiry"	INTEGER,
			"IsSystemKey"	INTEGER,
			"UserId" INTEGER NOT NULL,
			"PublicId" TEXT NOT NULL UNIQUE ,
			"UploadRequestId"	INTEGER NOT NULL,
			PRIMARY KEY("Id")
		) WITHOUT ROWID;
		CREATE TABLE "E2EConfig" (
			"id"	INTEGER NOT NULL UNIQUE,
			"Config"	BLOB NOT NULL,
			"UserId" INTEGER NOT NULL UNIQUE,
			PRIMARY KEY("id" AUTOINCREMENT)
		);
		CREATE TABLE "FileMetaData" (
			"Id"	TEXT NOT NULL UNIQUE,
			"Name"	TEXT NOT NULL,
			"Size"	TEXT NOT NULL,
			"SHA1"	TEXT NOT NULL,
			"ExpireAt"	INTEGER NOT NULL,
			"SizeBytes"	INTEGER NOT NULL,
			"DownloadsRemaining"	INTEGER NOT NULL,
			"DownloadCount"	INTEGER NOT NULL,
			"PasswordHash"	TEXT NOT NULL,
			"HotlinkId"	TEXT NOT NULL,
			"ContentType"	TEXT NOT NULL,
			"AwsBucket"	TEXT NOT NULL,
			"Encryption"	BLOB NOT NULL,
			"UnlimitedDownloads"	INTEGER NOT NULL,
			"UnlimitedTime"	INTEGER NOT NULL,
			"UserId"	INTEGER NOT NULL,
			"UploadDate"	INTEGER NOT NULL,
			"PendingDeletion"	INTEGER NOT NULL,
			"UploadRequestId"	INTEGER NOT NULL,
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
			"UserId"	INTEGER NOT NULL,
			PRIMARY KEY("Id")
		) WITHOUT ROWID;
		CREATE TABLE "Users" (
			"Id"	INTEGER NOT NULL UNIQUE,
			"Name"	TEXT NOT NULL UNIQUE,
			"Password"	TEXT,
			"Permissions"	INTEGER NOT NULL,
			"Userlevel"	INTEGER NOT NULL,
			"LastOnline"	INTEGER NOT NULL DEFAULT 0,
			"ResetPassword"	INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY("Id" AUTOINCREMENT)
		);
		CREATE TABLE "UploadRequests" (
			"id"	INTEGER NOT NULL UNIQUE,
			"name"	TEXT,
			"userid"	INTEGER NOT NULL,
			"expiry"	INTEGER NOT NULL,
			"maxFiles"	INTEGER NOT NULL,
			"maxSize"	INTEGER NOT NULL,
			"creation"	INTEGER NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT)
		);
		CREATE TABLE "Presign" (
			"id"	TEXT NOT NULL UNIQUE,
			"fileIds"	TEXT NOT NULL,
			"expiry"	INTEGER NOT NULL,
			"filename"	TEXT NOT NULL,
			PRIMARY KEY("id")
		);
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

var osExit = os.Exit
