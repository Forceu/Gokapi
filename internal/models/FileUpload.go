package models

type UploadRequest struct {
	AllowedDownloads int
	Expiry           int
	ExpiryTimestamp int64
	Password         string
	ExternalUrl      string
}
