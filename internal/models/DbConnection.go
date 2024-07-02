package models

type DbConnection struct {
	HostUrl     string
	RedisPrefix string
	Username    string
	Password    string
	RedisUseSsl bool
	Type        int
}
