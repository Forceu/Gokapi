package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/logging/serverstats"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"github.com/forceu/gokapi/internal/storage/chunking/chunkreservation"
	"github.com/forceu/gokapi/internal/storage/filerequest"
	"github.com/forceu/gokapi/internal/storage/presign"
	"github.com/forceu/gokapi/internal/webserver/api/errorcodes"
	"github.com/forceu/gokapi/internal/webserver/authentication/users"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"github.com/forceu/gokapi/internal/webserver/ratelimiter"
)

// LengthPublicId is the length of the public ID used for API keys
const LengthPublicId = 35

// LengthApiKey is the length of the private API key used for authentication
const LengthApiKey = 30

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	requestUrl := parseRequestUrl(r)

	routing, ok := getRouting(requestUrl)
	if !ok {
		sendError(w, http.StatusBadRequest, errorcodes.InvalidUrl, "Invalid request")
		return
	}
	var user models.User
	user, ok = isAuthorisedForApi(r, routing)
	if !ok {
		sendError(w, http.StatusUnauthorized, errorcodes.InvalidApiKey, "Unauthorized")
		return
	}
	if routing.AdminOnly && !user.IsAdmin() {
		sendError(w, http.StatusUnauthorized, errorcodes.AdminOnly, "Unauthorized")
		return
	}
	if routing.RequestParser == nil {
		routing.Continue(w, nil, user)
		return
	}
	parser := routing.RequestParser.New()
	err := parser.ParseRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, errorcodes.CannotParse, err.Error())
		return
	}
	routing.Continue(w, parser, user)
}

func parseRequestUrl(r *http.Request) string {
	return strings.Replace(r.URL.String(), "/api", "", 1)
}

func apiEditFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesModify)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := database.GetMetaDataById(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermEditOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to edit file.")
		return
	}
	if request.UnlimitedDownloads {
		file.UnlimitedDownloads = true
	} else {
		if request.AllowedDownloads != 0 {
			file.DownloadsRemaining = request.AllowedDownloads
			file.UnlimitedDownloads = false
		}
	}
	if request.UnlimitedExpiry {
		file.UnlimitedTime = true
	} else {
		if request.ExpiryTimestamp != 0 {
			file.ExpireAt = request.ExpiryTimestamp
			file.UnlimitedTime = false
		}
	}

	if !request.KeepPassword {
		file.PasswordHash = configuration.HashPassword(request.Password, true)
	}

	if file.HotlinkId != "" && !storage.IsAbleHotlink(file) {
		database.DeleteHotlink(file.HotlinkId)
		file.HotlinkId = ""
	} else if file.HotlinkId == "" && storage.IsAbleHotlink(file) {
		storage.AddHotlink(&file)
	}

	database.SaveMetaData(file)
	logging.LogEdit(file, user)
	outputFileApiInfo(w, file)
}

// generateNewKey generates and saves a new API key
func generateNewKey(defaultPermissions bool, userId int, friendlyName, filerequstId string) models.ApiKey {
	if friendlyName == "" {
		friendlyName = "Unnamed key"
	}
	newKey := models.ApiKey{
		Id:              helper.GenerateRandomString(LengthApiKey),
		PublicId:        helper.GenerateRandomString(LengthPublicId),
		FriendlyName:    friendlyName,
		Permissions:     models.ApiPermDefault,
		IsSystemKey:     false,
		UserId:          userId,
		UploadRequestId: filerequstId,
	}
	if !defaultPermissions {
		newKey.Permissions = models.ApiPermNone
	}
	database.SaveApiKey(newKey)
	return newKey
}

func apiDeleteKey(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthDelete)
	if !ok {
		panic("invalid parameter passed")
	}
	apiKeyOwner, apiKey, ok := isValidKeyForEditing(request.KeyId)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to delete this API key")
		return
	}
	database.DeleteApiKey(apiKey.Id)
}

func apiModifyApiKey(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthModify)
	if !ok {
		panic("invalid parameter passed")
	}
	apiKeyOwner, apiKey, ok := isValidKeyForEditing(request.KeyId)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to delete this API key")
		return
	}

	switch request.Permission {
	case models.ApiPermReplace:
		if !apiKeyOwner.HasPermissionReplace() {
			sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "Insufficient user permission for owner to set this API permission")
			return
		}
	case models.ApiPermManageUsers:
		if !apiKeyOwner.HasPermissionManageUsers() {
			sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "Insufficient user permission for owner to set this API permission")
			return
		}
	case models.ApiPermManageLogs:
		if !apiKeyOwner.HasPermissionManageLogs() {
			sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "Insufficient user permission for owner to set this API permission")
			return
		}
	case models.ApiPermManageFileRequests:
		if !apiKeyOwner.HasPermissionCreateFileRequests() {
			sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "Insufficient user permission for owner to set this API permission")
			return
		}
	default:
		// do nothing
	}
	if request.GrantPermission && !apiKey.HasPermission(request.Permission) {
		apiKey.GrantPermission(request.Permission)
		database.SaveApiKey(apiKey)
		return
	}
	if !request.GrantPermission && apiKey.HasPermission(request.Permission) {
		apiKey.RemovePermission(request.Permission)
		database.SaveApiKey(apiKey)
	}
}

