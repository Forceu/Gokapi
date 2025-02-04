package models

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	"net/url"
)

// File is a struct used for saving information about an uploaded file
type File struct {
	Id                      string         `json:"Id" redis:"Id"`                                 // The internal ID of the file
	Name                    string         `json:"Name" redis:"Name"`                             // The filename. Will be 'Encrypted file' for end-to-end encrypted files
	Size                    string         `json:"Size" redis:"Size"`                             // Filesize in a human-readable format
	SHA1                    string         `json:"SHA1" redis:"SHA1"`                             // The hash of the file, used for deduplication
	PasswordHash            string         `json:"PasswordHash" redis:"PasswordHash"`             // The hash of the password (if the file is password protected)
	HotlinkId               string         `json:"HotlinkId" redis:"HotlinkId"`                   // If file is a picture file and can be hotlinked, this is the ID for the hotlink
	ContentType             string         `json:"ContentType" redis:"ContentType"`               // The MIME type for the file
	AwsBucket               string         `json:"AwsBucket" redis:"AwsBucket"`                   // If the file is stored in the cloud, this is the bucket that is being used
	ExpireAtString          string         `json:"ExpireAtString" redis:"ExpireAtString"`         // Time expiry in a human-readable format in local time
	ExpireAt                int64          `json:"ExpireAt" redis:"ExpireAt"`                     // "UTC timestamp of file expiry
	SizeBytes               int64          `json:"SizeBytes" redis:"SizeBytes"`                   // Filesize in bytes
	DownloadsRemaining      int            `json:"DownloadsRemaining" redis:"DownloadsRemaining"` // The remaining downloads for this file
	DownloadCount           int            `json:"DownloadCount" redis:"DownloadCount"`           // The amount of times the file has been downloaded
	UserId                  int            `json:"UserId" redis:"UserId"`                         // The user ID of the uploader
	Encryption              EncryptionInfo `json:"Encryption" redis:"-"`                          // If the file is encrypted, this stores all info for decrypting
	UnlimitedDownloads      bool           `json:"UnlimitedDownloads" redis:"UnlimitedDownloads"` // True if the uploader did not limit the downloads
	UnlimitedTime           bool           `json:"UnlimitedTime" redis:"UnlimitedTime"`           // True if the uploader did not limit the time
	InternalRedisEncryption []byte         `redis:"EncryptionRedis"`                              // This field is an internal field, used to store the EncryptionInfo in a Redis Hashmap
}

// FileApiOutput will be displayed for public outputs from the ID, hiding sensitive information
type FileApiOutput struct {
	Id                           string `json:"Id"`                           // The internal ID of the file
	Name                         string `json:"Name"`                         // The filename. Will be 'Encrypted file' for end-to-end encrypted files
	Size                         string `json:"Size"`                         // Filesize in a human-readable format
	HotlinkId                    string `json:"HotlinkId"`                    // If file is a picture file and can be hotlinked, this is the ID for the hotlink
	ContentType                  string `json:"ContentType"`                  // The MIME type for the file
	ExpireAtString               string `json:"ExpireAtString"`               // Time expiry in a human-readable format in local time
	UrlDownload                  string `json:"UrlDownload"`                  // The public download URL for the file
	UrlHotlink                   string `json:"UrlHotlink"`                   // The public hotlink URL for the file
	ExpireAt                     int64  `json:"ExpireAt"`                     // "UTC timestamp of file expiry
	SizeBytes                    int64  `json:"SizeBytes"`                    // Filesize in bytes
	DownloadsRemaining           int    `json:"DownloadsRemaining"`           // The remaining downloads for this file
	DownloadCount                int    `json:"DownloadCount"`                // The amount of times the file has been downloaded
	UnlimitedDownloads           bool   `json:"UnlimitedDownloads"`           // True if the uploader did not limit the downloads
	UnlimitedTime                bool   `json:"UnlimitedTime"`                // True if the uploader did not limit the time
	RequiresClientSideDecryption bool   `json:"RequiresClientSideDecryption"` // True if the file has to be decrypted client-side
	IsEncrypted                  bool   `json:"IsEncrypted"`                  // True if the file is encrypted
	IsEndToEndEncrypted          bool   `json:"IsEndToEndEncrypted"`          // True if the file is end-to-end encrypted
	IsPasswordProtected          bool   `json:"IsPasswordProtected"`          // True if a password has to be entered before downloading the file
	IsSavedOnLocalStorage        bool   `json:"IsSavedOnLocalStorage"`        // True if the file does not use cloud storage
	UploaderId                   int    `json:"UploaderId"`                   // The user ID of the uploader
}

