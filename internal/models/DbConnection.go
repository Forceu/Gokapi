package models

type DbConnection struct {
	SqliteDataDir  string
	SqliteFileName string
	RedisUrl       string
	RedisPrefix    string
	RedisUsername  string
	RedisPassword  string
	RedisUseSsl    bool
	Type           int
}
