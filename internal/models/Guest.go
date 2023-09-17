package models

// GuestToken contains data of a single guest token
type GuestToken struct {
	Id             string `json:"Id"`
	LastUsed       int64  `json:"LastUsed"`
	LastUsedString string `json:"LastUsedString"`
}
