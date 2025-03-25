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
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"github.com/forceu/gokapi/internal/storage/filesystem"
	"github.com/forceu/gokapi/internal/storage/filesystem/s3filesystem/aws"
	"github.com/forceu/gokapi/internal/storage/processingstatus"
	"github.com/forceu/gokapi/internal/webserver/downloadstatus"
	"github.com/forceu/gokapi/internal/webserver/headers"
	"github.com/forceu/gokapi/internal/webserver/sse"
	"github.com/jinzhu/copier"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrorFileTooLarge is an error that is called when a file larger than the set maximum is uploaded
var ErrorFileTooLarge = errors.New("upload limit exceeded")

// ErrorReplaceE2EFile is caused when an end-to-end encrypted file is replaced
var ErrorReplaceE2EFile = errors.New("end-to-end encrypted files cannot be replaced")

// ErrorFileNotFound is raised when an invalid ID is passed or the file has expired
var ErrorFileNotFound = errors.New("file not found")

// NewFile creates a new file in the system. Called after an upload from the API has been completed. If a file with the same sha1 hash
// already exists, it is deduplicated. This function gathers information about the file, creates an ID and saves
// it into the global configuration. It is now only used by the API, the web UI uses NewFileFromChunk
func NewFile(fileContent io.Reader, fileHeader *multipart.FileHeader, userId int, uploadRequest models.UploadRequest) (models.File, error) {
	if !isAllowedFileSize(fileHeader.Size) {
		return models.File{}, ErrorFileTooLarge
	}
	var hasBeenRenamed bool
	reader, hash, tempFile, encInfo := generateHashAndEncrypt(fileContent, fileHeader)
	defer deleteTempFile(tempFile, &hasBeenRenamed)
	header, err := chunking.ParseMultipartHeader(fileHeader)
	if err != nil {
		return models.File{}, err
	}
	file := createNewMetaData(hex.EncodeToString(hash), header, userId, uploadRequest)
	file.Encryption = encInfo
	filename := configuration.Get().DataDir + "/" + file.SHA1
	dataDir := configuration.Get().DataDir

	fileWithHashExists := FileExists(file, configuration.Get().DataDir)

	if !file.IsLocalStorage() {
		if !fileWithHashExists {
			_, err = aws.Upload(reader, file)
			if err != nil {
				return models.File{}, err
			}
		}
		database.SaveMetaData(file)
		return file, nil
	}

	if fileWithHashExists {
		encryptionLevel := configuration.Get().Encryption.Level
		previousEncryption, ok := getEncInfoFromExistingFile(file.SHA1)
		if !ok && encryptionLevel != encryption.NoEncryption && encryptionLevel != encryption.EndToEndEncryption {
			err = os.Remove(dataDir + "/" + file.SHA1)
			helper.Check(err)
			fileWithHashExists = false
		} else {
			file.Encryption = previousEncryption
		}
	}

	if !fileWithHashExists {
		if tempFile != nil {
			err = tempFile.Close()
			helper.Check(err)
			err = os.Rename(tempFile.Name(), dataDir+"/"+file.SHA1)
			helper.Check(err)
			hasBeenRenamed = true
			database.SaveMetaData(file)
			return file, nil
		}
		destinationFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return models.File{}, err
		}
		defer destinationFile.Close()
		_, err = io.Copy(destinationFile, reader)
		if err != nil {
			return models.File{}, err
		}
	}
	database.SaveMetaData(file)
	return file, nil
}

// isAllowedFileSize returns true if the file is not greater than the allowed filesize
func isAllowedFileSize(size int64) bool {
	return size <= int64(configuration.Get().MaxFileSizeMB)*1024*1024
}

// validateChunkInfo checks if the filesize is allowed and if the submitted filesize (user input) is the actual filesize
func validateChunkInfo(file *os.File, fileHeader chunking.FileHeader) error {
	if !isAllowedFileSize(fileHeader.Size) {
		return ErrorFileTooLarge
	}
	size, err := helper.GetFileSize(file)
	if err != nil {
		return err
	}
	if size != fileHeader.Size {
		return errors.New("total filesize does not match")
	}
	return nil
}

// GetUploadCounts returns the currently uploaded files per user
func GetUploadCounts() map[int]int {
	result := make(map[int]int)
	timeNow := time.Now().Unix()
	files := database.GetAllMetadata()
	for _, file := range files {
		if !IsExpiredFile(file, timeNow) {
			result[file.UserId] = result[file.UserId] + 1
		}
	}
	return result
}

