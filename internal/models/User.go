package models

type User struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Permissions int8   `json:"permissions"`
	UserLevel   int8   `json:"userLevel"`
	Password    string
}

const UserLevelSuperAdmin = 0
const UserLevelAdmin = 1
const UserLevelUser = 2
