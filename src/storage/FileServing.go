package storage

/**
Serving and processing uploaded files
*/

import (
	"Gokapi/src/configuration"
	"Gokapi/src/helper"
	"Gokapi/src/storage/filestructure"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"os"
	"time"
)

// The length for IDs used in URLs. Can be increased to improve security and decreased to increase readability
const lengthId = 15

// Creates a new file in the system. Called after an upload has been completed. If a file with the same sha256 hash
// already exists, it is deduplicated. This function gathers information about the file, creates and ID and saves
// it into the global configuration.
func NewFile(fileContent *multipart.File, fileHeader *multipart.FileHeader, expireAt int64, downloads int, password string) (filestructure.File, error) {
	id := helper.GenerateRandomString(lengthId)
	fileBytes, err := ioutil.ReadAll(*fileContent)
	if err != nil {
		return filestructure.File{}, err
	}
	hash := sha1.New()
	hash.Write(fileBytes)
	file := filestructure.File{
		Id:                 id,
		Name:               fileHeader.Filename,
		SHA256:             hex.EncodeToString(hash.Sum(nil)),
		Size:               helper.ByteCountSI(fileHeader.Size),
		ExpireAt:           expireAt,
		ExpireAtString:     time.Unix(expireAt, 0).Format("2006-01-02 15:04"),
		DownloadsRemaining: downloads,
		PasswordHash:       configuration.HashPassword(password, true),
	}
	configuration.ServerSettings.Files[id] = file
	filename := configuration.Environment.DataDir + "/" + file.SHA256
	if !helper.FileExists(configuration.Environment.DataDir + "/" + file.SHA256) {
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return filestructure.File{}, err
		}
		defer destinationFile.Close()
		destinationFile.Write(fileBytes)
	}
	configuration.Save()
	return file, nil
}

// Removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func CleanUp(periodic bool) {
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	for key, element := range configuration.ServerSettings.Files {
		if element.ExpireAt < timeNow || element.DownloadsRemaining < 1 {
			deleteFile := true
			for _, secondLoopElement := range configuration.ServerSettings.Files {
				if element.Id != secondLoopElement.Id && element.SHA256 == secondLoopElement.SHA256 {
					deleteFile = false
				}
			}
			if deleteFile {
				err := os.Remove(configuration.Environment.DataDir + "/" + element.SHA256)
				if err != nil {
					fmt.Println(err)
				}
			}
			delete(configuration.ServerSettings.Files, key)
			wasItemDeleted = true
		}
	}
	if wasItemDeleted {
		configuration.Save()
		CleanUp(false)
	}
	if periodic {
		time.Sleep(time.Hour)
		go CleanUp(periodic)
	}
}