// NewFileFromChunk creates a new file in the system after a chunk upload has fully completed. If a file with the same sha1 hash
// already exists, it is deduplicated. This function gathers information about the file, creates an ID and saves
// it into the global configuration.
func NewFileFromChunk(chunkId string, fileHeader chunking.FileHeader, userId int, uploadRequest models.UploadRequest) (models.File, error) {
	file, err := chunking.GetFileByChunkId(chunkId)
	if err != nil {
		return models.File{}, err
	}
	defer file.Close()
	err = validateChunkInfo(file, fileHeader)
	if err != nil {
		return models.File{}, err
	}

	processingstatus.Set(chunkId, processingstatus.StatusHashingOrEncrypting, models.File{}, nil)
	hash, err := getChunkFileHash(file, uploadRequest.IsEndToEndEncrypted)
	if err != nil {
		return models.File{}, err
	}
	metaData := createNewMetaData(hash, fileHeader, userId, uploadRequest)
	fileExists := FileExists(metaData, configuration.Get().DataDir)
	if fileExists {
		fileExists = copyEncryptionInfo(&metaData)
		err = file.Close()
		if err != nil {
			return models.File{}, err
		}
		err = os.Remove(file.Name())
		if err != nil {
			return models.File{}, err
		}
	}

	if !fileExists {
		fileToMove := file
		if !isEncryptionRequested() {
			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				return models.File{}, err
			}
		} else {
			tempFile, err := encryptChunkFile(file, &metaData)
			if err != nil {
				return models.File{}, err
			}
			fileToMove = tempFile
		}
		processingstatus.Set(chunkId, processingstatus.StatusUploading, models.File{}, nil)
		if metaData.IsLocalStorage() {
			err = filesystem.GetLocal().MoveToFilesystem(fileToMove, metaData)
		} else {
			err = filesystem.ActiveStorageSystem.MoveToFilesystem(fileToMove, metaData)
		}
		if err != nil {
			return models.File{}, err
		}
	}
	database.SaveMetaData(metaData)
	processingstatus.Set(chunkId, processingstatus.StatusFinished, metaData, nil)
	return metaData, nil
}

// copyEncryptionInfo copies encryption info from an existing file,
// if possible. If not possible due to incompatible encryption level,
// the old file is removed.
//
// The function returns false if the old file was removed.
func copyEncryptionInfo(metaData *models.File) bool {
	encryptionLevel := configuration.Get().Encryption.Level
	previousEncryption, ok := getEncInfoFromExistingFile(metaData.SHA1)
	if !ok && encryptionLevel != encryption.NoEncryption && encryptionLevel != encryption.EndToEndEncryption {
		err := os.Remove(configuration.Get().DataDir + "/" + metaData.SHA1)
		helper.Check(err)
		return false
	}
	metaData.Encryption = previousEncryption
	return true
}

func getChunkFileHash(file *os.File, isEndToEndEncryted bool) (string, error) {
	if isEndToEndEncryted {
		return "e2e-" + helper.GenerateRandomString(20), nil
	}
	hash, err := hashFile(file, isEncryptionRequested())
	if err != nil {
		_ = file.Close()
		return "", err
	}
	return hash, nil
}

func encryptChunkFile(file *os.File, metadata *models.File) (*os.File, error) {

	var removeTempFiles = func() {
		err := file.Close()
		if err != nil {
			fmt.Println("Warning: cannot close plain-text file")
			fmt.Println(err)
		}
		err = os.Remove(file.Name())
		if err != nil {
			fmt.Println("Warning: cannot remove plain-text file")
			fmt.Println(err)
		}

	}

	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		removeTempFiles()
		return nil, err
	}
	tempFileEnc, err := os.CreateTemp(configuration.Get().DataDir, "upload")
	if err != nil {
		removeTempFiles()
		return nil, err
	}
	encInfo := metadata.Encryption
	err = encryption.Encrypt(&encInfo, file, tempFileEnc)
	if err != nil {
		removeTempFiles()
		return nil, err
	}
	_, err = tempFileEnc.Seek(0, io.SeekStart)
	if err != nil {
		removeTempFiles()
		return nil, err
	}
	metadata.Encryption = encInfo
	err = file.Close()
	if err != nil {
		return nil, err
	}
	err = os.Remove(file.Name())
	if err != nil {
		return nil, err
	}
	return tempFileEnc, nil
}