// isValidKeyForEditing checks if the provided API key is either a public or private ID and returns the user and API
// key model (including the private ID)
func isValidKeyForEditing(apiKey string) (models.User, models.ApiKey, bool) {
	apiKey = publicKeyToApiKey(apiKey)
	user, fullApiKey, ok := isValidApiKey(apiKey, false, models.ApiPermNone)
	if !ok {
		return models.User{}, models.ApiKey{}, false
	}
	return user, fullApiKey, true
}

func isValidUserForEditing(w http.ResponseWriter, userId int) (models.User, bool) {
	user, ok := database.GetUser(userId)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid user id provided.")
		return models.User{}, false
	}
	return user, true
}

func apiCreateApiKey(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthCreate)
	if !ok {
		panic("invalid parameter passed")
	}
	key := generateNewKey(request.BasicPermissions, user.Id, request.FriendlyName, "")
	output := models.ApiKeyOutput{
		Result:   "OK",
		Id:       key.Id,
		PublicId: key.PublicId,
	}
	result, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiCreateUser(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramUserCreate)
	if !ok {
		panic("invalid parameter passed")
	}
	newUser, err := users.Create(request.Username)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrorNameToShort):
			sendError(w, http.StatusBadRequest, errorcodes.NoPermission, "Invalid username provided.")
		case errors.Is(err, users.ErrorUserExists):
			sendError(w, http.StatusConflict, errorcodes.AlreadyExists, "User already exists.")
		default:
			sendError(w, http.StatusInternalServerError, errorcodes.InternalServer, err.Error())
		}
		return
	}
	logging.LogUserCreation(newUser, user)
	_, _ = w.Write([]byte(newUser.ToJson()))
}

func apiChangeFriendlyName(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthFriendlyName)
	if !ok {
		panic("invalid parameter passed")
	}
	ownerApiKey, apiKey, ok := isValidKeyForEditing(request.KeyId)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid key ID provided.")
		return
	}
	if ownerApiKey.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to edit this API key")
		return
	}
	err := renameApiKeyFriendlyName(apiKey.Id, request.FriendlyName)
	if err != nil {
		sendError(w, http.StatusInternalServerError, errorcodes.InternalServer, err.Error())
		return
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

func apiDeleteFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesDelete)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := database.GetMetaDataById(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermDeleteOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to delete this file")
		return
	}
	logging.LogDelete(file, user)
	if request.DelaySeconds == 0 {
		_ = storage.DeleteFile(request.Id, true)
	} else {
		_ = storage.DeleteFileSchedule(request.Id, request.DelaySeconds*1000, true)
	}
}

func apiRestoreFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesRestore)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := database.GetMetaDataById(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid file ID provided or file has already been deleted.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermDeleteOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to restore this file")
		return
	}
	file, ok = storage.CancelPendingFileDeletion(file.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid file ID provided or file has already been deleted.")
		return
	}
	logging.LogRestore(file, user)
	outputFileJson(w, file)
}

func apiChunkAdd(w http.ResponseWriter, r requestParser, _ models.User) {
	request, ok := r.(*paramChunkAdd)
	if !ok {
		panic("invalid parameter passed")
	}
	statusCode, errCode, errString := processNewChunk(w, request, configuration.Get().MaxFileSizeMB, "")
	if statusCode != http.StatusOK {
		sendError(w, statusCode, errCode, errString)
	}
}

