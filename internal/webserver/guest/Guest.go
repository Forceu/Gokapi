package guest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
)

// DeleteToken deletes the selected guest token
func DeleteToken(id string) bool {
	if !IsValidGuestToken(id, false) {
		return false
	}
	database.DeleteGuestToken(id)
	return true
}

// NewToken generates a new guest token
func NewToken() string {
	newToken := models.GuestToken{
		Id:            helper.GenerateRandomString(30),
		TimesUsed:     0,
		UnlimitedTime: true,
		LastUsed:      0,
	}
	database.SaveGuestToken(newToken, false)
	return newToken.Id
}

func list(w http.ResponseWriter) {
	var validFiles []models.FileApiOutput
	timeNow := time.Now().Unix()
	for _, element := range database.GetAllMetadata() {
		if !storage.IsExpiredFile(element, timeNow) {
			file, err := element.ToFileApiOutput()
			helper.Check(err)
			validFiles = append(validFiles, file)
		}
	}
	result, err := json.Marshal(validFiles)
	helper.Check(err)
	_, _ = w.Write(result)
}

func sendError(w http.ResponseWriter, errorInt int, errorMessage string) {
	w.WriteHeader(errorInt)
	_, _ = w.Write([]byte("{\"Result\":\"error\",\"ErrorMessage\":\"" + errorMessage + "\"}"))
}

type apiRequest struct {
	apiKey     string
	requestUrl string
	request    *http.Request
	fileInfo   fileInfo
	apiInfo    apiInfo
}

func (a *apiRequest) parseUploadRequest() error {
	uploadRequest, paramsToChange, filename, err := apiRequestToUploadRequest(a.request)
	if err != nil {
		return err
	}
	a.fileInfo.uploadRequest = uploadRequest
	a.fileInfo.paramsToChange = paramsToChange
	a.fileInfo.filename = filename
	return nil
}

func (a *apiRequest) parseForm() error {
	err := a.request.ParseForm()
	if err != nil {
		return err
	}
	if a.request.Form.Get("id") != "" {
		a.fileInfo.id = a.request.Form.Get("id")
	}
	return nil
}

type fileInfo struct {
	id             string               // apiRequest.parseForm() needs to be called first if id is encoded in form
	uploadRequest  models.UploadRequest // apiRequest.parseUploadRequest() needs to be called first
	paramsToChange int                  // apiRequest.parseUploadRequest() needs to be called first
	filename       string               // apiRequest.parseUploadRequest() needs to be called first
}

type apiInfo struct {
	friendlyName   string
	apiKeyToModify string
}

func parseRequest(r *http.Request) apiRequest {
	return apiRequest{
		apiKey:     r.Header.Get("apikey"),
		requestUrl: strings.Replace(r.URL.String(), "/api", "", 1),
		request:    r,
		fileInfo:   fileInfo{id: r.Header.Get("id")},
		apiInfo: apiInfo{
			friendlyName:   r.Header.Get("friendlyName"),
			apiKeyToModify: r.Header.Get("apiKeyToModify")},
	}
}

func apiRequestToUploadRequest(request *http.Request) (models.UploadRequest, int, string, error) {
	paramsToChange := 0
	allowedDownloads := 0
	daysExpiry := 0
	unlimitedTime := false
	unlimitedDownloads := false
	password := ""
	fileName := ""

	err := request.ParseForm()
	if err != nil {
		return models.UploadRequest{}, 0, "", err
	}

	if request.Form.Get("allowedDownloads") != "" {
		paramsToChange = paramsToChange | storage.ParamDownloads
		allowedDownloads, err = strconv.Atoi(request.Form.Get("allowedDownloads"))
		if err != nil {
			return models.UploadRequest{}, 0, "", err
		}
		if allowedDownloads == 0 {
			unlimitedDownloads = true
		}
	}

	if request.Form.Get("expiryDays") != "" {
		paramsToChange = paramsToChange | storage.ParamExpiry
		daysExpiry, err = strconv.Atoi(request.Form.Get("expiryDays"))
		if err != nil {
			return models.UploadRequest{}, 0, "", err
		}
		if daysExpiry == 0 {
			unlimitedTime = true
		}
	}

	if strings.ToLower(request.Form.Get("originalPassword")) != "true" {
		paramsToChange = paramsToChange | storage.ParamPassword
		password = request.Form.Get("password")
	}

	if request.Form.Get("filename") != "" {
		paramsToChange = paramsToChange | storage.ParamName
		fileName = request.Form.Get("filename")
	}

	return models.UploadRequest{
		AllowedDownloads:  allowedDownloads,
		Expiry:            daysExpiry,
		UnlimitedTime:     unlimitedTime,
		UnlimitedDownload: unlimitedDownloads,
		Password:          password,
		ExpiryTimestamp:   time.Now().Add(time.Duration(daysExpiry) * time.Hour * 24).Unix(),
	}, paramsToChange, fileName, nil
}

// IsValidGuestToken checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func IsValidGuestToken(token string, modifyTime bool) bool {
	if token == "" {
		return false
	}
	savedToken, ok := database.GetGuestToken(token)
	if ok && savedToken.Id != "" {
		if modifyTime {
			savedToken.LastUsed = time.Now().Unix()
			database.SaveGuestToken(savedToken, true)
		}
		return true
	}
	return false
}
