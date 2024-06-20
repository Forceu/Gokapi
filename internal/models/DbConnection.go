package models

type DbConnection struct {
	SqliteDataDir  string
	SqliteFileName string
	RedisUrl       string
	RedisPrefix    string
	Type           int
}
