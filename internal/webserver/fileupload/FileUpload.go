package fileupload

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"github.com/forceu/gokapi/internal/storage/chunking/chunkreservation"
	"github.com/forceu/gokapi/internal/webserver/errorHandling/errorcodes"
)

const minChunkSize = 5 * 1024 * 1024
const minChunkSizeLowMaxChunk = 1 * 1024 * 1024

// ErrorExpiryTooLong is returned when the requested expiry exceeds the server's maximum (GOKAPI_MAX_EXPIRY_SECONDS)
var ErrorExpiryTooLong = errors.New("expiry exceeds the server's maximum")

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

	config.FileRequestId = ""
	result, err := storage.NewFile(file, header, userId, config)
	defer file.Close()
	if err != nil {
		return err
	}
	user, _ := database.GetUser(userId)
	logging.LogUpload(result, user, models.FileRequest{})
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl, configuration.Get().IncludeFilename))
	return nil
}

func isChunkMinChunkSize(r *http.Request, offset, fileSize int64) bool {
	minReqChunkSize := minChunkSize
	if configuration.Get().ChunkSize < 5 {
		minReqChunkSize = minChunkSizeLowMaxChunk
	}
	if r.ContentLength >= int64(minReqChunkSize) {
		return true
	}
	if r.ContentLength >= (fileSize - offset) {
		return true
	}
	return false
}

// ProcessNewChunk processes a file chunk upload request
func ProcessNewChunk(w http.ResponseWriter, r *http.Request, isApiCall bool, filerequestId string, maxFileSize int64) (int, error) {
	err := r.ParseMultipartForm(int64(configuration.Get().MaxMemory) * 1024 * 1024)
	if err != nil {
		return errorcodes.CannotParse, err
	}
	defer r.MultipartForm.RemoveAll()
	chunkInfo, err := chunking.ParseChunkInfo(r, isApiCall)
	if err != nil {
		return errorcodes.InvalidUserInput, err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return errorcodes.InvalidUserInput, err
	}

	if !isChunkMinChunkSize(r, chunkInfo.Offset, chunkInfo.TotalFilesizeBytes) {
		return errorcodes.ChunkTooSmall, storage.ErrorChunkTooSmall
	}
	if chunkInfo.TotalFilesizeBytes > maxFileSize || chunkInfo.Offset > maxFileSize {
		return errorcodes.FileTooLarge, storage.ErrorFileTooLarge
	}

	if filerequestId != "" {
		if !chunkreservation.SetUploading(filerequestId, chunkInfo.UUID) {
			return errorcodes.InvalidChunkReservation, errors.New("chunk reservation has expired or was not requested")
		}
	}

	err = chunking.NewChunk(file, header, chunkInfo, maxFileSize)
	defer file.Close()
	if err != nil {
		return errorcodes.CannotAllocateFile, err
	}
	_, _ = io.WriteString(w, "{\"result\":\"OK\"}")
	return 0, nil
}

// ParseFileHeader parses the parameters for CompleteChunk()
// This is done as two operations, as CompleteChunk can be blocking too long
// for an HTTP request, by calling this function first, r can be closed afterwards
func ParseFileHeader(r *http.Request) (string, chunking.FileHeader, models.UploadParameters, error) {
	err := r.ParseForm()
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadParameters{}, err
	}
	chunkId := r.PostForm.Get("chunkid")
	config, err := parseConfig(r.Form)
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadParameters{}, err
	}
	header, err := chunking.ParseFileHeader(r)
	if err != nil {
		return "", chunking.FileHeader{}, models.UploadParameters{}, err
	}
	return chunkId, header, config, nil
}

// CompleteChunk processes a file after all the chunks have been completed
// The parameters can be generated with  ParseFileHeader()
func CompleteChunk(chunkId string, header chunking.FileHeader, userId int, config models.UploadParameters) (models.File, error) {
	return storage.NewFileFromChunk(chunkId, header, userId, config)
}

// CreateUploadConfig populates a new models.UploadParameters struct
func CreateUploadConfig(allowedDownloads, expiryDays int, password string, unlimitedTime, unlimitedDownload, isEnd2End bool, realSize int64, fileRequestId string) models.UploadParameters {
	settings := configuration.Get()
	return models.UploadParameters{
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
		FileRequestId:       fileRequestId,
	}
}

// CheckMaxExpiry returns an error wrapping ErrorExpiryTooLong if the server limits the lifetime of files
// and the given expiry is unlimited or later than the limit allows
func CheckMaxExpiry(unlimitedTime bool, expiryTimestamp int64) error {
	maxSeconds := configuration.GetEnvironment().MaxExpirySeconds
	if maxSeconds == 0 {
		return nil
	}
	if unlimitedTime || expiryTimestamp > time.Now().Unix()+maxSeconds {
		return fmt.Errorf("%w of %d seconds", ErrorExpiryTooLong, maxSeconds)
	}
	return nil
}

// CapToMaxExpiry reduces an expiry that was not chosen by the uploader (the default expiry, or the
// unlimited expiry of file request uploads) to the server's maximum lifetime of files, if one is set
func CapToMaxExpiry(config *models.UploadParameters) {
	maxSeconds := configuration.GetEnvironment().MaxExpirySeconds
	if maxSeconds == 0 {
		return
	}
	maxTimestamp := time.Now().Unix() + maxSeconds
	if config.UnlimitedTime || config.ExpiryTimestamp > maxTimestamp {
		config.UnlimitedTime = false
		config.ExpiryTimestamp = maxTimestamp
	}
}

func parseConfig(values formOrHeader) (models.UploadParameters, error) {
	fileRequestId := values.Get("fileRequestId")
	if fileRequestId != "" {
		config := CreateUploadConfig(0, 0, "",
			true, true, false, 0, fileRequestId)
		CapToMaxExpiry(&config)
		return config, nil
	}
	allowedDownloads := values.Get("allowedDownloads")
	expiryDays := values.Get("expiryDays")
	password := values.Get("password")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	if err != nil {
		allowedDownloadsInt = 1
	}
	isExpiryRequested := true
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		expiryDaysInt = 14
		isExpiryRequested = false
	}

	unlimitedDownload := values.Get("isUnlimitedDownload") == "true"
	unlimitedTime := values.Get("isUnlimitedTime") == "true"
	if unlimitedTime {
		isExpiryRequested = true
	}

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
			return models.UploadParameters{}, err
		}
	}
	config := CreateUploadConfig(allowedDownloadsInt, expiryDaysInt, password, unlimitedTime, unlimitedDownload, isEnd2End, realSize, "")
	if !isExpiryRequested {
		CapToMaxExpiry(&config)
		return config, nil
	}
	err = CheckMaxExpiry(config.UnlimitedTime, config.ExpiryTimestamp)
	if err != nil {
		return models.UploadParameters{}, err
	}
	return config, nil
}

type formOrHeader interface {
	Get(key string) string
}
