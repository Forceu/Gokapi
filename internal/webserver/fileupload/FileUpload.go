package fileupload

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ProcessCompleteFile processes a file upload request
// This is only used when a complete file is uploaded through the API with /files/add
// Normally a file is created from a chunk
func ProcessCompleteFile(w http.ResponseWriter, r *http.Request, userId, maxMemory int) error {
	err := r.ParseMultipartForm(int64(maxMemory) * 1024 * 1024)
	if err != nil {
		return err
	}
	defer r.MultipartForm.RemoveAll()
	config, err := parseConfig(r.Form)
	if err != nil {
		return err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return err
	}

	result, err := storage.NewFile(file, header, userId, config)
	defer file.Close()
	if err != nil {
		return err
	}
	user, _ := database.GetUser(userId)
	logging.LogUpload(result, user)
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl, configuration.Get().IncludeFilename))
	return nil
}

// ProcessNewChunk processes a file chunk upload request
func ProcessNewChunk(w http.ResponseWriter, r *http.Request, isApiCall bool) error {
	err := r.ParseMultipartForm(int64(configuration.Get().MaxMemory) * 1024 * 1024)
	if err != nil {
		return err
	}
	defer r.MultipartForm.RemoveAll()
	chunkInfo, err := chunking.ParseChunkInfo(r, isApiCall)
	if err != nil {
		return err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return err
	}

	err = chunking.NewChunk(file, header, chunkInfo)
	defer file.Close()
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, "{\"result\":\"OK\"}")
	return nil
}

// ParseFileHeader parses the parameters for CompleteChunk()
// This is done as two operations, as CompleteChunk can be blocking too long
// for an HTTP request, by calling this function first, r can be closed afterwards
func ParseFileHeader(r *http.Request) (string, chunking.FileHeader, models.UploadRequest, error) {
	err := r.ParseForm()
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadRequest{}, err
	}
	chunkId := r.Form.Get("chunkid")
	config, err := parseConfig(r.Form)
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadRequest{}, err
	}
	header, err := chunking.ParseFileHeader(r)
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadRequest{}, err
	}
	return chunkId, header, config, nil
}

// CompleteChunk processes a file after all the chunks have been completed
// The parameters can be generated with  ParseFileHeader()
func CompleteChunk(chunkId string, header chunking.FileHeader, userId int, config models.UploadRequest) (models.File, error) {
	return storage.NewFileFromChunk(chunkId, header, userId, config)
}

// CreateUploadConfig populates a new models.UploadRequest struct
func CreateUploadConfig(allowedDownloads, expiryDays int, password string, unlimitedTime, unlimitedDownload, isEnd2End bool, realSize int64) models.UploadRequest {
	settings := configuration.Get()
	return models.UploadRequest{
		AllowedDownloads:    allowedDownloads,
		Expiry:              expiryDays,
		ExpiryTimestamp:     time.Now().Add(time.Duration(expiryDays) * time.Hour * 24).Unix(),
		Password:            password,
		ExternalUrl:         settings.ServerUrl,
		MaxMemory:           settings.MaxMemory,
		UnlimitedTime:       unlimitedTime,
		UnlimitedDownload:   unlimitedDownload,
		IsEndToEndEncrypted: isEnd2End,
		RealSize:            realSize,
	}
}

func parseConfig(values formOrHeader) (models.UploadRequest, error) {
	allowedDownloads := values.Get("allowedDownloads")
	expiryDays := values.Get("expiryDays")
	password := values.Get("password")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	if err != nil {
		allowedDownloadsInt = 1
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		expiryDaysInt = 14
	}

	unlimitedDownload := values.Get("isUnlimitedDownload") == "true"
	unlimitedTime := values.Get("isUnlimitedTime") == "true"

	if allowedDownloadsInt == 0 {
		unlimitedDownload = true
	}
	if expiryDaysInt == 0 {
		unlimitedTime = true
	}

	var isEnd2End bool
	var realSize int64
	if values.Get("isE2E") == "true" {
		isEnd2End = true
		realSizeStr := values.Get("realSize")
		realSize, err = strconv.ParseInt(realSizeStr, 10, 64)
		if err != nil {
			return models.UploadRequest{}, err
		}
	}
	return CreateUploadConfig(allowedDownloadsInt, expiryDaysInt, password, unlimitedTime, unlimitedDownload, isEnd2End, realSize), nil
}

type formOrHeader interface {
	Get(key string) string
}
