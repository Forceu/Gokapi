package models

// Presign is a struct used for generating presigned URLs
type Presign struct {
	Id       string
	FileIds  []string
	Expiry   int64
	Filename string
}
