package models

// UploadRequest is used to set an upload request
type UploadRequest struct {
	UserId              int
	AllowedDownloads    int
	Expiry              int
	Password            string
	ExternalUrl         string
	MaxMemory           int
	UnlimitedDownload   bool
	UnlimitedTime       bool
	IsEndToEndEncrypted bool
	ExpiryTimestamp     int64
	RealSize            int64
}
