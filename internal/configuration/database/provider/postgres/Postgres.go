package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	// Required for the postgres driver
	_ "github.com/jackc/pgx/v5/stdlib"
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

// GetType returns 3, for being a PostgreSQL interface
func (p DatabaseProvider) GetType() int {
	return 3 // dbabstraction.TypePostgres
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentDbVersion int) {
	// No upgrade steps yet - this provider was introduced at schema version 1
}

// GetDbVersion gets the version number of the database
func (p DatabaseProvider) GetDbVersion() int {
	var version int
	row := p.sqlDb.QueryRow("SELECT version FROM schemaversion LIMIT 1")
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
	_, err := p.sqlDb.Exec("DELETE FROM schemaversion")
	helper.Check(err)
	_, err = p.sqlDb.Exec("INSERT INTO schemaversion (version) VALUES ($1)", newVersion)
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
		dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbConfig.Username, dbConfig.Password, dbConfig.HostUrl, dbConfig.DatabaseName)
		var err error
		p.sqlDb, err = sql.Open("pgx", dsn)
		if err != nil {
			return DatabaseProvider{}, err
		}
		p.sqlDb.SetMaxOpenConns(10)
		p.sqlDb.SetMaxIdleConns(10)

		err = p.sqlDb.Ping()
		if err != nil {
			return DatabaseProvider{}, err
		}

		exists, err := p.tableExists("filemetadata")
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
	row := p.sqlDb.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1", tableName)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p DatabaseProvider) createNewDatabase() error {
	statements := []string{
		`CREATE TABLE apikeys (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			friendlyname VARCHAR(255) NOT NULL,
			lastused BIGINT NOT NULL,
			permissions INT NOT NULL DEFAULT 0,
			expiry BIGINT NOT NULL DEFAULT 0,
			issystemkey SMALLINT NOT NULL DEFAULT 0,
			userid INT NOT NULL,
			publicid VARCHAR(64) NOT NULL UNIQUE,
			uploadrequestid VARCHAR(64) NOT NULL
		)`,
		`CREATE TABLE e2econfig (
			id SERIAL PRIMARY KEY,
			config BYTEA NOT NULL,
			userid INT NOT NULL UNIQUE
		)`,
		`CREATE TABLE filemetadata (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			name VARCHAR(1024) NOT NULL,
			size VARCHAR(64) NOT NULL,
			sha1 VARCHAR(64) NOT NULL,
			expireat BIGINT NOT NULL,
			sizebytes BIGINT NOT NULL,
			downloadsremaining INT NOT NULL,
			downloadcount INT NOT NULL,
			passwordhash VARCHAR(255) NOT NULL,
			hotlinkid VARCHAR(255) NOT NULL,
			contenttype VARCHAR(255) NOT NULL,
			awsbucket VARCHAR(255) NOT NULL,
			encryption BYTEA NOT NULL,
			unlimiteddownloads SMALLINT NOT NULL,
			unlimitedtime SMALLINT NOT NULL,
			userid INT NOT NULL,
			uploaddate BIGINT NOT NULL,
			pendingdeletion BIGINT NOT NULL,
			uploadrequestid VARCHAR(64) NOT NULL
		)`,
		`CREATE TABLE hotlinks (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			fileid VARCHAR(64) NOT NULL UNIQUE
		)`,
		`CREATE TABLE sessions (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			renewat BIGINT NOT NULL,
			validuntil BIGINT NOT NULL,
			userid INT NOT NULL
		)`,
		`CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(255),
			permissions INT NOT NULL,
			userlevel INT NOT NULL,
			lastonline BIGINT NOT NULL DEFAULT 0,
			resetpassword SMALLINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE uploadrequests (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			userid INT NOT NULL,
			expiry BIGINT NOT NULL,
			maxfiles INT NOT NULL,
			maxsize INT NOT NULL,
			creation BIGINT NOT NULL,
			apikey VARCHAR(255) NOT NULL UNIQUE,
			note TEXT NOT NULL
		)`,
		`CREATE TABLE statistics (
			id SERIAL PRIMARY KEY,
			type INT NOT NULL UNIQUE,
			value BIGINT
		)`,
		`CREATE TABLE schemaversion (
			version INT NOT NULL
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
