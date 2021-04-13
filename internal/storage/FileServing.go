package storage

/**
Serving and processing uploaded files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/filestructure"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	addHotlink(&file)
	configuration.ServerSettings.Files[id] = file
	filename := configuration.ServerSettings.DataDir + "/" + file.SHA256
	if !helper.FileExists(configuration.ServerSettings.DataDir + "/" + file.SHA256) {
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

// Gets the file by id. Returns (empty File, false) if invalid / expired file
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

// Gets the file by hotlink id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFileByHotlink(id string) (filestructure.File, bool) {
	var emptyResult = filestructure.File{}
	if id == "" {
		return emptyResult, false
	}
	hotlink := configuration.ServerSettings.Hotlinks[id]
	return GetFile(hotlink.FileId)
}

// Subtracts a download allowance and serves the file to the browser
func ServeFile(file filestructure.File, w http.ResponseWriter, r *http.Request, forceDownload bool) {
	file.DownloadsRemaining = file.DownloadsRemaining - 1
	configuration.ServerSettings.Files[file.Id] = file
	// Investigate: Possible race condition with clean-up routine?
	configuration.Save()

	if forceDownload {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	}
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	storageData, err := os.OpenFile(configuration.ServerSettings.DataDir+"/"+file.SHA256, os.O_RDONLY, 0644)
	helper.Check(err)
	defer storageData.Close()
	size, err := getFileSize(storageData)
	if err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	helper.Check(err)
	_, _ = io.Copy(w, storageData)
}

func getFileSize(file *os.File) (int64, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
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
