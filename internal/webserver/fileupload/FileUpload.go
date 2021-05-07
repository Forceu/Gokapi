package fileupload

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"Gokapi/internal/storage"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Process processes a file upload request
func Process(w http.ResponseWriter, r *http.Request, isWeb bool) error {
	err := r.ParseMultipartForm(20 * 1024 * 1024)
	if err != nil {
		return err
	}
	var config models.UploadRequest
	if isWeb {
		config = parseConfig(r.Form, true)
	} else {
		config = parseConfig(r.Header, false)
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
	if err != nil {
		helper.Check(err)
	}
	return nil
}

func parseConfig(values formOrHeader, setNewDefaults bool) models.UploadRequest {
	allowedDownloads := values.Get("allowedDownloads")
	expiryDays := values.Get("expiryDays")
	password := values.Get("password")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	settings := configuration.GetServerSettings()
	if err != nil {
		allowedDownloadsInt = settings.DefaultDownloads
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		expiryDaysInt = settings.DefaultExpiry
	}
	if setNewDefaults {
		settings.DefaultExpiry = expiryDaysInt
		settings.DefaultDownloads = allowedDownloadsInt
		settings.DefaultPassword = password
	}
	externalUrl := settings.ServerUrl
	configuration.Release()
	return models.UploadRequest{
		AllowedDownloads: allowedDownloadsInt,
		Expiry:           expiryDaysInt,
		ExpiryTimestamp:  time.Now().Add(time.Duration(expiryDaysInt) * time.Hour * 24).Unix(),
		Password:         password,
		ExternalUrl:      externalUrl,
	}
}

type formOrHeader interface {
	Get(key string) string
}
