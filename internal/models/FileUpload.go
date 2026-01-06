package models

// UploadParameters is used to set parameters for a new upload
type UploadParameters struct {
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
	FileRequestId       string
}
