package models

import (
	"encoding/json"
)

// UploadStatus contains information about the current status of a file upload
type UploadStatus struct {
	ChunkId       string `json:"chunkid"`
	CurrentStatus int    `json:"currentstatus"`
	LastUpdate    int64  `json:"lastupdate"`
}

// ToJson returns the struct as a Json byte array
func (u *UploadStatus) ToJson() ([]byte, error) {
	return json.Marshal(u)

}