func apiChunkReserve(w http.ResponseWriter, r requestParser, _ models.User) {
	request, ok := r.(*paramChunkReserve)
	if !ok {
		panic("invalid parameter passed")
	}
	fileRequest, ok, status, errorCode, errorMsg := checkFileRequestAndApiKey(request.Id, request.ApiKey)
	if !ok {
		sendError(w, status, errorCode, errorMsg)
		return
	}
	if fileRequest.FilesRemaining() <= 0 && !fileRequest.IsUnlimitedFiles() {
		sendError(w, http.StatusBadRequest, errorcodes.CannotUploadMoreFiles, "No more files can be uploaded for this file request")
		return
	}
	if fileRequest.IsUnlimitedFiles() && !ratelimiter.IsAllowedNewUuid(fileRequest.Id) {
		sendError(w, http.StatusTooManyRequests, errorcodes.RateLimited, "Too many reservations for this file request. Please wait a few seconds before reserving a new uuid.")
		return
	}
	uuid := chunkreservation.New(fileRequest.Id)
	result, err := json.Marshal(struct {
		Result string `json:"Result"`
		Uuid   string `json:"Uuid"`
	}{"OK", uuid})
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiChunkUnreserve(w http.ResponseWriter, r requestParser, _ models.User) {
	request, ok := r.(*paramChunkUnreserve)
	if !ok {
		panic("invalid parameter passed")
	}
	fileRequest, ok, status, errorCode, errorMsg := checkFileRequestAndApiKey(request.Id, request.ApiKey)
	if !ok {
		sendError(w, status, errorCode, errorMsg)
		return
	}
	chunkreservation.SetComplete(fileRequest.Id, request.Uuid)
	_ = chunking.DeleteChunk(request.Uuid)
	_, _ = w.Write([]byte(`{"Result":"OK"}`))
}

func apiChunkUploadRequestAdd(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramChunkUploadRequestAdd)
	if !ok {
		panic("invalid parameter passed")
	}
	fileRequest, ok, status, errorCode, errorMsg := checkFileRequestAndApiKey(request.FileRequestId, request.ApiKey)
	if !ok {
		sendError(w, status, errorCode, errorMsg)
		return
	}
	maxUpload := configuration.Get().MaxFileSizeMB
	if !user.IsAdmin() && configuration.GetEnvironment().MaxSizeGuestUploadMb != 0 {
		maxUpload = min(maxUpload, configuration.GetEnvironment().MaxSizeGuestUploadMb)
	}
	if !fileRequest.IsUnlimitedSize() {
		maxUpload = min(maxUpload, fileRequest.MaxSize)
	}
	statusCode, errorCode, errString := processNewChunk(w, request, maxUpload, fileRequest.Id)
	if statusCode != http.StatusOK {
		sendError(w, statusCode, errorCode, errString)
	}
}

func checkFileRequestAndApiKey(fileRequestId, apiKey string) (models.FileRequest, bool, int, int, string) {
	fileRequest, ok := filerequest.Get(fileRequestId)
	if !ok {
		return models.FileRequest{}, false, http.StatusNotFound, errorcodes.NotFound, "FileRequest does not exist with the given ID"
	}
	if fileRequest.ApiKey != apiKey {
		return models.FileRequest{}, false, http.StatusUnauthorized, errorcodes.InvalidApiKey, "Invalid API key"
	}
	if !fileRequest.IsUnlimitedTime() && fileRequest.Expiry < time.Now().Unix() {
		return models.FileRequest{}, false, http.StatusUnauthorized, errorcodes.RequestExpired, "Filerequest has expired"
	}
	if !fileRequest.IsUnlimitedFiles() && fileRequest.UploadedFiles >= fileRequest.MaxFiles {
		return models.FileRequest{}, false, http.StatusUnauthorized, errorcodes.CannotUploadMoreFiles, "Max file count has already been reached for this file request"
	}
	return fileRequest, true, 0, 0, ""
}

type chunkParams interface {
	GetRequest() *http.Request
}

func processNewChunk(w http.ResponseWriter, request chunkParams, maxFileSizeMb int, filerequestId string) (int, int, string) {
	maxUpload := int64(maxFileSizeMb) * 1024 * 1024
	if request.GetRequest().ContentLength > maxUpload {
		return http.StatusBadRequest, errorcodes.FileTooLarge, storage.ErrorFileTooLarge.Error()
	}
	request.GetRequest().Body = http.MaxBytesReader(w, request.GetRequest().Body, maxUpload)
	errCode, err := fileupload.ProcessNewChunk(w, request.GetRequest(), true, filerequestId)
	if err != nil {
		return http.StatusBadRequest, errCode, err.Error()
	}
	return http.StatusOK, 0, ""
}

func apiChunkComplete(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramChunkComplete)
	if !ok {
		panic("invalid parameter passed")
	}
	uploadParams := fileupload.CreateUploadConfig(request.AllowedDownloads,
		request.ExpiryDays,
		request.Password,
		request.UnlimitedTime,
		request.UnlimitedDownloads,
		request.IsE2E,
		request.FileSize,
		"")
	if request.IsNonBlocking {
		go doBlockingPartCompleteChunk(nil, request.Uuid, request.FileHeader, user, uploadParams)
		_, _ = io.WriteString(w, "{\"result\":\"OK\"}")
		return
	}
	doBlockingPartCompleteChunk(w, request.Uuid, request.FileHeader, user, uploadParams)
}

func doBlockingPartCompleteChunk(w http.ResponseWriter, uuid string, fileHeader chunking.FileHeader, user models.User, uploadParameters models.UploadParameters) {
	file, err := fileupload.CompleteChunk(uuid, fileHeader, user.Id, uploadParameters)
	if err != nil {
		sendError(w, http.StatusBadRequest, errorcodes.UnspecifiedError, err.Error())
		return
	}
	if uploadParameters.FileRequestId != "" {
		chunkreservation.SetComplete(uploadParameters.FileRequestId, uuid)
	}
	fr, _ := filerequest.Get(uploadParameters.FileRequestId)
	logging.LogUpload(file, user, fr)
	outputFileJson(w, file)
}

