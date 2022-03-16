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
	var validFiles []models.File
	timeNow := time.Now().Unix()
	for _, element := range database.GetAllMetadata() {
		if !storage.IsExpiredFile(element, timeNow) {
			validFiles = append(validFiles, element)
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
