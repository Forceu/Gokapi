package models

// DbConnection is a struct that contains the database configuration for connecting
type DbConnection struct {
	HostUrl     string
	RedisPrefix string
	Username    string
	Password    string
	RedisUseSsl bool
	Type        int
}
