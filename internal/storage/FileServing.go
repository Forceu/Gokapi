package storage

/**
Serving and processing uploaded files
*/

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/datastorage"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/forceu/gokapi/internal/webserver/downloadstatus"
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
func NewFile(fileContent io.Reader, fileHeader *multipart.FileHeader, uploadRequest models.UploadRequest) (models.File, error) {
	id := helper.GenerateRandomString(configuration.Get().LengthId)
	if fileHeader.Size > int64(configuration.Get().MaxFileSizeMB)*1024*1024 {
		return models.File{}, errors.New("upload limit exceeded")
	}
	var hasBeenRenamed bool
	reader, hash, tempFile := generateHash(fileContent, fileHeader, uploadRequest)
	defer deleteTempFile(tempFile, &hasBeenRenamed)
	file := models.File{
		Id:                 id,
		Name:               fileHeader.Filename,
		SHA256:             hex.EncodeToString(hash),
		Size:               helper.ByteCountSI(fileHeader.Size),
		ExpireAt:           uploadRequest.ExpiryTimestamp,
		ExpireAtString:     time.Unix(uploadRequest.ExpiryTimestamp, 0).Format("2006-01-02 15:04"),
		DownloadsRemaining: uploadRequest.AllowedDownloads,
		PasswordHash:       configuration.HashPassword(uploadRequest.Password, true),
		ContentType:        fileHeader.Header.Get("Content-Type"),
	}
	addHotlink(&file)
	if aws.IsAvailable() {
		aws.AddBucketName(&file)
	}
	filename := configuration.Get().DataDir + "/" + file.SHA256
	dataDir := configuration.Get().DataDir
	if aws.IsAvailable() {
		aws.AddBucketName(&file)
		_, err := aws.Upload(reader, file)
		if err != nil {
			return models.File{}, err
		}
		return file, nil
	}
	if !helper.FileExists(dataDir + "/" + file.SHA256) {
		if tempFile != nil {
			err := tempFile.Close()
			helper.Check(err)
			err = os.Rename(tempFile.Name(), dataDir+"/"+file.SHA256)
			helper.Check(err)
			hasBeenRenamed = true
			datastorage.SaveMetaData(file)
			return file, nil
		}
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return models.File{}, err
		}
		defer destinationFile.Close()
		_, err = io.Copy(destinationFile, reader)
		if err != nil {
			return models.File{}, err
		}
	}
	datastorage.SaveMetaData(file)
	return file, nil
}

func deleteTempFile(file *os.File, hasBeenRenamed *bool) {
	if file != nil && !*hasBeenRenamed {
		err := file.Close()
		helper.Check(err)
		err = os.Remove(file.Name())
		helper.Check(err)
	}
}

// Generates the SHA1 hash of an uploaded file and returns a reader for the file, the hash and if a temporary file was created the
// reference to that file.
func generateHash(fileContent io.Reader, fileHeader *multipart.FileHeader, uploadRequest models.UploadRequest) (io.Reader, []byte, *os.File) {
	hash := sha1.New()
	if fileHeader.Size <= int64(uploadRequest.MaxMemory)*1024*1024 {
		content, err := ioutil.ReadAll(fileContent)
		helper.Check(err)
		hash.Write(content)
		return bytes.NewReader(content), hash.Sum(nil), nil
	}
	tempFile, err := os.CreateTemp(uploadRequest.DataDir, "upload")
	helper.Check(err)
	multiWriter := io.MultiWriter(tempFile, hash)
	_, err = io.Copy(multiWriter, fileContent)
	helper.Check(err)
	_, err = tempFile.Seek(0, io.SeekStart)
	helper.Check(err)
	// Instead of returning a reference to the file as the 3rd result, one could use reflections. However, that would be more expensive.
	return tempFile, hash.Sum(nil), tempFile
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
	datastorage.SaveHotlink(*file)
}

// GetFile gets the file by id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFile(id string) (models.File, bool) {
	var emptyResult = models.File{}
	if id == "" {
		return emptyResult, false
	}
	file, ok := datastorage.GetMetaDataById(id)
	if !ok {
		return emptyResult, false
	}
	if file.ExpireAt < time.Now().Unix() || file.DownloadsRemaining < 1 {
		return emptyResult, false
	}
	if !FileExists(file, configuration.Get().DataDir) {
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
	fileId, ok := datastorage.GetHotlink(id)
	if !ok {
		return emptyResult, false
	}
	return GetFile(fileId)
}

