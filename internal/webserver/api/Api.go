package api

import (
	"encoding/json"
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request, maxMemory int) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	request := parseRequest(r)
	if !isAuthorisedForApi(w, request) {
		return
	}
	switch request.requestUrl {
	case "/chunk/add":
		chunkAdd(w, request)
	case "/chunk/complete":
		chunkComplete(w, request)
	case "/files/list":
		list(w)
	case "/files/add":
		upload(w, request, maxMemory)
	case "/files/delete":
		deleteFile(w, request)
	case "/files/duplicate":
		duplicateFile(w, request)
	case "/files/modify":
		editFile(w, request)
	case "/auth/create":
		createApiKey(w, request)
	case "/auth/friendlyname":
		changeFriendlyName(w, request)
	case "/auth/modify":
		modifyApiPermission(w, request)
	case "/auth/delete":
		deleteApiKey(w, request)
	default:
		sendError(w, http.StatusBadRequest, "Invalid request")
	}
}

func editFile(w http.ResponseWriter, request apiRequest) {
	file, ok := database.GetMetaDataById(request.filemodInfo.id)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid file ID provided.")
		return
	}
	if request.filemodInfo.downloads != "" {
		downloadsInt, err := strconv.Atoi(request.filemodInfo.downloads)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid download count provided.")
			return
		}
		if downloadsInt != 0 {
			file.DownloadsRemaining = downloadsInt
			file.UnlimitedDownloads = false
		} else {
			file.UnlimitedDownloads = true
		}
	}
	if request.filemodInfo.expiry != "" {
		expiryInt, err := strconv.ParseInt(request.filemodInfo.expiry, 10, 64)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid expiry timestamp provided.")
			return
		}
		if expiryInt != 0 {
			file.ExpireAt = expiryInt
			file.ExpireAtString = storage.FormatTimestamp(expiryInt)
			file.UnlimitedTime = false
		} else {
			file.UnlimitedTime = true
		}
	}

	if !request.filemodInfo.originalPassword {
		file.PasswordHash = configuration.HashPassword(request.filemodInfo.password, true)
	}

	if file.HotlinkId != "" && !storage.IsAbleHotlink(file) {
		database.DeleteHotlink(file.HotlinkId)
		file.HotlinkId = ""
	} else if file.HotlinkId == "" && storage.IsAbleHotlink(file) {
		storage.AddHotlink(&file)
	}

	database.SaveMetaData(file)
	outputFileInfo(w, file)
}

func getApiPermissionRequired(requestUrl string) (uint8, bool) {
	switch requestUrl {
	case "/chunk/add":
		return models.ApiPermUpload, true
	case "/chunk/complete":
		return models.ApiPermUpload, true
	case "/files/list":
		return models.ApiPermView, true
	case "/files/add":
		return models.ApiPermUpload, true
	case "/files/delete":
		return models.ApiPermDelete, true
	case "/files/duplicate":
		return models.ApiPermUpload, true
	case "/files/modify":
		return models.ApiPermEdit, true
	case "/auth/create":
		return models.ApiPermApiMod, true
	case "/auth/friendlyname":
		return models.ApiPermApiMod, true
	case "/auth/modify":
		return models.ApiPermApiMod, true
	case "/auth/delete":
		return models.ApiPermApiMod, true
	default:
		return models.ApiPermNone, false
	}
}

// DeleteKey deletes the selected API key
func DeleteKey(id string) bool {
	if !IsValidApiKey(id, false, models.ApiPermNone) {
		return false
	}
	database.DeleteApiKey(id)
	return true
}

// NewKey generates a new API key
func NewKey(defaultPermissions bool) string {
	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(30),
		FriendlyName: "Unnamed key",
		LastUsed:     0,
		Permissions:  models.ApiPermAllNoApiMod,
		Expiry:       0,
		IsSystemKey:  false,
	}
	if !defaultPermissions {
		newKey.Permissions = models.ApiPermNone
	}
	database.SaveApiKey(newKey)
	return newKey.Id
}

