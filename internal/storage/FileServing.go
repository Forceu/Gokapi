package storage

/**
Serving and processing uploaded files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/downloadStatus"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/filestructure"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// NewFile creates a new file in the system. Called after an upload has been completed. If a file with the same sha256 hash
// already exists, it is deduplicated. This function gathers information about the file, creates an ID and saves
// it into the global configuration.
func NewFile(fileContent *multipart.File, fileHeader *multipart.FileHeader, expireAt int64, downloads int, password string) (filestructure.File, error) {
	fileBytes, err := ioutil.ReadAll(*fileContent)
	if err != nil {
		return filestructure.File{}, err
	}
	return processUpload(&fileBytes, fileHeader, expireAt, downloads, password)
}

// Called by NewFile, split into second function to make unit testing easier
func processUpload(fileContent *[]byte, fileHeader *multipart.FileHeader, expireAt int64, downloads int, password string) (filestructure.File, error) {
	id := helper.GenerateRandomString(configuration.ServerSettings.LengthId)
	hash := sha1.New()
	hash.Write(*fileContent)
	file := filestructure.File{
		Id:                 id,
		Name:               fileHeader.Filename,
		SHA256:             hex.EncodeToString(hash.Sum(nil)),
		Size:               helper.ByteCountSI(fileHeader.Size),
		ExpireAt:           expireAt,
		ExpireAtString:     time.Unix(expireAt, 0).Format("2006-01-02 15:04"),
		DownloadsRemaining: downloads,
		PasswordHash:       configuration.HashPassword(password, true),
		ContentType:        fileHeader.Header.Get("Content-Type"),
	}
	addHotlink(&file)
	configuration.ServerSettings.Files[id] = file
	filename := configuration.ServerSettings.DataDir + "/" + file.SHA256
	if !helper.FileExists(configuration.ServerSettings.DataDir + "/" + file.SHA256) {
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return filestructure.File{}, err
		}
		defer destinationFile.Close()
		destinationFile.Write(*fileContent)
	}
	configuration.Save()
	return file, nil
}

var imageFileExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}

// If file is an image, create link for hotlinking
func addHotlink(file *filestructure.File) {
	extension := strings.ToLower(filepath.Ext(file.Name))
	if !helper.IsInArray(imageFileExtensions, extension) {
		return
	}
	link := helper.GenerateRandomString(40) + extension
	file.HotlinkId = link
	configuration.ServerSettings.Hotlinks[link] = filestructure.Hotlink{
		Id:     link,
		FileId: file.Id,
	}
}

// GetFile gets the file by id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFile(id string) (filestructure.File, bool) {
	var emptyResult = filestructure.File{}
	if id == "" {
		return emptyResult, false
	}
	file := configuration.ServerSettings.Files[id]
	if file.ExpireAt < time.Now().Unix() || file.DownloadsRemaining < 1 {
		return emptyResult, false
	}
	if !helper.FileExists(configuration.ServerSettings.DataDir + "/" + file.SHA256) {
		return emptyResult, false
	}
	return file, true
}

// GetFileByHotlink gets the file by hotlink id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFileByHotlink(id string) (filestructure.File, bool) {
	var emptyResult = filestructure.File{}
	if id == "" {
		return emptyResult, false
	}
	hotlink := configuration.ServerSettings.Hotlinks[id]
	return GetFile(hotlink.FileId)
}

// ServeFile subtracts a download allowance and serves the file to the browser
func ServeFile(file filestructure.File, w http.ResponseWriter, r *http.Request, forceDownload bool) {
	file.DownloadsRemaining = file.DownloadsRemaining - 1
	configuration.ServerSettings.Files[file.Id] = file
	storageData, err := os.OpenFile(configuration.ServerSettings.DataDir+"/"+file.SHA256, os.O_RDONLY, 0644)
	helper.Check(err)
	defer storageData.Close()
	size, err := helper.GetFileSize(storageData)
	helper.Check(err)
	if forceDownload {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Type", file.ContentType)
	statusId := downloadStatus.SetDownload(file)
	configuration.Save()
	http.ServeContent(w, r, file.Name, time.Now(), storageData)
	downloadStatus.SetComplete(statusId)
	configuration.Save()
}

// CleanUp removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func CleanUp(periodic bool) {
	downloadStatus.Clean()
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	for key, element := range configuration.ServerSettings.Files {
		fileExists := helper.FileExists(configuration.ServerSettings.DataDir + "/" + element.SHA256)
		if (element.ExpireAt < timeNow || element.DownloadsRemaining < 1 || !fileExists) && !downloadStatus.IsCurrentlyDownloading(element) {
			deleteFile := true
			for _, secondLoopElement := range configuration.ServerSettings.Files {
				if element.Id != secondLoopElement.Id && element.SHA256 == secondLoopElement.SHA256 {
					deleteFile = false
				}
			}
			if deleteFile && fileExists {
				err := os.Remove(configuration.ServerSettings.DataDir + "/" + element.SHA256)
				if err != nil {
					fmt.Println(err)
				}
			}
			if element.HotlinkId != "" {
				delete(configuration.ServerSettings.Hotlinks, element.HotlinkId)
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