// FormatTimestamp converts a timestamp to a string in the format YYYY-MM-DD HH:MM
func FormatTimestamp(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04")
}

func createNewMetaData(hash string, fileHeader chunking.FileHeader, userId int, uploadRequest models.UploadRequest) models.File {
	file := models.File{
		Id:                 createNewId(),
		Name:               fileHeader.Filename,
		SHA1:               hash,
		Size:               helper.ByteCountSI(fileHeader.Size),
		SizeBytes:          fileHeader.Size,
		ContentType:        fileHeader.ContentType,
		ExpireAt:           uploadRequest.ExpiryTimestamp,
		ExpireAtString:     FormatTimestamp(uploadRequest.ExpiryTimestamp),
		DownloadsRemaining: uploadRequest.AllowedDownloads,
		UnlimitedTime:      uploadRequest.UnlimitedTime,
		UnlimitedDownloads: uploadRequest.UnlimitedDownload,
		PasswordHash:       configuration.HashPassword(uploadRequest.Password, true),
		UserId:             userId,
	}
	if uploadRequest.IsEndToEndEncrypted {
		file.Encryption = models.EncryptionInfo{IsEndToEndEncrypted: true, IsEncrypted: true}
		file.Size = helper.ByteCountSI(uploadRequest.RealSize)
	}
	if isEncryptionRequested() {
		file.Encryption.IsEncrypted = true
	}
	if aws.IsAvailable() {
		if !configuration.Get().PicturesAlwaysLocal || !isPictureFile(file.Name) {
			aws.AddBucketName(&file)
		}
	}
	AddHotlink(&file)
	return file
}

// createNewId returns a random ID
func createNewId() string {
	return helper.GenerateRandomString(configuration.Get().LengthId)
}

func getEncInfoFromExistingFile(hash string) (models.EncryptionInfo, bool) {
	encryptionLevel := configuration.Get().Encryption.Level
	if encryptionLevel == encryption.NoEncryption || encryptionLevel == encryption.EndToEndEncryption {
		return models.EncryptionInfo{}, true
	}
	allFiles := database.GetAllMetadata()
	for _, existingFile := range allFiles {
		if existingFile.SHA1 == hash {
			return existingFile.Encryption, true
		}
	}
	return models.EncryptionInfo{}, false
}

func deleteTempFile(file *os.File, hasBeenRenamed *bool) {
	if file != nil && !*hasBeenRenamed {
		err := file.Close()
		helper.Check(err)
		err = os.Remove(file.Name())
		helper.Check(err)
	}
}

const (
	// ParamExpiry is a bit to indicate that the time remaining shall be changed after a duplication
	ParamExpiry int = 1 << iota
	// ParamDownloads is a bit to indicate that the downloads remaining shall be changed after a duplication
	ParamDownloads
	// ParamPassword is a bit to indicate that the password shall be changed after a duplication
	ParamPassword
	// ParamName is a bit to indicate that the filename shall be changed after a duplication
	ParamName
)

// ReplaceFile replaces the file content of fileId with the content of newFileContentId
// Replacing e2e encrypted files is NOT possible
func ReplaceFile(fileId, newFileContentId string, delete bool) (models.File, error) {
	file, ok := GetFile(fileId)
	if !ok {
		return models.File{}, ErrorFileNotFound
	}
	newFileContent, ok := GetFile(newFileContentId)
	if !ok {
		return models.File{}, ErrorFileNotFound
	}
	if file.Encryption.IsEndToEndEncrypted || newFileContent.Encryption.IsEndToEndEncrypted {
		return models.File{}, ErrorReplaceE2EFile
	}

	file.Name = newFileContent.Name
	file.Size = newFileContent.Size
	file.SHA1 = newFileContent.SHA1
	file.ContentType = newFileContent.ContentType
	file.AwsBucket = newFileContent.AwsBucket
	file.SizeBytes = newFileContent.SizeBytes
	file.Encryption = newFileContent.Encryption
	database.SaveMetaData(file)
	if delete {
		DeleteFile(newFileContent.Id, false)
	}
	return file, nil
}

func isChangeRequested(parametersToChange, parameter int) bool {
	return parametersToChange&parameter != 0
}