// newSystemKey generates a new API key that is only used internally for the GUI
// and will be valid for 48 hours
func newSystemKey() string {
	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(30),
		FriendlyName: "Internal System Key",
		LastUsed:     0,
		Permissions:  models.ApiPermAll,
		Expiry:       time.Now().Add(time.Hour * 48).Unix(),
		IsSystemKey:  true,
	}
	database.SaveApiKey(newKey)
	return newKey.Id
}

// GetSystemKey returns the latest System API key or generates a new one, if none exists or the current one expires
// within the next 24 hours
func GetSystemKey() string {
	key, ok := database.GetSystemKey()
	if !ok || key.Expiry < time.Now().Add(time.Hour*24).Unix() {
		return newSystemKey()
	}
	return key.Id
}

func deleteApiKey(w http.ResponseWriter, request apiRequest) {
	if !isValidKeyForEditing(w, request) {
		return
	}
	database.DeleteApiKey(request.apiInfo.apiKeyToModify)
}

func modifyApiPermission(w http.ResponseWriter, request apiRequest) {
	if !isValidKeyForEditing(w, request) {
		return
	}
	if request.apiInfo.permission < models.ApiPermView || request.apiInfo.permission > models.ApiPermEdit {
		sendError(w, http.StatusBadRequest, "Invalid permission sent")
		return
	}
	key, ok := database.GetApiKey(request.apiInfo.apiKeyToModify)
	if !ok {
		sendError(w, http.StatusInternalServerError, "Could not modify API key")
		return
	}
	if request.apiInfo.grantPermission && !key.HasPermission(request.apiInfo.permission) {
		key.SetPermission(request.apiInfo.permission)
		database.SaveApiKey(key)
		return
	}
	if !request.apiInfo.grantPermission && key.HasPermission(request.apiInfo.permission) {
		key.RemovePermission(request.apiInfo.permission)
		database.SaveApiKey(key)
	}
}

func isValidKeyForEditing(w http.ResponseWriter, request apiRequest) bool {
	if !IsValidApiKey(request.apiInfo.apiKeyToModify, false, models.ApiPermNone) {
		sendError(w, http.StatusBadRequest, "Invalid api key provided.")
		return false
	}
	return true
}

func createApiKey(w http.ResponseWriter, request apiRequest) {
	key := NewKey(request.apiInfo.basicPermissions)
	output := models.ApiKeyOutput{
		Result: "OK",
		Id:     key,
	}
	if request.apiInfo.friendlyName != "" {
		err := renameApiKeyFriendlyName(key, request.apiInfo.friendlyName)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	result, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(result)
}

func changeFriendlyName(w http.ResponseWriter, request apiRequest) {
	if !isValidKeyForEditing(w, request) {
		return
	}
	err := renameApiKeyFriendlyName(request.apiInfo.apiKeyToModify, request.apiInfo.friendlyName)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
	}
}

func renameApiKeyFriendlyName(id string, newName string) error {
	if newName == "" {
		newName = "Unnamed key"
	}
	key, ok := database.GetApiKey(id)
	if !ok {
		return errors.New("could not modify API key")
	}
	if key.FriendlyName != newName {
		key.FriendlyName = newName
		database.SaveApiKey(key)
	}
	return nil
}

func deleteFile(w http.ResponseWriter, request apiRequest) {
	ok := storage.DeleteFile(request.fileInfo.id, true)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid file ID provided.")
	}
}

func chunkAdd(w http.ResponseWriter, request apiRequest) {
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, storage.ErrorFileTooLarge.Error())
		return
	}

	request.request.Body = http.MaxBytesReader(w, request.request.Body, maxUpload)
	err := fileupload.ProcessNewChunk(w, request.request, true)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
	}
}
func chunkComplete(w http.ResponseWriter, request apiRequest) {
	err := request.request.ParseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	request.request.Form.Set("chunkid", request.request.Form.Get("uuid"))
	err = fileupload.CompleteChunk(w, request.request, true)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
	}
}

func list(w http.ResponseWriter) {
	var validFiles []models.FileApiOutput
	timeNow := time.Now().Unix()
	config := configuration.Get()
	for _, element := range database.GetAllMetadata() {
		if !storage.IsExpiredFile(element, timeNow) {
			file, err := element.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
			helper.Check(err)
			validFiles = append(validFiles, file)
		}
	}
	result, err := json.Marshal(validFiles)
	helper.Check(err)
	_, _ = w.Write(result)
}

