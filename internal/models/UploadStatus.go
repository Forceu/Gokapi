package models

import (
	"encoding/json"
)

// UploadStatus contains information about the current status of a file upload
type UploadStatus struct {
	// ChunkId is the identifier for the chunk
	ChunkId string `json:"chunkid"`
	// CurrentStatus indicates if the chunk is currently being processed (e.g. encrypting or
	// hashing) or being moved/uploaded to the file storage
	// See processingstatus for definition
	CurrentStatus int `json:"currentstatus"`
	// LastUpdate indicates the last status change
	LastUpdate int64 `json:"lastupdate"`
}

// ToJson returns the struct as a Json byte array
func (u *UploadStatus) ToJson() ([]byte, error) {
	return json.Marshal(u)

}