func apiChunkUploadRequestComplete(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramChunkUploadRequestComplete)
	if !ok {
		panic("invalid parameter passed")
	}
	fileRequest, ok, status, errorCode, errorMsg := checkFileRequestAndApiKey(request.FileRequestId, request.ApiKey)
	if !ok {
		sendError(w, status, errorCode, errorMsg)
		return
	}
	uploadParams := fileupload.CreateUploadConfig(0,
		0, "", true, true,
		false, request.FileSize, fileRequest.Id)
	if request.IsNonBlocking {
		go doBlockingPartCompleteChunk(nil, request.Uuid, request.FileHeader, user, uploadParams)
		_, _ = io.WriteString(w, "{\"result\":\"OK\"}")
		return
	}
	doBlockingPartCompleteChunk(w, request.Uuid, request.FileHeader, user, uploadParams)
}

func apiVersionInfo(w http.ResponseWriter, _ requestParser, _ models.User) {
	type versionInfo struct {
		Version    string
		VersionInt int
	}
	result, err := json.Marshal(versionInfo{versionReadable, versionInt})
	helper.Check(err)
	_, _ = w.Write(result)
}
func apiConfigInfo(w http.ResponseWriter, _ requestParser, _ models.User) {
	type configInfo struct {
		MaxFilesize               int
		MaxChunksize              int
		EndToEndEncryptionEnabled bool
	}
	config := configuration.Get()
	result, err := json.Marshal(configInfo{
		MaxFilesize:               config.MaxFileSizeMB,
		MaxChunksize:              config.ChunkSize,
		EndToEndEncryptionEnabled: config.Encryption.Level == encryption.EndToEndEncryption,
	})
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiList(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesListAll)
	if !ok {
		panic("invalid parameter passed")
	}
	validFiles := getFilesForUser(user, request.ShowFileRequests)
	result, err := json.Marshal(validFiles)
	helper.Check(err)
	_, _ = w.Write(result)
}

func getFilesForUser(user models.User, includeUploadRequests bool) []models.FileApiOutput {
	var validFiles []models.FileApiOutput
	timeNow := time.Now().Unix()
	config := configuration.Get()
	for _, element := range database.GetAllMetadata() {
		if !includeUploadRequests && element.IsFileRequest() {
			continue
		}
		if element.UserId == user.Id || user.HasPermission(models.UserPermListOtherUploads) {
			if !storage.IsExpiredFile(element, timeNow) {
				file, err := element.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
				helper.Check(err)
				validFiles = append(validFiles, file)
			}
		}
	}
	return validFiles
}

func apiListSingle(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesListSingle)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := storage.GetFile(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "File not found")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to view file")
		return
	}
	config := configuration.Get()
	output, err := file.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
	helper.Check(err)
	result, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiDownloadSingle(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesDownloadSingle)
	if !ok {
		panic("invalid parameter passed")
	}
	file, statusCode, errCode, errMessage := checkDownloadAllowed(request.Id, user)
	if statusCode != 0 {
		sendError(w, statusCode, errCode, errMessage)
		return
	}
	if !request.PresignUrl {
		forceDecryption := !file.Encryption.IsEndToEndEncrypted
		storage.ServeFile(file, w, request.WebRequest, true, request.IncreaseCounter, forceDecryption)
		return
	}
	createAndOutputPresignedUrl([]string{file.Id}, w, "")
}

func apiDownloadZip(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesDownloadZip)
	if !ok {
		panic("invalid parameter passed")
	}
	requestedFiles := make([]models.File, 0)
	requestedFileIds := make([]string, 0)
	for _, fileId := range request.Ids {
		file, statusCode, errCode, errMessage := checkDownloadAllowed(fileId, user)
		if statusCode != 0 {
			sendError(w, statusCode, errCode, errMessage)
			return
		}
		requestedFiles = append(requestedFiles, file)
		requestedFileIds = append(requestedFileIds, file.Id)
	}
	if !request.PresignUrl {
		storage.ServeFilesAsZip(requestedFiles, request.Filename, w, request.WebRequest)
		return
	}
	createAndOutputPresignedUrl(requestedFileIds, w, request.Filename)
}

func checkDownloadAllowed(fileId string, user models.User) (models.File, int, int, string) {
	file, ok := storage.GetFile(fileId)
	if !ok {
		return models.File{}, http.StatusNotFound, errorcodes.NotFound, "file not found"
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		return models.File{}, http.StatusUnauthorized, errorcodes.NoPermission, "no permission to download file"
	}
	return file, 0, 0, ""
}

