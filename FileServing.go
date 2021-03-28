package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"os"
	"time"
)

/**
Serving and processing uploaded files
*/

// The length for IDs used in URLs. Can be increased to improve security and decreased to increase readability
const lengthId = 15

// Struct used for saving information about an uploaded file
type FileList struct {
	Id                 string `json:"Id"`
	Name               string `json:"Name"`
	Size               string `json:"Size"`
	SHA256             string `json:"SHA256"`
	ExpireAt           int64  `json:"ExpireAt"`
	ExpireAtString     string `json:"ExpireAtString"`
	DownloadsRemaining int    `json:"DownloadsRemaining"`
	PasswordHash       string `json:"PasswordHash"`
}

// Converts the file info to a json String used for returning a result for an upload
func (f *FileList) toJsonResult() string {
	result := Result{
		Result:   "OK",
		Url:      globalConfig.ServerUrl + "d?id=",
		FileInfo: f,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		return "{\"Result\":\"error\",\"ErrorMessage\":\"" + err.Error() + "\"}"
	}
	return string(bytes)
}

// The struct used for the result after an upload
type Result struct {
	Result   string    `json:"Result"`
	FileInfo *FileList `json:"FileInfo"`
	Url      string    `json:"Url"`
}

// Creates a new file in the system. Called after an upload has been completed. If a file with the same sha256 hash
// already exists, it is deduplicated. This function gathers information about the file, creates and ID and saves
// it into the global configuration.
func createNewFile(fileContent *multipart.File, fileHeader *multipart.FileHeader, expireAt int64, downloads int, password string) (FileList, error) {
	id, err := generateRandomString(lengthId)
	if err != nil {
		id = unsafeId(lengthId)
	}

	fileBytes, err := ioutil.ReadAll(*fileContent)
	if err != nil {
		return FileList{}, err
	}
	hash := sha1.New()
	hash.Write(fileBytes)
	file := FileList{
		Id:                 id,
		Name:               fileHeader.Filename,
		SHA256:             hex.EncodeToString(hash.Sum(nil)),
		Size:               byteCountSI(fileHeader.Size),
		ExpireAt:           expireAt,
		ExpireAtString:     time.Unix(expireAt, 0).Format("2006-01-02 15:04"),
		DownloadsRemaining: downloads,
		PasswordHash:       hashPassword(password, SALT_PW_FILES),
	}
	globalConfig.Files[id] = file
	filename := dataDir + "/" + file.SHA256
	if !fileExists(dataDir + "/" + file.SHA256) {
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return FileList{}, err
		}
		defer destinationFile.Close()
		destinationFile.Write(fileBytes)
	}
	saveConfig()
	return file, nil
}

// Removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func cleanUpOldFiles(periodic bool) {
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	for key, element := range globalConfig.Files {
		if element.ExpireAt < timeNow || element.DownloadsRemaining < 1 {
			deleteFile := true
			for _, secondLoopElement := range globalConfig.Files {
				if element.Id != secondLoopElement.Id && element.SHA256 == secondLoopElement.SHA256 {
					deleteFile = false
				}
			}
			if deleteFile {
				err := os.Remove(dataDir + "/" + element.SHA256)
				if err != nil {
					fmt.Println(err)
				}
			}
			delete(globalConfig.Files, key)
			wasItemDeleted = true
		}
	}
	if wasItemDeleted {
		saveConfig()
		cleanUpOldFiles(false)
	}
	if periodic {
		time.Sleep(time.Hour)
		go cleanUpOldFiles(periodic)
	}
}