func upload(w http.ResponseWriter, request apiRequest, maxMemory int) {
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, storage.ErrorFileTooLarge.Error())
		return
	}

	request.request.Body = http.MaxBytesReader(w, request.request.Body, maxUpload)
	err := fileupload.Process(w, request.request, false, maxMemory)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
}

func duplicateFile(w http.ResponseWriter, request apiRequest) {
	err := request.parseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, ok := storage.GetFile(request.fileInfo.id)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid id provided.")
		return
	}
	err = request.parseUploadRequest()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	newFile, err := storage.DuplicateFile(file, request.fileInfo.paramsToChange, request.fileInfo.filename, request.fileInfo.uploadRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	outputFileInfo(w, newFile)
}

func outputFileInfo(w http.ResponseWriter, file models.File) {
	config := configuration.Get()
	publicOutput, err := file.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
	helper.Check(err)
	result, err := json.Marshal(publicOutput)
	helper.Check(err)
	_, _ = w.Write(result)
}

func isAuthorisedForApi(w http.ResponseWriter, request apiRequest) bool {
	perm, ok := getApiPermissionRequired(request.requestUrl)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid request")
		return false
	}
	if IsValidApiKey(request.apiKey, true, perm) {
		return true
	}
	sendError(w, http.StatusUnauthorized, "Unauthorized")
	return false
}

// Probably from new API permission system
func sendError(w http.ResponseWriter, errorInt int, errorMessage string) {
	w.WriteHeader(errorInt)
	_, _ = w.Write([]byte("{\"Result\":\"error\",\"ErrorMessage\":\"" + errorMessage + "\"}"))
}

type apiRequest struct {
	apiKey      string
	requestUrl  string
	request     *http.Request
	fileInfo    fileInfo
	apiInfo     apiInfo
	filemodInfo filemodInfo
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
	friendlyName     string
	apiKeyToModify   string
	permission       uint8
	grantPermission  bool
	basicPermissions bool
}
type filemodInfo struct {
	id               string
	downloads        string
	expiry           string
	password         string
	originalPassword bool
}

func parseRequest(r *http.Request) apiRequest {
	permission := models.ApiPermNone
	switch r.Header.Get("permission") {
	case "PERM_VIEW":
		permission = models.ApiPermView
	case "PERM_UPLOAD":
		permission = models.ApiPermUpload
	case "PERM_DELETE":
		permission = models.ApiPermDelete
	case "PERM_API_MOD":
		permission = models.ApiPermApiMod
	case "PERM_EDIT":
		permission = models.ApiPermEdit
	}
	return apiRequest{
		apiKey:     r.Header.Get("apikey"),
		requestUrl: strings.Replace(r.URL.String(), "/api", "", 1),
		request:    r,
		fileInfo:   fileInfo{id: r.Header.Get("id")},
		filemodInfo: filemodInfo{
			id:               r.Header.Get("id"),
			downloads:        r.Header.Get("allowedDownloads"),
			expiry:           r.Header.Get("expiryTimestamp"),
			password:         r.Header.Get("password"),
			originalPassword: r.Header.Get("originalPassword") == "true",
		},
		apiInfo: apiInfo{
			friendlyName:     r.Header.Get("friendlyName"),
			apiKeyToModify:   r.Header.Get("apiKeyToModify"),
			permission:       uint8(permission),
			grantPermission:  r.Header.Get("permissionModifier") == "GRANT",
			basicPermissions: r.Header.Get("basicPermissions") == "true",
		},
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

// IsValidApiKey checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func IsValidApiKey(key string, modifyTime bool, requiredPermission uint8) bool {
	if key == "" {
		return false
	}
	savedKey, ok := database.GetApiKey(key)
	if ok && savedKey.Id != "" && (savedKey.Expiry == 0 || savedKey.Expiry > time.Now().Unix()) {
		if modifyTime {
			savedKey.LastUsed = time.Now().Unix()
			database.UpdateTimeApiKey(savedKey)
		}
		return savedKey.HasPermission(requiredPermission)
	}
	return false
}
