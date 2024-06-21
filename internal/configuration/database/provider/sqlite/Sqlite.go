package sqlite

import (
	"database/sql"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
	"path/filepath"
	// Required for sqlite driver
	_ "modernc.org/sqlite"
)

var sqliteDb *sql.DB

type DatabaseProvider struct {
}

// New returns an instance
func New() DatabaseProvider {
	return DatabaseProvider{}
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentVersion int) {
	// < v1.8.5
	if currentVersion < 20 {
		err := rawSqlite(`DROP TABLE UploadStatus; CREATE TABLE "UploadStatus" (
			"ChunkId"	TEXT NOT NULL UNIQUE,
			"CurrentStatus"	INTEGER NOT NULL,
			"CreationDate"	INTEGER NOT NULL,
			PRIMARY KEY("ChunkId")
		) WITHOUT ROWID;`)
		helper.Check(err)
	}
}

// Init connects to the database and creates the table structure, if necessary
func (p DatabaseProvider) Init(dbConfig models.DbConnection) error {
	if sqliteDb == nil {
		dataDir := filepath.Clean(dbConfig.SqliteDataDir)
		var err error
		if !helper.FolderExists(dataDir) {
			err = os.MkdirAll(dataDir, 0700)
			if err != nil {
				return err
			}
		}
		dbFullPath := dataDir + "/" + dbConfig.SqliteFileName
		sqliteDb, err = sql.Open("sqlite", dbFullPath+"?_pragma=busy_timeout=10000&_pragma=journal_mode=WAL")
		if err != nil {
			return err
		}
		sqliteDb.SetMaxOpenConns(10000)
		sqliteDb.SetMaxIdleConns(10000)

		if !helper.FileExists(dbFullPath) {
			return createNewDatabase()
		}
	}
	return nil
}

// Close the database connection
func (p DatabaseProvider) Close() {
	if sqliteDb != nil {
		err := sqliteDb.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	sqliteDb = nil
}

// RunGarbageCollection runs the databases GC
func (p DatabaseProvider) RunGarbageCollection() {
	p.cleanExpiredSessions()
	p.cleanUploadStatus()
}

type schemaPragma struct {
	Cid        string
	Name       string
	Type       string
	NotNull    int
	DefaultVal sql.NullString
	Pk         int
}

// ColumnExists returns true if a column with the name columnName exists in table tableName
// Should only be used for upgrading
func ColumnExists(tableName, columnName string) (bool, error) {
	if sqliteDb == nil {
		panic("Sqlite not initialised")
	}
	rows, err := sqliteDb.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var pragmaInfo schemaPragma
		err = rows.Scan(&pragmaInfo.Cid, &pragmaInfo.Name, &pragmaInfo.Type, &pragmaInfo.NotNull, &pragmaInfo.DefaultVal, &pragmaInfo.Pk)
		if err != nil {
			return false, err
		}
		if pragmaInfo.Name == columnName {
			return true, nil
		}
	}
	return false, nil
}

func createNewDatabase() error {
	sqlStmt := `
		CREATE TABLE "ApiKeys" (
			"Id"	TEXT NOT NULL UNIQUE,
			"FriendlyName"	TEXT NOT NULL,
			"LastUsed"	INTEGER NOT NULL,
			"LastUsedString"	TEXT NOT NULL,
			"Permissions"	INTEGER NOT NULL DEFAULT 0,
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
		CREATE TABLE "UploadConfig" (
			"id"	INTEGER NOT NULL UNIQUE,
			"Downloads"	INTEGER,
			"TimeExpiry"	INTEGER,
			"Password"	TEXT,
			"UnlimitedDownloads"	INTEGER,
			"UnlimitedTime"	INTEGER,
			PRIMARY KEY("id")
		);
		CREATE TABLE "UploadStatus" (
			"ChunkId"	TEXT NOT NULL UNIQUE,
			"CurrentStatus"	INTEGER NOT NULL,
			"CreationDate"	INTEGER NOT NULL,
			PRIMARY KEY("ChunkId")
		) WITHOUT ROWID;
`
	err := rawSqlite(sqlStmt)
	return err
}

// rawSqlite runs a raw SQL statement. Should only be used for upgrading
func rawSqlite(statement string) error {
	if sqliteDb == nil {
		panic("Sqlite not initialised")
	}
	_, err := sqliteDb.Exec(statement)
	return err
}
