package models

// UploadRequest is used to set an upload request
type UploadRequest struct {
	AllowedDownloads    int
	Expiry              int
	ExpiryTimestamp     int64
	Password            string
	ExternalUrl         string
	MaxMemory           int
	UnlimitedDownload   bool
	UnlimitedTime       bool
	IsEndToEndEncrypted bool
	RealSize            int64
}
