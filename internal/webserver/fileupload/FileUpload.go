package fileupload

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
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
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.WriteString(w, result.ToJsonResult(config.ExternalUrl))
	helper.Check(err)
	err = r.MultipartForm.RemoveAll()
	helper.Check(err)
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
		DataDir:           settings.DataDir,
		UnlimitedTime:     unlimitedTime,
		UnlimitedDownload: unlimitedDownload,
	}
}

type formOrHeader interface {
	Get(key string) string
}
