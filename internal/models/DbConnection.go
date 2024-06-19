package models

type DbConnection struct {
	SqliteDataDir  string
	SqliteFileName string
	Type           int
}
