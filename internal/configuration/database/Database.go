package database

import (
	"database/sql"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"log"
	// Required for sqlite driver
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

var sqliteDb *sql.DB

// Init creates the database files and connects to it
func Init(dataDir, dbName string) {
	if sqliteDb == nil {
		dataDir = filepath.Clean(dataDir)
		var err error
		if !helper.FolderExists(dataDir) {
			err = os.MkdirAll(dataDir, 0700)
			helper.Check(err)
		}
		dbFullPath := dataDir + "/" + dbName
		sqliteDb, err = sql.Open("sqlite", dbFullPath+"?_pragma=busy_timeout=10000&_pragma=journal_mode=WAL")
		if err != nil {
			log.Fatal(err)
		}
		sqliteDb.SetMaxOpenConns(10000)
		sqliteDb.SetMaxIdleConns(10000)

		if !helper.FileExists(dbFullPath) {
			createNewDatabase()
		}
	}
}

// Close the database connection
func Close() {
	if sqliteDb != nil {
		err := sqliteDb.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	sqliteDb = nil
}

// RunGarbageCollection runs the databases GC
func RunGarbageCollection() {
	cleanExpiredSessions()
	cleanUploadStatus()
}

// RawSqlite runs a raw SQL statement. Should only be used for upgrading
func RawSqlite(statement string) error {
	_, err := sqliteDb.Exec(statement)
	return err
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

func createNewDatabase() {
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
			"LastUpdate"	INTEGER NOT NULL,
			"CreationDate"	INTEGER NOT NULL,
			PRIMARY KEY("ChunkId")
		) WITHOUT ROWID;
`
	err := RawSqlite(sqlStmt)
	helper.Check(err)
}
