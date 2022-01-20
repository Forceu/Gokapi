package fileupload

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"Gokapi/internal/storage"
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
		previous, _, _ := dataStorage.GetUploadDefaults()
		allowedDownloadsInt = previous
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		_, previous, _ := dataStorage.GetUploadDefaults()
		expiryDaysInt = previous
	}
	if setNewDefaults {
		dataStorage.SaveUploadDefaults(allowedDownloadsInt, expiryDaysInt, password)
	}
	settings := configuration.Get()
	return models.UploadRequest{
		AllowedDownloads: allowedDownloadsInt,
		Expiry:           expiryDaysInt,
		ExpiryTimestamp:  time.Now().Add(time.Duration(expiryDaysInt) * time.Hour * 24).Unix(),
		Password:         password,
		ExternalUrl:      settings.ServerUrl,
		MaxMemory:        settings.MaxMemory,
		DataDir:          settings.DataDir,
	}
}

type formOrHeader interface {
	Get(key string) string
}
