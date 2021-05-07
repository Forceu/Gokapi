package models

// ApiKey contains data of a single api key
type ApiKey struct {
	Id             string `json:"Id"`
	FriendlyName   string `json:"FriendlyName"`
	LastUsed       int64  `json:"LastUsed"`
	LastUsedString string `json:"LastUsedString"`
}
