package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"os"
	"time"
)

type FileList struct {
	Id                 string `json:"Id"`
	Name               string `json:"Name"`
	Size               string `json:"Size"`
	SHA256             string `json:"SHA256"`
	ExpireAt           int64  `json:"ExpireAt"`
	ExpireAtString     string `json:"ExpireAtString"`
	DownloadsRemaining int    `json:"DownloadsRemaining"`
}

func createNewFile(fileContent *multipart.File, fileHeader *multipart.FileHeader, expireAt int64, downloads int) (FileList, error) {
	id, err := generateRandomString(15)
	if err != nil {
		id = unsafeId(15)
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
	}
	globalConfig.Files[id] = file
	filename := "data/" + file.SHA256
	if !fileExists("data/" + file.SHA256) {
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

func cleanUpOldFiles(sleep bool) {
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
				err := os.Remove("data/" + element.SHA256)
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
	if sleep {
		time.Sleep(time.Hour)
		go cleanUpOldFiles(true)
	}
}