func createAndOutputPresignedUrl(ids []string, w http.ResponseWriter, filename string) {
	presignUrl := models.Presign{
		Id:       helper.GenerateRandomString(60),
		FileIds:  ids,
		Expiry:   time.Now().Add(time.Second * 30).Unix(),
		Filename: filename,
	}
	presign.Save(presignUrl)
	response := struct {
		Result      string `json:"Result"`
		DownloadUrl string `json:"downloadUrl"`
	}{"OK", configuration.Get().ServerUrl + "downloadPresigned?key=" + presignUrl.Id}
	result, err := json.Marshal(response)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiUploadFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesAdd)
	if !ok {
		panic("invalid parameter passed")
	}
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.Request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, errorcodes.FileTooLarge, storage.ErrorFileTooLarge.Error())
		return
	}

	request.Request.Body = http.MaxBytesReader(w, request.Request.Body, maxUpload)
	err := fileupload.ProcessCompleteFile(w, request.Request, user.Id, configuration.Get().MaxMemory)
	if err != nil {
		sendError(w, http.StatusBadRequest, errorcodes.UnspecifiedError, err.Error())
		return
	}
}

func apiDuplicateFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesDuplicate)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := storage.GetFile(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid id provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to duplicate this file")
		return
	}
	uploadConfig := fileupload.CreateUploadConfig(request.AllowedDownloads,
		request.ExpiryDays,
		request.Password,
		request.UnlimitedTime,
		request.UnlimitedDownloads,
		false, // is not being used by storage.DuplicateFile
		0,     // is not being used by storage.DuplicateFile
		"")
	uploadConfig.UserId = user.Id
	newFile, err := storage.DuplicateFile(file, request.RequestedChanges, request.FileName, uploadConfig)
	if err != nil {
		sendError(w, http.StatusInternalServerError, errorcodes.InternalServer, err.Error())
		return
	}
	outputFileApiInfo(w, newFile)
}

func apiChangeFileOwner(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesChangeOwner)
	if !ok {
		panic("invalid parameter passed")
	}
	file, ok := storage.GetFile(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid id provided.")
		return
	}
	if !user.HasPermission(models.UserPermEditOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to edit this file")
		return
	}
	_, exists := database.GetUser(request.NewOwner)
	if !exists {
		sendError(w, http.StatusBadRequest, errorcodes.NotFound, "User does not exist")
		return
	}
	file.UserId = request.NewOwner
	database.SaveMetaData(file)
	outputFileApiInfo(w, file)
}

func apiReplaceFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesReplace)
	if !ok {
		panic("invalid parameter passed")
	}
	fileOriginal, ok := storage.GetFile(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid id provided.")
		return
	}
	if fileOriginal.UserId != user.Id && !user.HasPermission(models.UserPermReplaceOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to replace this file")
		return
	}

	if fileOriginal.IsFileRequest() {
		sendError(w, http.StatusBadRequest, errorcodes.UnsupportedFile, "Cannot replace a file request upload")
		return
	}
	fileNewContent, ok := storage.GetFile(request.IdNewContent)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "Invalid id provided.")
		return
	}
	if fileNewContent.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to duplicate this file")
		return
	}

	modifiedFile, err := storage.ReplaceFile(request.Id, request.IdNewContent, request.Delete)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrorReplaceE2EFile):
			sendError(w, http.StatusBadRequest, errorcodes.EndToEndNotSupported, "End-to-End encrypted files cannot be replaced")
		case errors.Is(err, storage.ErrorFileNotFound):
			sendError(w, http.StatusNotFound, errorcodes.NotFound, "A file with such an ID could not be found")
		default:
			sendError(w, http.StatusBadRequest, errorcodes.InvalidUserInput, err.Error())
		}
		return
	}
	logging.LogReplace(fileOriginal, modifiedFile, user)
	outputFileApiInfo(w, modifiedFile)
}

func outputFileApiInfo(w http.ResponseWriter, file models.File) {
	config := configuration.Get()
	publicOutput, err := file.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
	helper.Check(err)
	result, err := json.Marshal(publicOutput)
	helper.Check(err)
	_, _ = w.Write(result)
}

func outputFileJson(w http.ResponseWriter, file models.File) {
	if w == nil {
		return
	}
	config := configuration.Get()
	_, _ = io.WriteString(w, file.ToJsonResult(config.ServerUrl, config.IncludeFilename))
}

func apiModifyUser(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramUserModify)
	if !ok {
		panic("invalid parameter passed")
	}
	userEdit, ok := isValidUserForEditing(w, request.Id)
	if !ok {
		return
	}
	if userEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot modify super admin")
		return
	}
	if userEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot modify yourself")
		return
	}
	logging.LogUserEdit(userEdit, user)
	if request.GrantPermission {
		if !userEdit.HasPermission(request.Permission) {
			userEdit.GrantPermission(request.Permission)
			database.SaveUser(userEdit, false)
			updateApiKeyPermsOnUserPermChange(userEdit.Id, request.Permission, true)
		}
		return
	}
	if userEdit.HasPermission(request.Permission) {
		userEdit.RemovePermission(request.Permission)
		database.SaveUser(userEdit, false)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, request.Permission, false)
	}
}

