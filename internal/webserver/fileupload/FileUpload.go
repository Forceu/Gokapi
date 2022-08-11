package fileupload

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Process processes a file upload request
func Process(w http.ResponseWriter, r *http.Request, isWeb bool, maxMemory int) error {
	err := r.ParseMultipartForm(int64(maxMemory) * 1024 * 1024)
	if err != nil {
		return err
	}
	defer r.MultipartForm.RemoveAll()
	config, err := parseConfig(r.Form, isWeb)
	if err != nil {
		return err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return err
	}

	result, err := storage.NewFile(file, header, config)
	defer file.Close()
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl, storage.RequiresClientDecryption(result)))
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

// CompleteChunk processes a file after all the chunks have been completed
func CompleteChunk(w http.ResponseWriter, r *http.Request, isApiCall bool) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}
	chunkId := r.Form.Get("chunkid")
	config, err := parseConfig(r.Form, !isApiCall)
	if err != nil {
		return err
	}
	header, err := chunking.ParseFileHeader(r)
	if err != nil {
		return err
	}

	result, err := storage.NewFileFromChunk(chunkId, header, config)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl, storage.RequiresClientDecryption(result)))
	return nil
}

func parseConfig(values formOrHeader, setNewDefaults bool) (models.UploadRequest, error) {
	allowedDownloads := values.Get("allowedDownloads")
	expiryDays := values.Get("expiryDays")
	password := values.Get("password")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	if err != nil {
		previousValues := database.GetUploadDefaults()
		allowedDownloadsInt = previousValues.Downloads
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		previousValues := database.GetUploadDefaults()
		expiryDaysInt = previousValues.TimeExpiry
	}

	unlimitedDownload := values.Get("isUnlimitedDownload") == "true"
	unlimitedTime := values.Get("isUnlimitedTime") == "true"

	if allowedDownloadsInt == 0 {
		unlimitedDownload = true
	}
	if expiryDaysInt == 0 {
		unlimitedTime = true
	}

	if setNewDefaults {
		values := models.LastUploadValues{
			Downloads:         allowedDownloadsInt,
			TimeExpiry:        expiryDaysInt,
			Password:          password,
			UnlimitedDownload: unlimitedDownload,
			UnlimitedTime:     unlimitedTime,
		}
		database.SaveUploadDefaults(values)
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
	settings := configuration.Get()
	return models.UploadRequest{
		AllowedDownloads:    allowedDownloadsInt,
		Expiry:              expiryDaysInt,
		ExpiryTimestamp:     time.Now().Add(time.Duration(expiryDaysInt) * time.Hour * 24).Unix(),
		Password:            password,
		ExternalUrl:         settings.ServerUrl,
		MaxMemory:           settings.MaxMemory,
		UnlimitedTime:       unlimitedTime,
		UnlimitedDownload:   unlimitedDownload,
		IsEndToEndEncrypted: isEnd2End,
		RealSize:            realSize,
	}, nil
}

type formOrHeader interface {
	Get(key string) string
}