// ServeFile subtracts a download allowance and serves the file to the browser
func ServeFile(file models.File, w http.ResponseWriter, r *http.Request, forceDownload bool) {
	file.DownloadsRemaining = file.DownloadsRemaining - 1
	datastorage.SaveMetaData(file)
	logging.AddDownload(&file, r)

	// If file is not stored on AWS
	if file.AwsBucket == "" {
		fileData, size := getFileHandler(file, configuration.Get().DataDir)
		defer fileData.Close()
		if forceDownload {
			w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
		} else {
			w.Header().Set("Content-Disposition", "inline; filename=\""+file.Name+"\"")
		}
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Content-Type", file.ContentType)
		statusId := downloadstatus.SetDownload(file)
		http.ServeContent(w, r, file.Name, time.Now(), fileData)
		downloadstatus.SetComplete(statusId)
	} else {
		// If file is stored on AWS
		downloadstatus.SetDownload(file)
		err := aws.RedirectToDownload(w, r, file, forceDownload)
		helper.Check(err)
		// We are not setting a download complete status, as there is no reliable way to confirm that the
		// file has been completely downloaded. It expires automatically after 24 hours.
	}
}

func getFileHandler(file models.File, dataDir string) (*os.File, int64) {
	storageData, err := os.OpenFile(dataDir+"/"+file.SHA256, os.O_RDONLY, 0644)
	helper.Check(err)
	size, err := helper.GetFileSize(storageData)
	helper.Check(err)
	return storageData, size
}

// FileExists checks if the file exists locally or in S3
func FileExists(file models.File, dataDir string) bool {
	if file.AwsBucket != "" {
		result, err := aws.FileExists(file)
		if err != nil {
			fmt.Println("Warning, cannot check file " + file.Id + ": " + err.Error())
			return true
		}
		return result
	}
	return helper.FileExists(dataDir + "/" + file.SHA256)
}

// CleanUp removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func CleanUp(periodic bool) {
	datastorage.RunGarbageCollection()
	downloadstatus.Clean()
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	for key, element := range datastorage.GetAllMetadata() {
		fileExists := FileExists(element, configuration.Get().DataDir)
		if (element.ExpireAt < timeNow || element.DownloadsRemaining < 1 || !fileExists) && !downloadstatus.IsCurrentlyDownloading(element) {
			deleteFile := true
			for _, secondLoopElement := range datastorage.GetAllMetadata() {
				if element.Id != secondLoopElement.Id && element.SHA256 == secondLoopElement.SHA256 {
					deleteFile = false
				}
			}
			if deleteFile && fileExists {
				deleteSource(element, configuration.Get().DataDir)
			}
			if element.HotlinkId != "" {
				datastorage.DeleteHotlink(element.HotlinkId)
			}
			datastorage.DeleteMetaData(key)
			wasItemDeleted = true
		}
	}
	if wasItemDeleted {
		CleanUp(false)
	}
	if periodic {
		time.Sleep(time.Hour)
		go CleanUp(periodic)
	}
}

func deleteSource(file models.File, dataDir string) {
	var err error
	if file.AwsBucket != "" {
		_, err = aws.DeleteObject(file)
		helper.Check(err)
	} else {
		err = os.Remove(dataDir + "/" + file.SHA256)
	}
	if err != nil {
		fmt.Println(err)
	}
}

// DeleteFile is called when an admin requests deletion of a file
// Returns true if file was deleted or false if ID did not exist
func DeleteFile(keyId string) bool {
	if keyId == "" {
		return false
	}
	item, ok := datastorage.GetMetaDataById(keyId)
	if !ok {
		return false
	}
	item.ExpireAt = 0
	datastorage.SaveMetaData(item)
	for _, status := range downloadstatus.GetAll() {
		if status.FileId == item.Id {
			downloadstatus.SetComplete(status.Id)
		}
	}
	CleanUp(false)
	return true
}