func apiChangeUserRank(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramUserChangeRank)
	if !ok {
		panic("invalid parameter passed")
	}
	userEdit, ok := isValidUserForEditing(w, request.Id)
	if !ok {
		return
	}
	if userEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot modify yourself")
		return
	}
	if userEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot modify super admin")
		return
	}
	userEdit.UserLevel = request.NewRank
	switch request.NewRank {
	case models.UserLevelAdmin:
		userEdit.Permissions = models.UserPermissionAll
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermReplaceUploads, true)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermManageUsers, true)
	case models.UserLevelUser:
		userEdit.Permissions = models.UserPermissionNone
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermReplaceUploads, false)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermManageUsers, false)
	default:
		sendError(w, http.StatusBadRequest, errorcodes.InvalidUserInput, "invalid rank sent")
		return
	}
	logging.LogUserEdit(userEdit, user)
	database.SaveUser(userEdit, false)
}

func updateApiKeyPermsOnUserPermChange(userId int, userPerm models.UserPermission, isNewlyGranted bool) {
	var affectedPermission models.ApiPermission
	switch userPerm {
	case models.UserPermManageUsers:
		affectedPermission = models.ApiPermManageUsers
	case models.UserPermReplaceUploads:
		affectedPermission = models.ApiPermReplace
	case models.UserPermManageLogs:
		affectedPermission = models.ApiPermManageLogs
	case models.UserPermGuestUploads:
		affectedPermission = models.ApiPermManageFileRequests
	default:
		return
	}
	for _, apiKey := range database.GetAllApiKeys() {
		if apiKey.UserId != userId {
			continue
		}
		if isNewlyGranted {
			if apiKey.IsSystemKey {
				apiKey.GrantPermission(affectedPermission)
				database.SaveApiKey(apiKey)
			}
		} else if apiKey.HasPermission(affectedPermission) {
			apiKey.RemovePermission(affectedPermission)
			database.SaveApiKey(apiKey)
		}
	}
}

func apiResetPassword(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramUserResetPw)
	if !ok {
		panic("invalid parameter passed")
	}
	userToEdit, ok := isValidUserForEditing(w, request.Id)
	if !ok {
		return
	}
	if userToEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot reset password of super admin")
		return
	}
	if userToEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot reset password of yourself")
		return
	}
	userToEdit.ResetPassword = true
	password := ""
	if request.NewPassword {
		password = helper.GenerateRandomString(configuration.GetEnvironment().MinLengthPassword + 2)
		userToEdit.Password = configuration.HashPassword(password, false)
	}
	database.DeleteAllSessionsByUser(userToEdit.Id)
	database.SaveUser(userToEdit, false)
	_, _ = w.Write([]byte("{\"Result\":\"ok\",\"password\":\"" + password + "\"}"))
}

func apiDeleteUser(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramUserDelete)
	if !ok {
		panic("invalid parameter passed")
	}
	userToDelete, ok := isValidUserForEditing(w, request.Id)
	if !ok {
		return
	}
	if userToDelete.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot delete super admin")
		return
	}
	if userToDelete.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, errorcodes.ResourceCanNotBeEdited, "Cannot delete yourself")
		return
	}
	logging.LogUserDeletion(userToDelete, user)
	database.DeleteUser(userToDelete.Id)

	for _, fRequest := range database.GetAllFileRequests() {
		if fRequest.UserId == userToDelete.Id {
			if request.DeleteFiles {
				filerequest.Delete(fRequest)
			} else {
				fRequest.UserId = user.Id
				database.SaveFileRequest(fRequest)
			}
		}
	}

	for _, file := range database.GetAllMetadata() {
		if file.UserId == userToDelete.Id {
			if request.DeleteFiles {
				database.DeleteMetaData(file.Id)
			} else {
				file.UserId = user.Id
				database.SaveMetaData(file)
			}
		}
	}
	for _, apiKey := range database.GetAllApiKeys() {
		if apiKey.UserId == userToDelete.Id {
			database.DeleteApiKey(apiKey.Id)
		}
	}
	database.DeleteAllSessionsByUser(userToDelete.Id)
	database.DeleteEnd2EndInfo(userToDelete.Id)
}

