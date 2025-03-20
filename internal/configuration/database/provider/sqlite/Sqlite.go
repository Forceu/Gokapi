package sqlite

import (
	"bytes"
	"database/sql"
	"encoding/gob"
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
const DatabaseSchemeVersion = 8

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
	// < v1.9.6
	if currentDbVersion < 6 {
		fmt.Println("Please update to v1.9.6 before upgrading to 2.0.0")
	}
	// < v2.0.0-beta
	if currentDbVersion < 7 {
		legacyE2E := getLegacyE2EConfig(p)

		err := p.rawSqlite(`ALTER TABLE "ApiKeys" ADD COLUMN UserId INTEGER NOT NULL DEFAULT 0;
									 ALTER TABLE "ApiKeys" ADD COLUMN PublicId TEXT NOT NULL DEFAULT '';`)
		helper.Check(err)
		err = p.rawSqlite(`DELETE FROM "ApiKeys" WHERE IsSystemKey = 1`)
		helper.Check(err)
		err = p.rawSqlite(`ALTER TABLE "E2EConfig" ADD COLUMN UserId INTEGER NOT NULL DEFAULT 0;`)
		helper.Check(err)
		err = p.rawSqlite(`ALTER TABLE "FileMetaData" ADD COLUMN UserId INTEGER NOT NULL DEFAULT 0;`)
		helper.Check(err)
		err = p.rawSqlite(`DROP TABLE "Sessions"; CREATE TABLE "Sessions" (
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
		DROP TABLE "E2EConfig"; CREATE TABLE "E2EConfig" (
			"id"	INTEGER NOT NULL UNIQUE,
			"Config"	BLOB NOT NULL,
			"UserId" INTEGER NOT NULL UNIQUE ,
			PRIMARY KEY("id" AUTOINCREMENT)
		);
	    DROP TABLE IF EXISTS "UploadConfig";`)
		helper.Check(err)

		if legacyE2E.Version != 0 {
			p.SaveEnd2EndInfo(legacyE2E, 0)
		}
	}
	// < v2.0.0-beta2
	if currentDbVersion < 8 {
		keys := p.GetAllApiKeys()
		for _, key := range keys {
			if key.IsSystemKey {
				p.DeleteApiKey(key.Id)
			}
		}
	}
}

func getLegacyE2EConfig(p DatabaseProvider) models.E2EInfoEncrypted {
	result := models.E2EInfoEncrypted{}
	rowResult := schemaE2EConfig{}

	row := p.sqliteDb.QueryRow("SELECT Config FROM E2EConfig WHERE id = 1")
	err := row.Scan(&rowResult.Config)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result
		}
		helper.Check(err)
		return result
	}

	buf := bytes.NewBuffer(rowResult.Config)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&result)
	helper.Check(err)
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
			"UserId" INTEGER NOT NULL,
			"PublicId" TEXT NOT NULL UNIQUE ,
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
			"UserId"	INTEGER NOT NULL,
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