// EncryptionInfo holds information about the encryption used on the file
type EncryptionInfo struct {
	IsEncrypted         bool   `json:"IsEncrypted" redis:"IsEncrypted"`
	IsEndToEndEncrypted bool   `json:"IsEndToEndEncrypted" redis:"IsEndToEndEncrypted"`
	DecryptionKey       []byte `json:"DecryptionKey" redis:"DecryptionKey"`
	Nonce               []byte `json:"Nonce" redis:"Nonce"`
}

// IsLocalStorage returns true if the file is not stored on a remote storage
func (f *File) IsLocalStorage() bool {
	return f.AwsBucket == ""
}

// ToFileApiOutput returns a json object without sensitive information
func (f *File) ToFileApiOutput(serverUrl string, useFilenameInUrl bool) (FileApiOutput, error) {
	var result FileApiOutput
	err := copier.Copy(&result, &f)
	if err != nil {
		return FileApiOutput{}, err
	}
	result.IsPasswordProtected = f.PasswordHash != ""
	result.IsEncrypted = f.Encryption.IsEncrypted
	result.IsSavedOnLocalStorage = f.AwsBucket == ""
	if f.Encryption.IsEndToEndEncrypted || f.RequiresClientDecryption() {
		result.RequiresClientSideDecryption = true
	}
	result.IsEndToEndEncrypted = f.Encryption.IsEndToEndEncrypted
	result.UrlHotlink = getHotlinkUrl(result, serverUrl, useFilenameInUrl)
	result.UrlDownload = getDownloadUrl(result, serverUrl, useFilenameInUrl)
	result.UploaderId = f.UserId

	return result, nil
}

func getDownloadUrl(input FileApiOutput, serverUrl string, useFilename bool) string {
	if useFilename {
		return serverUrl + "d/" + input.Id + "/" + url.PathEscape(input.Name)
	}
	return serverUrl + "d?id=" + input.Id
}

func getHotlinkUrl(input FileApiOutput, serverUrl string, useFilename bool) string {
	if input.RequiresClientSideDecryption || input.IsPasswordProtected {
		return ""
	}
	if input.HotlinkId != "" {
		return serverUrl + "hotlink/" + input.HotlinkId
	}
	if useFilename {
		return serverUrl + "dh/" + input.Id + "/" + url.PathEscape(input.Name)
	}
	return serverUrl + "downloadFile?id=" + input.Id
}

// ToJsonResult converts the file info to a json String used for returning a result for an upload
func (f *File) ToJsonResult(serverUrl string, includeFilename bool) string {
	info, err := f.ToFileApiOutput(serverUrl, includeFilename)
	if err != nil {
		return errorAsJson(err)
	}

	byteOutput, err := json.Marshal(Result{
		Result:          "OK",
		IncludeFilename: includeFilename,
		FileInfo:        info,
	})
	if err != nil {
		return errorAsJson(err)
	}
	return string(byteOutput)
}

// RequiresClientDecryption checks if the file needs to be decrypted by the client
// (if remote storage or end-to-end encryption)
func (f *File) RequiresClientDecryption() bool {
	if !f.Encryption.IsEncrypted {
		return false
	}
	return !f.IsLocalStorage() || f.Encryption.IsEndToEndEncrypted
}
func errorAsJson(err error) string {
	fmt.Println(err)
	return "{\"Result\":\"error\",\"ErrorMessage\":\"" + err.Error() + "\"}"
}

// Result is the struct used for the result after an upload
// swagger:model UploadResult
type Result struct {
	Result          string        `json:"Result"`
	FileInfo        FileApiOutput `json:"FileInfo"`
	IncludeFilename bool          `json:"IncludeFilename"`
}

// DownloadStatus contains current downloads, so they do not get removed during cleanup
type DownloadStatus struct {
	Id       string
	FileId   string
	ExpireAt int64
}