func apiLogsDelete(_ http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramLogsDelete)
	if !ok {
		panic("invalid parameter passed")
	}
	logging.DeleteLogs(user.Name, user.Id, request.Timestamp, request.Request)
}
func apiLogsGet(w http.ResponseWriter, r requestParser, _ models.User) {
	request, ok := r.(*paramLogsGet)
	if !ok {
		panic("invalid parameter passed")
	}
	result := struct {
		LogEntries string `json:"logEntries"`
		Timestamp  int64  `json:"timestamp"`
	}{}
	if request.Timestamp == 0 {
		result.LogEntries, _ = logging.GetAll()
	} else {
		result.LogEntries = logging.GetSince(request.Timestamp)
	}
	result.Timestamp = time.Now().Unix()
	resultJson, err := json.Marshal(result)
	helper.Check(err)
	_, _ = w.Write(resultJson)
}

func apiLogSystemStatus(w http.ResponseWriter, _ requestParser, _ models.User) {
	result := struct {
		Uptime                int64  `json:"uptime"`
		TrafficRecordingSince int64  `json:"trafficRecordingSince"`
		CpuLoad               int    `json:"cpuLoad"`
		MemoryUsagePercentage int    `json:"memoryUsagePercentage"`
		DiskUsagePercentage   int    `json:"diskUsagePercentage"`
		ActiveFiles           int    `json:"activeFiles"`
		MemoryUsed            uint64 `json:"memoryUsed"`
		MemoryTotal           uint64 `json:"memoryTotal"`
		DiskUsed              uint64 `json:"diskUsed"`
		DiskTotal             uint64 `json:"diskTotal"`
		DataServed            uint64 `json:"dataServed"`
	}{
		Uptime:      serverstats.GetUptime(),
		CpuLoad:     serverstats.GetCpuUsage(),
		ActiveFiles: serverstats.GetTotalFiles(),
	}
	result.DataServed, result.TrafficRecordingSince = serverstats.GetCurrentTraffic()
	_, result.MemoryUsed, result.MemoryTotal, result.MemoryUsagePercentage = serverstats.GetMemoryInfo()
	_, result.DiskUsed, result.DiskTotal, result.DiskUsagePercentage = serverstats.GetDiskInfo()
	resultJson, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		sendError(w, http.StatusInternalServerError, errorcodes.InternalServer, err.Error())
		return
	}
	_, _ = w.Write(resultJson)
}

func apiLogResetTraffic(w http.ResponseWriter, _ requestParser, _ models.User) {
	serverstats.ClearTraffic()
	_, _ = w.Write([]byte(`{"Result":"OK"}`))
}

func apiE2eGet(w http.ResponseWriter, _ requestParser, user models.User) {
	info := database.GetEnd2EndInfo(user.Id)
	// If e2e is supported for upload requests at some point, this needs to be changed
	files := getFilesForUser(user, false)
	ids := make([]string, len(files))
	for i, file := range files {
		ids[i] = file.Id
	}
	info.AvailableFiles = ids
	bytesE2e, err := json.Marshal(info)
	helper.Check(err)
	_, _ = w.Write(bytesE2e)
}

func apiE2eSet(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramE2eStore)
	if !ok {
		panic("invalid parameter passed")
	}
	database.SaveEnd2EndInfo(request.EncryptedInfo, user.Id)
	_, _ = w.Write([]byte("{\"result\":\"OK\"}"))
}

func apiURequestDelete(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramURequestDelete)
	if !ok {
		panic("invalid parameter passed")
	}

	uploadRequest, ok := database.GetFileRequest(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "FileRequest does not exist with the given ID")
		return
	}
	if uploadRequest.UserId != user.Id && !user.HasPermission(models.UserPermDeleteOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to delete this upload request")
		return
	}
	filerequest.Delete(uploadRequest)
	logging.LogDeleteFileRequest(uploadRequest, user)
	_, _ = w.Write([]byte("{\"result\":\"OK\"}"))
}

func isUserAllowedUnlimited(request *paramURequestSave, isNewRequest bool, user models.User) bool {
	if user.IsAdmin() {
		return true
	}
	isServerLimitMaxSize := configuration.GetEnvironment().MaxSizeGuestUploadMb != 0
	isServerLimitMaxFiles := configuration.GetEnvironment().MaxFilesGuestUpload != 0
	if isServerLimitMaxSize {
		if (request.IsMaxSizeSet || isNewRequest) &&
			(request.MaxSizeMb == 0 || request.MaxSizeMb > configuration.GetEnvironment().MaxSizeGuestUploadMb) {
			return false
		}
	}
	if isServerLimitMaxFiles {
		if (request.IsMaxFilesSet || isNewRequest) &&
			(request.MaxFiles == 0 || request.MaxFiles > configuration.GetEnvironment().MaxFilesGuestUpload) {
			return false
		}
	}

	return true
}

