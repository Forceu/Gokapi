package models

// UploadToken contains data of a single guest upload token.
// It is essntially a single-use API key for uploading one item.
type UploadToken struct {
	Id             string `json:"Id"`
	LastUsedString string `json:"LastUsedString"`
	LastUsed       int64  `json:"LastUsed"`
}
