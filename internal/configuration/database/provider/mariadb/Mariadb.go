package mariadb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	// Required for the mariadb/mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// DatabaseProvider contains the database instance
type DatabaseProvider struct {
	sqlDb *sql.DB
}

// DatabaseSchemeVersion contains the version number to be expected from the current database. If lower, an upgrade will be performed
const DatabaseSchemeVersion = 1

// New returns an instance
func New(dbConfig models.DbConnection) (DatabaseProvider, error) {
	return DatabaseProvider{}.init(dbConfig)
}

// GetType returns 2, for being a MariaDB/MySQL interface
func (p DatabaseProvider) GetType() int {
	return 2 // dbabstraction.TypeMariaDb
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentDbVersion int) {
	// No upgrade steps yet - this provider was introduced at schema version 1
}

// GetDbVersion gets the version number of the database
func (p DatabaseProvider) GetDbVersion() int {
	var version int
	row := p.sqlDb.QueryRow("SELECT Version FROM SchemaVersion LIMIT 1")
	err := row.Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		helper.Check(err)
	}
	return version
}

// SetDbVersion sets the version number of the database
func (p DatabaseProvider) SetDbVersion(newVersion int) {
	_, err := p.sqlDb.Exec("DELETE FROM SchemaVersion")
	helper.Check(err)
	_, err = p.sqlDb.Exec("INSERT INTO SchemaVersion (Version) VALUES (?)", newVersion)
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
	if dbConfig.DatabaseName == "" {
		return DatabaseProvider{}, errors.New("no database name was provided")
	}
	if p.sqlDb == nil {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=false", dbConfig.Username, dbConfig.Password, dbConfig.HostUrl, dbConfig.DatabaseName)
		var err error
		p.sqlDb, err = sql.Open("mysql", dsn)
		if err != nil {
			return DatabaseProvider{}, err
		}
		p.sqlDb.SetMaxOpenConns(10)
		p.sqlDb.SetMaxIdleConns(10)

		err = p.sqlDb.Ping()
		if err != nil {
			return DatabaseProvider{}, err
		}

		exists, err := p.tableExists("FileMetaData")
		if err != nil {
			return DatabaseProvider{}, err
		}
		if !exists {
			return p, p.createNewDatabase()
		}
		return p, nil
	}
	return p, nil
}

// Close the database connection
func (p DatabaseProvider) Close() {
	if p.sqlDb != nil {
		err := p.sqlDb.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	p.sqlDb = nil
}

// RunGarbageCollection runs the databases GC
func (p DatabaseProvider) RunGarbageCollection() {
	p.cleanExpiredSessions()
	p.cleanApiKeys()
}

func (p DatabaseProvider) tableExists(tableName string) (bool, error) {
	var count int
	row := p.sqlDb.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p DatabaseProvider) createNewDatabase() error {
	statements := []string{
		`CREATE TABLE ApiKeys (
			Id VARCHAR(64) NOT NULL,
			FriendlyName VARCHAR(255) NOT NULL,
			LastUsed BIGINT NOT NULL,
			Permissions INT NOT NULL DEFAULT 0,
			Expiry BIGINT NOT NULL DEFAULT 0,
			IsSystemKey TINYINT NOT NULL DEFAULT 0,
			UserId INT NOT NULL,
			PublicId VARCHAR(64) NOT NULL,
			UploadRequestId VARCHAR(64) NOT NULL,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_apikeys_publicid (PublicId)
		)`,
		`CREATE TABLE E2EConfig (
			Id INT NOT NULL AUTO_INCREMENT,
			Config MEDIUMBLOB NOT NULL,
			UserId INT NOT NULL,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_e2econfig_userid (UserId)
		)`,
		`CREATE TABLE FileMetaData (
			Id VARCHAR(64) NOT NULL,
			Name VARCHAR(1024) NOT NULL,
			Size VARCHAR(64) NOT NULL,
			SHA1 VARCHAR(64) NOT NULL,
			ExpireAt BIGINT NOT NULL,
			SizeBytes BIGINT NOT NULL,
			DownloadsRemaining INT NOT NULL,
			DownloadCount INT NOT NULL,
			PasswordHash VARCHAR(255) NOT NULL,
			HotlinkId VARCHAR(255) NOT NULL,
			ContentType VARCHAR(255) NOT NULL,
			AwsBucket VARCHAR(255) NOT NULL,
			Encryption MEDIUMBLOB NOT NULL,
			UnlimitedDownloads TINYINT NOT NULL,
			UnlimitedTime TINYINT NOT NULL,
			UserId INT NOT NULL,
			UploadDate BIGINT NOT NULL,
			PendingDeletion BIGINT NOT NULL,
			UploadRequestId VARCHAR(64) NOT NULL,
			PRIMARY KEY (Id)
		)`,
		`CREATE TABLE Hotlinks (
			Id VARCHAR(255) NOT NULL,
			FileId VARCHAR(64) NOT NULL,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_hotlinks_fileid (FileId)
		)`,
		`CREATE TABLE Sessions (
			Id VARCHAR(255) NOT NULL,
			RenewAt BIGINT NOT NULL,
			ValidUntil BIGINT NOT NULL,
			UserId INT NOT NULL,
			PRIMARY KEY (Id)
		)`,
		`CREATE TABLE Users (
			Id INT NOT NULL AUTO_INCREMENT,
			Name VARCHAR(255) NOT NULL,
			Password VARCHAR(255),
			Permissions INT NOT NULL,
			Userlevel INT NOT NULL,
			LastOnline BIGINT NOT NULL DEFAULT 0,
			ResetPassword TINYINT NOT NULL DEFAULT 0,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_users_name (Name)
		)`,
		`CREATE TABLE UploadRequests (
			Id VARCHAR(64) NOT NULL,
			Name VARCHAR(255) NOT NULL,
			UserId INT NOT NULL,
			Expiry BIGINT NOT NULL,
			MaxFiles INT NOT NULL,
			MaxSize INT NOT NULL,
			Creation BIGINT NOT NULL,
			ApiKey VARCHAR(255) NOT NULL,
			Note TEXT NOT NULL,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_uploadrequests_apikey (ApiKey)
		)`,
		`CREATE TABLE Statistics (
			Id INT NOT NULL AUTO_INCREMENT,
			Type INT NOT NULL,
			Value BIGINT,
			PRIMARY KEY (Id),
			UNIQUE KEY idx_statistics_type (Type)
		)`,
		`CREATE TABLE SchemaVersion (
			Version INT NOT NULL
		)`,
	}
	for _, statement := range statements {
		_, err := p.sqlDb.Exec(statement)
		if err != nil {
			return err
		}
	}
	p.SetDbVersion(DatabaseSchemeVersion)
	return nil
}
