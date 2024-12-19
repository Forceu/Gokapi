package models

type User struct {
	Id          int    `json:"id" redis:"id""`
	Name        string `json:"name" redis:"Name"`
	Email       string `json:"email" redis:"Email"`
	Permissions uint8  `json:"permissions" redis:"Permissions"`
	UserLevel   uint8  `json:"userLevel" redis:"UserLevel"`
	LastOnline  int64  `json:"lastOnline" redis:"LastOnline"`
	Password    string `redis:"Password"`
}

const UserLevelSuperAdmin = 0
const UserLevelAdmin = 1
const UserLevelUser = 2
