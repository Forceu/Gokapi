package models

import (
	"encoding/json"
	"fmt"
)

// File is a struct used for saving information about an uploaded file
type File struct {
	Id                 string         `json:"Id"`
	Name               string         `json:"Name"`
	Size               string         `json:"Size"`
	SHA256             string         `json:"SHA256"`
	ExpireAt           int64          `json:"ExpireAt"`
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

// EncryptionInfo holds information about the encryption used on the file
type EncryptionInfo struct {
	IsEncrypted   bool   `json:"IsEncrypted"`
	DecryptionKey []byte `json:"DecryptionKey"`
	Nonce         []byte `json:"Nonce"`
}

// ToJsonResult converts the file info to a json String used for returning a result for an upload
func (f *File) ToJsonResult(serverUrl string) string {
	result := Result{
		Result:     "OK",
		Url:        serverUrl + "d?id=",
		HotlinkUrl: serverUrl + "hotlink/",
		FileInfo:   f,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		return "{\"Result\":\"error\",\"ErrorMessage\":\"" + err.Error() + "\"}"
	}
	return string(bytes)
}

// Result is the struct used for the result after an upload
// swagger:model UploadResult
type Result struct {
	Result     string `json:"Result"`
	FileInfo   *File  `json:"FileInfo"`
	Url        string `json:"Url"`
	HotlinkUrl string `json:"HotlinkUrl"`
}

// DownloadStatus contains current downloads, so they do not get removed during cleanup
type DownloadStatus struct {
	Id       string
	FileId   string
	ExpireAt int64
}
