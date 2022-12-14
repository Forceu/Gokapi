package models

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
)

// File is a struct used for saving information about an uploaded file
type File struct {
	Id                 string         `json:"Id"`
	Name               string         `json:"Name"`
	Size               string         `json:"Size"`
	SHA1               string         `json:"SHA1"`
	ExpireAt           int64          `json:"ExpireAt"`
	SizeBytes          int64          `json:"SizeBytes"`
	ExpireAtString     string         `json:"ExpireAtString"`
	DownloadsRemaining int            `json:"DownloadsRemaining"`
	DownloadCount      int            `json:"DownloadCount"`
	PasswordHash       string         `json:"PasswordHash"`
	HotlinkId          string         `json:"HotlinkId"`
	ContentType        string         `json:"ContentType"`
	AwsBucket          string         `json:"AwsBucket"`
	Encryption         EncryptionInfo `json:"Encryption"`
	UnlimitedDownloads bool           `json:"UnlimitedDownloads"`
	UnlimitedTime      bool           `json:"UnlimitedTime"`
}

// FileApiOutput will be displayed for public outputs from the ID, hiding sensitive information
type FileApiOutput struct {
	Id                           string `json:"Id"`
	Name                         string `json:"Name"`
	Size                         string `json:"Size"`
	HotlinkId                    string `json:"HotlinkId"`
	ContentType                  string `json:"ContentType"`
	ExpireAt                     int64  `json:"ExpireAt"`
	SizeBytes                    int64  `json:"SizeBytes"`
	ExpireAtString               string `json:"ExpireAtString"`
	DownloadsRemaining           int    `json:"DownloadsRemaining"`
	DownloadCount                int    `json:"DownloadCount"`
	UnlimitedDownloads           bool   `json:"UnlimitedDownloads"`
	UnlimitedTime                bool   `json:"UnlimitedTime"`
	RequiresClientSideDecryption bool   `json:"RequiresClientSideDecryption"`
	IsEncrypted                  bool   `json:"IsEncrypted"`
	IsPasswordProtected          bool   `json:"IsPasswordProtected"`
	IsSavedOnLocalStorage        bool   `json:"IsSavedOnLocalStorage"`
}

// EncryptionInfo holds information about the encryption used on the file
type EncryptionInfo struct {
	IsEncrypted         bool   `json:"IsEncrypted"`
	IsEndToEndEncrypted bool   `json:"IsEndToEndEncrypted"`
	DecryptionKey       []byte `json:"DecryptionKey"`
	Nonce               []byte `json:"Nonce"`
}

// IsLocalStorage returns true if the file is not stored on a remote storage
func (f *File) IsLocalStorage() bool {
	return f.AwsBucket == ""
}

// ToFileApiOutput returns a json object without sensitive information
func (f *File) ToFileApiOutput(isClientSideDecryption bool) (FileApiOutput, error) {
	var result FileApiOutput
	err := copier.Copy(&result, &f)
	if err != nil {
		return FileApiOutput{}, err
	}
	result.IsPasswordProtected = f.PasswordHash != ""
	result.IsEncrypted = f.Encryption.IsEncrypted
	result.IsSavedOnLocalStorage = f.AwsBucket == ""
	if f.Encryption.IsEndToEndEncrypted || isClientSideDecryption {
		result.RequiresClientSideDecryption = true
	}
	return result, nil
}

// ToJsonResult converts the file info to a json String used for returning a result for an upload
func (f *File) ToJsonResult(serverUrl string, isClientSideDecryption bool) string {
	info, err := f.ToFileApiOutput(isClientSideDecryption)
	if err != nil {
		return errorAsJson(err)
	}
	result := Result{
		Result:            "OK",
		Url:               serverUrl + "d?id=",
		HotlinkUrl:        serverUrl + "hotlink/",
		GenericHotlinkUrl: serverUrl + "downloadFile?id=",
		FileInfo:          info,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return errorAsJson(err)
	}
	return string(bytes)
}

func errorAsJson(err error) string {
	fmt.Println(err)
	return "{\"Result\":\"error\",\"ErrorMessage\":\"" + err.Error() + "\"}"
}

// Result is the struct used for the result after an upload
// swagger:model UploadResult
type Result struct {
	Result            string        `json:"Result"`
	FileInfo          FileApiOutput `json:"FileInfo"`
	Url               string        `json:"Url"`
	HotlinkUrl        string        `json:"HotlinkUrl"`
	GenericHotlinkUrl string        `json:"GenericHotlinkUrl"`
}

// DownloadStatus contains current downloads, so they do not get removed during cleanup
type DownloadStatus struct {
	Id       string
	FileId   string
	ExpireAt int64
}
