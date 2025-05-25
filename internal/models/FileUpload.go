package models

// UploadRequest is used to set an upload request
type UploadRequest struct {
	UserId              int
	AllowedDownloads    int
	Expiry              int
	MaxMemory           int
	ExpiryTimestamp     int64
	RealSize            int64
	UnlimitedDownload   bool
	UnlimitedTime       bool
	IsEndToEndEncrypted bool
	Password            string
	ExternalUrl         string
}
