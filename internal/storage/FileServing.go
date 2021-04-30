package storage

/**
Serving and processing uploaded files
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/downloadstatus"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
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

// NewFile creates a new file in the system. Called after an upload has been completed. If a file with the same sha256 hash
// already exists, it is deduplicated. This function gathers information about the file, creates an ID and saves
// it into the global configuration.
func NewFile(fileContent io.Reader, fileHeader *multipart.FileHeader, expireAt int64, downloads int, password string) (models.File, error) {
	fileBytes, err := ioutil.ReadAll(fileContent)
	if err != nil {
		return models.File{}, err
	}
	id := helper.GenerateRandomString(configuration.GetLengthId())
	hash := sha1.New()
	hash.Write(fileBytes)
	file := models.File{
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
	settings := configuration.GetServerSettings()
	defer func() { configuration.ReleaseAndSave() }()
	settings.Files[id] = file
	filename := settings.DataDir + "/" + file.SHA256
	if !helper.FileExists(settings.DataDir + "/" + file.SHA256) {
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return models.File{}, err
		}
		defer destinationFile.Close()
		destinationFile.Write(fileBytes)
	}
	return file, nil
}

var imageFileExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}

// If file is an image, create link for hotlinking
func addHotlink(file *models.File) {
	extension := strings.ToLower(filepath.Ext(file.Name))
	if !helper.IsInArray(imageFileExtensions, extension) {
		return
	}
	link := helper.GenerateRandomString(40) + extension
	file.HotlinkId = link
	settings := configuration.GetServerSettings()
	settings.Hotlinks[link] = models.Hotlink{
		Id:     link,
		FileId: file.Id,
	}
	configuration.Release()
}

// GetFile gets the file by id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFile(id string) (models.File, bool) {
	var emptyResult = models.File{}
	if id == "" {
		return emptyResult, false
	}
	settings := configuration.GetServerSettings()
	file := settings.Files[id]
	configuration.Release()
	if file.ExpireAt < time.Now().Unix() || file.DownloadsRemaining < 1 {
		return emptyResult, false
	}
	if !helper.FileExists(settings.DataDir + "/" + file.SHA256) {
		return emptyResult, false
	}
	return file, true
}

// GetFileByHotlink gets the file by hotlink id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFileByHotlink(id string) (models.File, bool) {
	var emptyResult = models.File{}
	if id == "" {
		return emptyResult, false
	}
	settings := configuration.GetServerSettings()
	hotlink := settings.Hotlinks[id]
	configuration.Release()
	return GetFile(hotlink.FileId)
}

// ServeFile subtracts a download allowance and serves the file to the browser
func ServeFile(file models.File, w http.ResponseWriter, r *http.Request, forceDownload bool) {
	file.DownloadsRemaining = file.DownloadsRemaining - 1
	settings := configuration.GetServerSettings()
	settings.Files[file.Id] = file
	storageData, err := os.OpenFile(settings.DataDir+"/"+file.SHA256, os.O_RDONLY, 0644)
	configuration.Release()
	helper.Check(err)
	defer storageData.Close()
	size, err := helper.GetFileSize(storageData)
	helper.Check(err)
	if forceDownload {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Type", file.ContentType)
	statusId := downloadstatus.SetDownload(file)
	http.ServeContent(w, r, file.Name, time.Now(), storageData)
	downloadstatus.SetComplete(statusId)
}

// CleanUp removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func CleanUp(periodic bool) {
	downloadstatus.Clean()
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	settings := configuration.GetServerSettings()
	for key, element := range settings.Files {
		fileExists := helper.FileExists(settings.DataDir + "/" + element.SHA256)
		if (element.ExpireAt < timeNow || element.DownloadsRemaining < 1 || !fileExists) && !downloadstatus.IsCurrentlyDownloading(element, settings) {
			deleteFile := true
			for _, secondLoopElement := range settings.Files {
				if element.Id != secondLoopElement.Id && element.SHA256 == secondLoopElement.SHA256 {
					deleteFile = false
				}
			}
			if deleteFile && fileExists {
				err := os.Remove(settings.DataDir + "/" + element.SHA256)
				if err != nil {
					fmt.Println(err)
				}
			}
			if element.HotlinkId != "" {
				delete(settings.Hotlinks, element.HotlinkId)
			}
			delete(settings.Files, key)
			wasItemDeleted = true
		}
	}
	configuration.Release()
	if wasItemDeleted {
		configuration.Save()
		CleanUp(false)
	}
	if periodic {
		time.Sleep(time.Hour)
		go CleanUp(periodic)
	}
}

// DeleteFile is called when an admin requests deletion of a file
func DeleteFile(keyId string) {
	settings := configuration.GetServerSettings()
	item := settings.Files[keyId]
	item.ExpireAt = 0
	settings.Files[keyId] = item
	configuration.Release()
	CleanUp(false)
}