// DuplicateFile creates a copy of an existing file with new parameters
func DuplicateFile(file models.File, parametersToChange int, newFileName string, fileParameters models.UploadRequest) (models.File, error) {

	// apiDuplicateFile expects fileParameters.IsEndToEndEncrypted and fileParameters.RealSize not to be used,
	// change in apiDuplicateFile if using in this function!

	var newFile models.File
	err := copier.Copy(&newFile, &file)
	if err != nil {
		return models.File{}, err
	}

	changeExpiry := isChangeRequested(parametersToChange, ParamExpiry)
	changeDownloads := isChangeRequested(parametersToChange, ParamDownloads)
	changePassword := isChangeRequested(parametersToChange, ParamPassword)
	changeName := isChangeRequested(parametersToChange, ParamName)

	if changeExpiry {
		newFile.ExpireAt = fileParameters.ExpiryTimestamp
		newFile.ExpireAtString = FormatTimestamp(fileParameters.ExpiryTimestamp)
		newFile.UnlimitedTime = fileParameters.UnlimitedTime
	}
	if changeDownloads {
		newFile.DownloadsRemaining = fileParameters.AllowedDownloads
		newFile.UnlimitedDownloads = fileParameters.UnlimitedDownload
	}
	if changePassword {
		newFile.PasswordHash = configuration.HashPassword(fileParameters.Password, true)
	}
	if changeName {
		newFile.Name = newFileName
	}

	newFile.Id = createNewId()
	newFile.DownloadCount = 0
	AddHotlink(&newFile)

	database.SaveMetaData(newFile)
	return newFile, nil
}

