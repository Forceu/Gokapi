package api

import (
	"encoding/json"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:generate cp ../../../openapi.json ../web/static/apidocumentation/
//go:generate echo "Copied openapi.json"

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request, maxMemory int) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	request := parseRequest(r)
	if !isAuthorisedForApi(w, request) {
		return
	}
	switch request.requestUrl {
	case "/files/list":
		list(w)
	case "/files/add":
		upload(w, request, maxMemory)
	case "/files/delete":
		deleteFile(w, request)
	case "/files/duplicate":
		duplicateFile(w, request)
	case "/auth/friendlyname":
		changeFriendlyName(w, request)
	default:
		sendError(w, http.StatusBadRequest, "Invalid request")
	}
}

// DeleteKey deletes the selected API key
func DeleteKey(id string) bool {
	if !IsValidApiKey(id, false) {
		return false
	}
	database.DeleteApiKey(id)
	return true
}

// NewKey generates a new API key
func NewKey() string {
	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(30),
		FriendlyName: "Unnamed key",
		LastUsed:     0,
	}
	database.SaveApiKey(newKey, false)
	return newKey.Id
}

func changeFriendlyName(w http.ResponseWriter, request apiRequest) {
	if !IsValidApiKey(request.apiKeyToModify, false) {
		sendError(w, http.StatusBadRequest, "Invalid api key provided.")
		return
	}
	if request.friendlyName == "" {
		request.friendlyName = "Unnamed key"
	}
	key, ok := database.GetApiKey(request.apiKeyToModify)
	if !ok {
		sendError(w, http.StatusInternalServerError, "Could not modify API key")
		return
	}
	if key.FriendlyName != request.friendlyName {
		key.FriendlyName = request.friendlyName
		database.SaveApiKey(key, false)
	}
}

func deleteFile(w http.ResponseWriter, request apiRequest) {
	ok := storage.DeleteFile(request.fileId, true)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid id provided.")
	}
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

func upload(w http.ResponseWriter, request apiRequest, maxMemory int) {
	err := fileupload.Process(w, request.request, false, maxMemory)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
}

func duplicateFile(w http.ResponseWriter, request apiRequest) {

	err := request.request.ParseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, ok := storage.GetFile(request.request.Form.Get("id"))
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid id provided.")
		return
	}
	uploadRequest, paramsToChange, filename, err := apiRequestToUploadRequest(request.request)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	newFile, err := storage.DuplicateFile(file, paramsToChange, filename, uploadRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	publicOutput, err := newFile.ToFileApiOutput()
	helper.Check(err)
	result, err := json.Marshal(publicOutput)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiRequestToUploadRequest(request *http.Request) (models.UploadRequest, int, string, error) {
	paramsToChange := 0
	allowedDownloads := 0
	daysExpiry := 0
	unlimitedTime := false
	unlimitedDownloads := false
	password := ""
	fileName := ""
	var err error

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

func isAuthorisedForApi(w http.ResponseWriter, request apiRequest) bool {
	if IsValidApiKey(request.apiKey, true) || sessionmanager.IsValidSession(w, request.request) {
		return true
	}
	sendError(w, http.StatusUnauthorized, "Unauthorized")
	return false
}

func sendError(w http.ResponseWriter, errorInt int, errorMessage string) {
	w.WriteHeader(errorInt)
	_, _ = w.Write([]byte("{\"Result\":\"error\",\"ErrorMessage\":\"" + errorMessage + "\"}"))
}

type apiRequest struct {
	apiKey         string
	requestUrl     string
	fileId         string
	friendlyName   string
	apiKeyToModify string
	request        *http.Request
}

func parseRequest(r *http.Request) apiRequest {
	return apiRequest{
		apiKey:         r.Header.Get("apikey"),
		fileId:         r.Header.Get("id"),
		friendlyName:   r.Header.Get("friendlyName"),
		apiKeyToModify: r.Header.Get("apiKeyToModify"),
		requestUrl:     strings.Replace(r.URL.String(), "/api", "", 1),
		request:        r,
	}
}

// IsValidApiKey checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func IsValidApiKey(key string, modifyTime bool) bool {
	if key == "" {
		return false
	}
	savedKey, ok := database.GetApiKey(key)
	if ok && savedKey.Id != "" {
		if modifyTime {
			savedKey.LastUsed = time.Now().Unix()
			database.SaveApiKey(savedKey, true)
		}
		return true
	}
	return false
}
