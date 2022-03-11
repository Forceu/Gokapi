package models

import (
	"encoding/json"
	"log"
)

// Configuration is a struct that contains the global configuration
type Configuration struct {
	Authentication      AuthenticationConfig `json:"Authentication"`
	Port                string               `json:"Port"`
	ServerUrl           string               `json:"ServerUrl"`
	RedirectUrl         string               `json:"RedirectUrl"`
	ConfigVersion       int                  `json:"ConfigVersion"`
	LengthId            int                  `json:"LengthId"`
	DataDir             string               `json:"DataDir"`
	MaxMemory           int                  `json:"MaxMemory"`
	UseSsl              bool                 `json:"UseSsl"`
	MaxFileSizeMB       int                  `json:"MaxFileSizeMB"`
	Encryption          Encryption           `json:"Encryption"`
	PicturesAlwaysLocal bool                 `json:"PicturesAlwaysLocal"`
}

// Encryption hold information about the encryption used on this file
type Encryption struct {
	Level        int
	Cipher       []byte
	Salt         string
	Checksum     string
	ChecksumSalt string
}

// LastUploadValues is used to save the last used values for uploads in the database
type LastUploadValues struct {
	Downloads         int
	TimeExpiry        int
	Password          string
	UnlimitedDownload bool
	UnlimitedTime     bool
}

// ToJson returns an idented JSon representation
func (c Configuration) ToJson() []byte {
	result, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Fatal("Error encoding configuration:", err)
	}
	return result
}

// ToString returns the object as an unidented Json string used for test units
func (c Configuration) ToString() string {
	result, err := json.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}