func hashFile(input io.Reader, useSalt bool) (string, error) {
	hash := sha1.New()
	_, err := io.Copy(hash, input)
	if err != nil {
		return "", err
	}
	if useSalt {
		hash.Write([]byte(configuration.Get().Authentication.SaltFiles))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Generates the SHA1 hash of an uploaded file and returns a reader for the file, the hash and if a temporary file was created the
// reference to that file.
func generateHashAndEncrypt(fileContent io.Reader, fileHeader *multipart.FileHeader) (io.Reader, []byte, *os.File, models.EncryptionInfo) {
	hash := sha1.New()
	encInfo := models.EncryptionInfo{}
	if fileHeader.Size <= int64(configuration.Get().MaxMemory)*1024*1024 {
		content, err := io.ReadAll(fileContent)
		helper.Check(err)
		hash.Write(content)
		if isEncryptionRequested() {
			encContent := new(bytes.Buffer)
			err = encryption.Encrypt(&encInfo, bytes.NewReader(content), encContent)
			helper.Check(err)
			hash.Write([]byte(configuration.Get().Authentication.SaltFiles))
			return bytes.NewReader(encContent.Bytes()), hash.Sum(nil), nil, encInfo
		}
		return bytes.NewReader(content), hash.Sum(nil), nil, encInfo
	}
	tempFile, err := os.CreateTemp(configuration.Get().DataDir, "upload")
	helper.Check(err)
	var multiWriter io.Writer

	multiWriter = io.MultiWriter(tempFile, hash)
	_, err = io.Copy(multiWriter, fileContent)
	helper.Check(err)
	_, err = tempFile.Seek(0, io.SeekStart)
	helper.Check(err)

	if isEncryptionRequested() {
		tempFileEnc, err := os.CreateTemp(configuration.Get().DataDir, "upload")
		helper.Check(err)
		err = encryption.Encrypt(&encInfo, tempFile, tempFileEnc)
		helper.Check(err)
		err = os.Remove(tempFile.Name())
		helper.Check(err)
		hash.Write([]byte(configuration.Get().Authentication.SaltFiles))
		tempFile = tempFileEnc
	}
	// Instead of returning a reference to the file as the 3rd result, one could use reflections. However, that would be more expensive.
	return tempFile, hash.Sum(nil), tempFile, encInfo
}

func isEncryptionRequested() bool {
	switch configuration.Get().Encryption.Level {
	case encryption.NoEncryption:
		return false
	case encryption.LocalEncryptionStored, encryption.LocalEncryptionInput:
		return !aws.IsAvailable()
	case encryption.FullEncryptionStored, encryption.FullEncryptionInput:
		return true
	case encryption.EndToEndEncryption:
		return false
	default:
		log.Fatalln("Unknown encryption level requested")
		return false
	}
}

// imageFileExtensions contains all known image extensions that can be used for hotlinks
var imageFileExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".tiff", ".tif", ".ico", ".avif", ".avifs", ".apng"}

// videoFileExtensions contains all known video extensions that can be used for hotlinks, if enabled with the env var ENABLE_HOTLINK_VIDEOS
var videoFileExtensions = []string{".3gp", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpg", ".mpeg", ".ts", ".webm", ".wmv"}

// AddHotlink will first check if the file may use a hotlink (e.g. not encrypted or password-protected).
// If file is an image, it will generate a new hotlink in the database and add it to the parameter file
// Otherwise no changes will be made
func AddHotlink(file *models.File) {
	if !IsAbleHotlink(*file) {
		return
	}
	link := helper.GenerateRandomString(40) + getFileExtension(file.Name)
	file.HotlinkId = link
	database.SaveHotlink(*file)
}

// IsAbleHotlink returns true, if the file may use hotlinks (e.g. an image file that is not encrypted or password-protected).
func IsAbleHotlink(file models.File) bool {
	if file.RequiresClientDecryption() {
		return false
	}
	if file.PasswordHash != "" {
		return false
	}
	if isPictureFile(file.Name) {
		return true
	}
	env := environment.New()
	if !env.HotlinkVideos {
		return false
	}
	return isVideoFile(file.Name)
}

// getFileExtension returns the file extension of a filename in lowercase
func getFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// isPictureFile returns true if it has one of supported extensions saved in imageFileExtensions
func isPictureFile(filename string) bool {
	extension := getFileExtension(filename)
	return helper.IsInArray(imageFileExtensions, extension)
}

func isVideoFile(filename string) bool {
	extension := getFileExtension(filename)
	return helper.IsInArray(videoFileExtensions, extension)
}

// GetFile gets the file by id. Returns (empty File, false) if invalid / expired file
// or (file, true) if valid file
func GetFile(id string) (models.File, bool) {
	var emptyResult = models.File{}
	if id == "" {
		return emptyResult, false
	}
	file, ok := database.GetMetaDataById(id)
	if !ok {
		return emptyResult, false
	}
	if IsExpiredFile(file, time.Now().Unix()) {
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
	fileId, ok := database.GetHotlink(id)
	if !ok {
		return emptyResult, false
	}
	return GetFile(fileId)
}

// ServeFile subtracts a download allowance and serves the file to the browser
func ServeFile(file models.File, w http.ResponseWriter, r *http.Request, forceDownload bool) {
	file.DownloadsRemaining = file.DownloadsRemaining - 1
	file.DownloadCount = file.DownloadCount + 1
	database.IncreaseDownloadCount(file.Id, !file.UnlimitedDownloads)
	logging.LogDownload(file, r, configuration.Get().SaveIp)
	go sse.PublishDownloadCount(file)

	if !file.IsLocalStorage() {
		// If non-blocking, we are not setting a download complete status as there is no reliable way to
		// confirm that the file has been completely downloaded. It expires automatically after 24 hours.
		statusId := downloadstatus.SetDownload(file)
		isBlocking, err := aws.ServeFile(w, r, file, forceDownload)
		helper.Check(err)
		if isBlocking {
			downloadstatus.SetComplete(statusId)
		}
		return
	}
	fileData, size := getFileHandler(file, configuration.Get().DataDir)
	if file.Encryption.IsEncrypted && !file.RequiresClientDecryption() {
		if !encryption.IsCorrectKey(file.Encryption, fileData) {
			w.Write([]byte("Internal error - Error decrypting file, source data might be damaged or an incorrect key has been used"))
			return
		}
	}
	statusId := downloadstatus.SetDownload(file)
	headers.Write(file, w, forceDownload)
	if file.Encryption.IsEncrypted && !file.RequiresClientDecryption() {
		err := encryption.DecryptReader(file.Encryption, fileData, w)
		if err != nil {
			w.Write([]byte("Error decrypting file"))
			fmt.Println(err)
			return
		}
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		http.ServeContent(w, r, file.Name, time.Now(), fileData)
	}
	downloadstatus.SetComplete(statusId)
}

func getFileHandler(file models.File, dataDir string) (*os.File, int64) {
	storageData, err := os.OpenFile(dataDir+"/"+file.SHA1, os.O_RDONLY, 0644)
	helper.Check(err)
	size, err := helper.GetFileSize(storageData)
	helper.Check(err)
	return storageData, size
}

// FileExists checks if the file exists locally or in S3
func FileExists(file models.File, dataDir string) bool {
	if !file.IsLocalStorage() {
		exists, size, err := aws.FileExists(file)
		if err != nil {
			fmt.Println("Warning, cannot check file " + file.Id + ": " + err.Error())
			return true
		}
		if !exists {
			return false
		}
		if size == 0 && file.Size != "0 B" {
			return false
		}
		return true
	}
	return helper.FileExists(dataDir + "/" + file.SHA1)
}

// CleanUp removes expired files from the config and from the filesystem if they are not referenced by other files anymore
// Will be called periodically or after a file has been manually deleted in the admin view.
// If parameter periodic is true, this function is recursive and calls itself every hour.
func CleanUp(periodic bool) {
	downloadstatus.Clean()
	timeNow := time.Now().Unix()
	wasItemDeleted := false
	for key, element := range database.GetAllMetadata() {
		fileExists := FileExists(element, configuration.Get().DataDir)
		if !fileExists || isExpiredFileWithoutDownload(element, timeNow) {
			deleteFile := true
			for _, secondLoopElement := range database.GetAllMetadata() {
				if (element.Id != secondLoopElement.Id) && (element.SHA1 == secondLoopElement.SHA1) {
					deleteFile = false
				}
			}
			if deleteFile && fileExists {
				deleteSource(element, configuration.Get().DataDir)
			}
			if element.HotlinkId != "" {
				database.DeleteHotlink(element.HotlinkId)
			}
			database.DeleteMetaData(key)
			wasItemDeleted = true
		}
	}
	if wasItemDeleted {
		CleanUp(false)
	}
	cleanOldTempFiles()
	cleanHotlinks()
	database.RunGarbageCollection()

	if periodic {
		go func() {
			select {
			case <-time.After(time.Hour):
				CleanUp(periodic)
			}
		}()
	}
}

// cleanHotlinks removes hotlinks from the database where the file has expired
func cleanHotlinks() {
	hotlinks := database.GetAllHotlinks()
	for _, hotlink := range hotlinks {
		_, ok := GetFileByHotlink(hotlink)
		if !ok {
			database.DeleteHotlink(hotlink)
		}
	}
}

// cleanOldTempFiles removes temporary chunk or upload files that are older than 24 hours
func cleanOldTempFiles() {
	tmpfiles, err := os.ReadDir(configuration.Get().DataDir)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, file := range tmpfiles {
		if isOldTempFile(file) {
			err = os.Remove(configuration.Get().DataDir + "/" + file.Name())
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

// isOldTempFile returns true if a file is older than 24 hours and starts with the name upload or chunk
func isOldTempFile(file os.DirEntry) bool {
	if file.IsDir() {
		return false
	}
	if !strings.HasPrefix(file.Name(), "upload") && !strings.HasPrefix(file.Name(), "chunk-") {
		return false
	}
	info, err := file.Info()
	if err != nil {
		return false
	}
	return time.Now().Sub(info.ModTime()) > 24*time.Hour

}

// IsExpiredFile returns true if the file is expired, either due to download count
// or if the provided timestamp is after the expiry timestamp
func IsExpiredFile(file models.File, timeNow int64) bool {
	return (file.ExpireAt < timeNow && !file.UnlimitedTime) ||
		(file.DownloadsRemaining < 1 && !file.UnlimitedDownloads)
}

// isExpiredFileWithoutDownload returns true if there is no active download for an expired file
func isExpiredFileWithoutDownload(file models.File, timeNow int64) bool {
	if downloadstatus.IsCurrentlyDownloading(file) {
		return false
	}
	return IsExpiredFile(file, timeNow)
}

// deleteSource removes the source file from the file system or cloud storage.
func deleteSource(file models.File, dataDir string) {
	var err error
	if !file.IsLocalStorage() {
		_, err = aws.DeleteObject(file)
	} else {
		err = os.Remove(dataDir + "/" + file.SHA1)
	}
	if err != nil {
		fmt.Println("Warning, cannot delete file " + file.Id + ": " + err.Error())
	}
}

// DeleteFile is called when an admin requests deletion of a file
// Returns true if file was deleted or false if ID did not exist
// deleteSource forces a clean-up and will delete the source if it is not
// used by a different file
func DeleteFile(keyId string, deleteSource bool) bool {
	if keyId == "" {
		return false
	}
	item, ok := database.GetMetaDataById(keyId)
	if !ok {
		return false
	}
	item.ExpireAt = 0
	item.UnlimitedTime = false
	database.SaveMetaData(item)
	downloadstatus.SetAllComplete(item.Id)
	if deleteSource {
		go CleanUp(false)
	}
	return true
}