func apiURequestSave(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramURequestSave)
	if !ok {
		panic("invalid parameter passed")
	}
	uploadRequest := models.FileRequest{}
	isNewRequest := request.Id == ""

	if !isUserAllowedUnlimited(request, isNewRequest, user) {
		sendError(w, http.StatusBadRequest, errorcodes.AdminOnly, "Only admin users can create requests with unlimited size / file count"+
			" or values larger than the server's max size / file count")
		return
	}

	if !isNewRequest {
		uploadRequest, ok = database.GetFileRequest(request.Id)
		if !ok {
			sendError(w, http.StatusNotFound, errorcodes.NotFound, "FileRequest does not exist with the given ID")
			return
		}
		if uploadRequest.UserId != user.Id && !user.HasPermission(models.UserPermEditOtherUploads) {
			sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to edit this upload request")
			return
		}
	} else {
		uploadRequest = filerequest.New(user)
		apiKey := generateNewKey(false, user.Id, "File Request Public Access", uploadRequest.Id)
		uploadRequest.ApiKey = apiKey.Id
	}

	if request.Name == "" {
		if request.IsNameSet || uploadRequest.Name == "" {
			uploadRequest.Name = "Unnamed Request"
		}
	} else {
		uploadRequest.Name = request.Name
	}
	if request.IsExpirySet {
		uploadRequest.Expiry = request.Expiry
	}
	if request.IsMaxFilesSet {
		uploadRequest.MaxFiles = request.MaxFiles
	}
	if request.IsMaxSizeSet {
		uploadRequest.MaxSize = request.MaxSizeMb
	}
	if request.IsNotesSet {
		uploadRequest.Notes = request.Notes
	}
	database.SaveFileRequest(uploadRequest)
	uploadRequest, ok = filerequest.Get(uploadRequest.Id)
	if isNewRequest {
		logging.LogCreateFileRequest(uploadRequest, user)
	} else {
		logging.LogEditFileRequest(uploadRequest, user)
	}
	result, err := json.Marshal(uploadRequest)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiUploadRequestList(w http.ResponseWriter, _ requestParser, user models.User) {
	userRequests := make([]models.FileRequest, 0)
	for _, request := range filerequest.GetAll() {
		if request.UserId == user.Id || user.HasPermission(models.UserPermListOtherUploads) {
			userRequests = append(userRequests, request)
		}
	}
	result, err := json.Marshal(userRequests)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiUploadRequestListSingle(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramURequestListSingle)
	if !ok {
		panic("invalid parameter passed")
	}

	uploadRequest, ok := filerequest.Get(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, errorcodes.NotFound, "FileRequest does not exist with the given ID")
		return
	}
	if uploadRequest.UserId != user.Id && !user.HasPermission(models.UserPermDeleteOtherUploads) {
		sendError(w, http.StatusUnauthorized, errorcodes.NoPermission, "No permission to delete this upload request")
		return
	}
	result, err := json.Marshal(uploadRequest)
	helper.Check(err)
	_, _ = w.Write(result)
}

func isAuthorisedForApi(r *http.Request, routing apiRoute) (models.User, bool) {
	keyId := r.Header.Get("apikey")
	user, apiKey, ok := isValidApiKey(keyId, true, routing.ApiPerm)
	if !ok {
		return models.User{}, false
	}
	// Returns false if a public upload key is used for non-public api call or vice versa
	if routing.IsFileRequestApi != apiKey.IsUploadRequestKey() {
		return models.User{}, false
	}
	return user, true
}

func sendError(w http.ResponseWriter, statusCode, errorCode int, errorMessage string) {
	if w == nil {
		return
	}
	w.WriteHeader(statusCode)
	output := struct {
		Result  string `json:"Result"`
		Message string `json:"ErrorMessage"`
		Code    int    `json:"ErrorCode"`
	}{Result: "error", Message: errorMessage, Code: errorCode}
	outputBytes, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(outputBytes)
}

// publicKeyToApiKey tries to convert a (possible) public key to a private key
// If not a public key or if invalid, the original value is returned
func publicKeyToApiKey(publicKey string) string {
	if len(publicKey) == LengthPublicId {
		privateApiKey, ok := database.GetApiKeyByPublicKey(publicKey)
		if ok {
			return privateApiKey
		}
	}
	return publicKey
}

// isValidApiKey checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func isValidApiKey(key string, modifyTime bool, requiredPermissionApiKey models.ApiPermission) (models.User, models.ApiKey, bool) {
	if key == "" {
		return models.User{}, models.ApiKey{}, false
	}
	savedKey, ok := database.GetApiKey(key)
	if ok && savedKey.Id != "" && (savedKey.Expiry == 0 || savedKey.Expiry > time.Now().Unix()) {
		if modifyTime {
			savedKey.LastUsed = time.Now().Unix()
			database.UpdateTimeApiKey(savedKey)
		}
		if !savedKey.HasPermission(requiredPermissionApiKey) {
			return models.User{}, models.ApiKey{}, false
		}
		user, ok := database.GetUser(savedKey.UserId)
		if !ok {
			return models.User{}, models.ApiKey{}, false
		}
		return user, savedKey, true
	}
	return models.User{}, models.ApiKey{}, false
}
