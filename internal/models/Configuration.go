package models

import (
	"encoding/json"
)

// Configuration is a struct that contains the global configuration
type Configuration struct {
	Authentication      AuthenticationConfig `json:"Authentication"`
	Port                string               `json:"Port"`
	ServerUrl           string               `json:"ServerUrl"`
	RedirectUrl         string               `json:"RedirectUrl"`
	PublicName          string               `json:"PublicName"`
	DataDir             string               `json:"DataDir"`
	DatabaseUrl         string               `json:"DatabaseUrl"`
	ConfigVersion       int                  `json:"ConfigVersion"`
	MaxFileSizeMB       int                  `json:"MaxFileSizeMB"`
	MaxMemory           int                  `json:"MaxMemory"`
	ChunkSize           int                  `json:"ChunkSize"`
	MaxParallelUploads  int                  `json:"MaxParallelUploads"`
	Encryption          Encryption           `json:"Encryption"`
	UseSsl              bool                 `json:"UseSsl"`
	PicturesAlwaysLocal bool                 `json:"PicturesAlwaysLocal"`
	SaveIp              bool                 `json:"SaveIp"`
	IncludeFilename     bool                 `json:"IncludeFilename"`
}

// Encryption holds information about the encryption used on this file
type Encryption struct {
	Level        int
	Cipher       []byte
	Salt         string
	Checksum     string
	ChecksumSalt string
}

// ToJson returns an indented Json representation
func (c Configuration) ToJson() []byte {
	result, err := json.MarshalIndent(c, "", "  ")
	checkError(err)
	return result
}

// ToString returns the object as an unindented JSON string used for test units
func (c Configuration) ToString() string {
	result, err := json.Marshal(c)
	checkError(err)
	return string(result)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
