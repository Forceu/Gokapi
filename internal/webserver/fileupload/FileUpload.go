package fileupload

import (
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
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
	defer r.MultipartForm.RemoveAll()
	if err != nil {
		return err
	}
	var config models.UploadRequest
	if isWeb {
		config = parseConfig(r.Form, true)
	} else {
		config = parseConfig(r.Form, false)
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
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl))
	return nil
}

// ProcessChunk processes a file chunk upload request
func ProcessChunk(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseMultipartForm(int64(configuration.Get().MaxMemory) * 1024 * 1024)
	defer r.MultipartForm.RemoveAll()
	if err != nil {
		return err
	}
	chunkInfo, err := chunking.ParseChunkInfo(r)
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

func CompleteChunk(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}
	chunkId := r.Form.Get("chunkid")
	if chunkId == "" {
		return errors.New("empty chunk id provided")
	}
	if !helper.FileExists(configuration.Get().DataDir + "/chunk-" + chunkId) {
		return errors.New("chunk file does not exist")
	}
	config := parseConfig(r.Form, true)
	header, err := chunking.ParseFileHeader(r)
	if err != nil {
		return err
	}

	result, err := storage.NewFileFromChunk(chunkId, header, config)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, result.ToJsonResult(config.ExternalUrl))
	return nil
}

func parseConfig(values formOrHeader, setNewDefaults bool) models.UploadRequest {
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
	settings := configuration.Get()
	return models.UploadRequest{
		AllowedDownloads:  allowedDownloadsInt,
		Expiry:            expiryDaysInt,
		ExpiryTimestamp:   time.Now().Add(time.Duration(expiryDaysInt) * time.Hour * 24).Unix(),
		Password:          password,
		ExternalUrl:       settings.ServerUrl,
		MaxMemory:         settings.MaxMemory,
		UnlimitedTime:     unlimitedTime,
		UnlimitedDownload: unlimitedDownload,
	}
}

type formOrHeader interface {
	Get(key string) string
}
