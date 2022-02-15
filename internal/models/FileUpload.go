package models

// UploadRequest is used to set an upload request
type UploadRequest struct {
	AllowedDownloads  int
	Expiry            int
	ExpiryTimestamp   int64
	Password          string
	ExternalUrl       string
	MaxMemory         int
	DataDir           string
	UnlimitedDownload bool
	UnlimitedTime     bool
}
