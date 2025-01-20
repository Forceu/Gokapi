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
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const lengthPublicId = 35
const lengthApiKey = 30
const minLengthUser = 4

type apiRoute struct {
	Url         string
	HasWildcard bool
	ApiPerm     models.ApiPermission
	execution   apiFunc
}

func (r apiRoute) Continue(w http.ResponseWriter, request apiRequest, user models.User) {
	r.execution(w, request, user)
}

type apiFunc func(w http.ResponseWriter, request apiRequest, user models.User)

var routes = []apiRoute{
	{
		Url:       "/files/list",
		ApiPerm:   models.ApiPermView,
		execution: apiList,
	},
	{
		Url:         "/files/list/",
		ApiPerm:     models.ApiPermView,
		execution:   apiListSingle,
		HasWildcard: true,
	},
	{
		Url:       "/chunk/add",
		ApiPerm:   models.ApiPermUpload,
		execution: apiChunkAdd,
	},
	{
		Url:       "/chunk/complete",
		ApiPerm:   models.ApiPermUpload,
		execution: apiChunkComplete,
	},
	{
		Url:       "/files/add",
		ApiPerm:   models.ApiPermUpload,
		execution: apiUploadFile,
	},
	{
		Url:       "/files/delete",
		ApiPerm:   models.ApiPermDelete,
		execution: apiDeleteFile,
	},
	{
		Url:       "/files/duplicate",
		ApiPerm:   models.ApiPermUpload,
		execution: apiDuplicateFile,
	},
	{
		Url:       "/files/modify",
		ApiPerm:   models.ApiPermEdit,
		execution: apiEditFile,
	},
	{
		Url:       "/files/replace",
		ApiPerm:   models.ApiPermReplace,
		execution: apiReplaceFile,
	},
	{
		Url:       "/auth/create",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiCreateApiKey,
	},
	{
		Url:       "/auth/friendlyname",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiChangeFriendlyName,
	},
	{
		Url:       "/auth/modify",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiModifyApiKey,
	},
	{
		Url:       "/auth/delete",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiDeleteKey,
	},
	{
		Url:       "/user/create",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiCreateUser,
	},
	{
		Url:       "/user/changeRank",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiChangeUserRank,
	},
	{
		Url:       "/user/delete",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiDeleteUser,
	},
	{
		Url:       "/user/modify",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiModifyUser,
	},
	{
		Url:       "/user/resetPassword",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiResetPassword,
	},
}

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	request, err := parseRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	routing, ok := getRouting(request.requestUrl)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	var user models.User
	user, ok = isAuthorisedForApi(w, request, routing)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	routing.Continue(w, request, user)
}

func apiEditFile(w http.ResponseWriter, request apiRequest, user models.User) {
	file, ok := database.GetMetaDataById(request.filemodInfo.id)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermEditOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to edit file.")
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

func getRouting(requestUrl string) (apiRoute, bool) {
	for _, route := range routes {
		if (!route.HasWildcard && requestUrl == route.Url) ||
			(route.HasWildcard && strings.HasPrefix(requestUrl, route.Url)) {
			return route, true
		}
	}
	return apiRoute{}, false
}

// generateNewKey generates and saves a new API key
func generateNewKey(defaultPermissions bool, userId int) models.ApiKey {
	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(lengthApiKey),
		PublicId:     helper.GenerateRandomString(lengthPublicId),
		FriendlyName: "Unnamed key",
		Permissions:  models.ApiPermDefault,
		IsSystemKey:  false,
		UserId:       userId,
	}
	if !defaultPermissions {
		newKey.Permissions = models.ApiPermNone
	}
	database.SaveApiKey(newKey)
	return newKey
}

// newSystemKey generates a new API key that is only used internally for the GUI
// and will be valid for 48 hours
func newSystemKey(userId int) string {
	user, ok := database.GetUser(userId)
	if !ok {
		panic("user not found")
	}
	tempKey := models.ApiKey{
		Permissions: models.ApiPermAll,
	}
	if !user.HasPermissionReplace() {
		tempKey.RemovePermission(models.ApiPermReplace)
	}
	if !user.HasPermissionManageUsers() {
		tempKey.RemovePermission(models.ApiPermManageUsers)
	}

	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(lengthApiKey),
		PublicId:     helper.GenerateRandomString(lengthPublicId),
		FriendlyName: "Internal System Key",
		Permissions:  tempKey.Permissions,
		Expiry:       time.Now().Add(time.Hour * 48).Unix(),
		IsSystemKey:  true,
		UserId:       userId,
	}
	database.SaveApiKey(newKey)
	return newKey.Id
}

// GetSystemKey returns the latest System API key or generates a new one, if none exists or the current one expires
// within the next 24 hours
func GetSystemKey(userId int) string {
	key, ok := database.GetSystemKey(userId)
	if !ok || key.Expiry < time.Now().Add(time.Hour*24).Unix() {
		return newSystemKey(userId)
	}
	return key.Id
}

func apiDeleteKey(w http.ResponseWriter, request apiRequest, user models.User) {
	apiKeyOwner, ok := isValidKeyForEditing(w, request)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to delete this API key")
		return
	}
	database.DeleteApiKey(request.apiInfo.apiKeyToModify)
}

func apiModifyApiKey(w http.ResponseWriter, request apiRequest, user models.User) {
	apiKeyOwner, ok := isValidKeyForEditing(w, request)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to delete this API key")
		return
	}

	validPermissions := []models.ApiPermission{models.ApiPermView,
		models.ApiPermUpload, models.ApiPermDelete,
		models.ApiPermApiMod, models.ApiPermEdit,
		models.ApiPermReplace, models.ApiPermManageUsers}
	if !slices.Contains(validPermissions, request.apiInfo.permission) {
		sendError(w, http.StatusBadRequest, "Invalid permission sent")
		return
	}
	switch request.apiInfo.permission {
	case models.ApiPermReplace:
		if !apiKeyOwner.HasPermissionReplace() {
			sendError(w, http.StatusUnauthorized, "Insufficient user permission for owner to set this API permission")
			return
		}
	case models.ApiPermManageUsers:
		if !apiKeyOwner.HasPermissionManageUsers() {
			sendError(w, http.StatusUnauthorized, "Insufficient user permission for owner to set this API permission")
			return
		}
	default:
		// do nothing
	}
	key, ok := database.GetApiKey(request.apiInfo.apiKeyToModify)
	if !ok {
		sendError(w, http.StatusInternalServerError, "Could not modify API key")
		return
	}
	if request.apiInfo.grantPermission && !key.HasPermission(request.apiInfo.permission) {
		key.GrantPermission(request.apiInfo.permission)
		database.SaveApiKey(key)
		return
	}
	if !request.apiInfo.grantPermission && key.HasPermission(request.apiInfo.permission) {
		key.RemovePermission(request.apiInfo.permission)
		database.SaveApiKey(key)
	}
}

func isValidKeyForEditing(w http.ResponseWriter, request apiRequest) (models.User, bool) {
	user, ok := isValidApiKey(request.apiInfo.apiKeyToModify, false, models.ApiPermNone)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid api key provided.")
		return models.User{}, false
	}
	return user, true
}

func isValidUserForEditing(w http.ResponseWriter, request apiRequest) (models.User, bool) {
	user, ok := database.GetUser(request.usermodInfo.userId)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid user id provided.")
		return models.User{}, false
	}
	return user, true
}

func apiCreateApiKey(w http.ResponseWriter, request apiRequest, user models.User) {
	key := generateNewKey(request.apiInfo.basicPermissions, user.Id)
	output := models.ApiKeyOutput{
		Result:   "OK",
		Id:       key.Id,
		PublicId: key.PublicId,
	}
	if request.apiInfo.friendlyName != "" {
		err := renameApiKeyFriendlyName(key.Id, request.apiInfo.friendlyName)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	result, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiCreateUser(w http.ResponseWriter, request apiRequest, _ models.User) {
	name := request.usermodInfo.newUserName
	if len(name) < minLengthUser {
		sendError(w, http.StatusBadRequest, "Invalid username provided.")
		return
	}
	_, ok := database.GetUserByName(name)
	if ok {
		sendError(w, http.StatusConflict, "User already exists.")
		return
	}
	newUser := models.User{
		Name:      name,
		UserLevel: models.UserLevelUser,
	}
	database.SaveUser(newUser, true)
	newUser, ok = database.GetUserByName(name)
	if !ok {
		sendError(w, http.StatusInternalServerError, "Could not save user")
		return
	}
	_, _ = w.Write([]byte(newUser.ToJson()))
}

func apiChangeFriendlyName(w http.ResponseWriter, request apiRequest, user models.User) {
	ownerApiKey, ok := isValidKeyForEditing(w, request)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid api key provided.")
		return
	}
	if ownerApiKey.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to edit this key")
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

func apiDeleteFile(w http.ResponseWriter, request apiRequest, user models.User) {
	file, ok := database.GetMetaDataById(request.fileInfo.id)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId == user.Id || user.HasPermission(models.UserPermDeleteOtherUploads) {
		_ = storage.DeleteFile(request.fileInfo.id, true)
	} else {
		sendError(w, http.StatusUnauthorized, "No permission to delete this file")
		return
	}
}

func apiChunkAdd(w http.ResponseWriter, request apiRequest, user models.User) {
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
func apiChunkComplete(w http.ResponseWriter, request apiRequest, user models.User) {
	err := request.request.ParseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	request.request.Form.Set("chunkid", request.request.Form.Get("uuid"))
	chunkId, header, config, err := fileupload.ParseFileHeader(request.request)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, err := fileupload.CompleteChunk(chunkId, header, user.Id, config)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	_, _ = io.WriteString(w, file.ToJsonResult(config.ExternalUrl, configuration.Get().IncludeFilename))
}

func apiList(w http.ResponseWriter, _ apiRequest, user models.User) {
	var validFiles []models.FileApiOutput
	timeNow := time.Now().Unix()
	config := configuration.Get()
	for _, element := range database.GetAllMetadata() {
		if element.UserId == user.Id || user.HasPermission(models.UserPermListOtherUploads) {
			if !storage.IsExpiredFile(element, timeNow) {
				file, err := element.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
				helper.Check(err)
				validFiles = append(validFiles, file)
			}
		}
	}
	result, err := json.Marshal(validFiles)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiListSingle(w http.ResponseWriter, request apiRequest, user models.User) {
	id := strings.TrimPrefix(request.requestUrl, "/files/list/")
	file, ok := storage.GetFile(id)
	if !ok {
		sendError(w, http.StatusNotFound, "File not found")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to view file")
		return
	}
	config := configuration.Get()
	output, err := file.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
	helper.Check(err)
	result, err := json.Marshal(output)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiUploadFile(w http.ResponseWriter, request apiRequest, user models.User) {
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, storage.ErrorFileTooLarge.Error())
		return
	}

	request.request.Body = http.MaxBytesReader(w, request.request.Body, maxUpload)
	err := fileupload.ProcessCompleteFile(w, request.request, user.Id, configuration.Get().MaxMemory)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
}

func apiDuplicateFile(w http.ResponseWriter, request apiRequest, user models.User) {
	err := request.parseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, ok := storage.GetFile(request.fileInfo.id)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to duplicate this file")
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

func apiReplaceFile(w http.ResponseWriter, request apiRequest, user models.User) {
	err := request.parseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	fileOriginal, ok := storage.GetFile(request.fileInfo.id)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if fileOriginal.UserId != user.Id && !user.HasPermission(models.UserPermReplaceOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to replace this file")
		return
	}

	fileNewContent, ok := storage.GetFile(request.filemodInfo.idNewContent)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if fileNewContent.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to duplicate this file")
		return
	}

	modifiedFile, err := storage.ReplaceFile(request.fileInfo.id, request.filemodInfo.idNewContent, request.filemodInfo.deleteNewFile)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrorReplaceE2EFile):
			sendError(w, http.StatusBadRequest, "End-to-End encrypted files cannot be replaced")
		case errors.Is(err, storage.ErrorFileNotFound):
			sendError(w, http.StatusNotFound, "A file with such an ID could not be found")
		default:
			sendError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	outputFileInfo(w, modifiedFile)
}

func outputFileInfo(w http.ResponseWriter, file models.File) {
	config := configuration.Get()
	publicOutput, err := file.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
	helper.Check(err)
	result, err := json.Marshal(publicOutput)
	helper.Check(err)
	_, _ = w.Write(result)
}

func apiModifyUser(w http.ResponseWriter, request apiRequest, user models.User) {
	userEdit, ok := isValidUserForEditing(w, request)
	if !ok {
		return
	}
	if userEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, "Cannot modify super admin")
		return
	}
	if userEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot modify yourself")
		return
	}
	reqPermission := request.usermodInfo.permission
	addPerm := request.usermodInfo.grantPermission
	validPermissions := []uint16{models.UserPermReplaceUploads,
		models.UserPermListOtherUploads, models.UserPermEditOtherUploads,
		models.UserPermReplaceOtherUploads, models.UserPermDeleteOtherUploads,
		models.UserPermManageLogs, models.UserPermManageApiKeys,
		models.UserPermManageUsers}
	if !slices.Contains(validPermissions, reqPermission) {
		sendError(w, http.StatusBadRequest, "Invalid permission sent")
		return
	}

	if addPerm {
		if !userEdit.HasPermission(reqPermission) {
			userEdit.GrantPermission(reqPermission)
			database.SaveUser(userEdit, false)
			updateApiKeyPermsOnUserPermChange(userEdit.Id, reqPermission, true)
		}
		return
	}
	if userEdit.HasPermission(reqPermission) {
		userEdit.RemovePermission(reqPermission)
		database.SaveUser(userEdit, false)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, reqPermission, false)
	}
}

func apiChangeUserRank(w http.ResponseWriter, request apiRequest, user models.User) {
	userEdit, ok := isValidUserForEditing(w, request)
	if !ok {
		return
	}
	if userEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot modify yourself")
		return
	}
	if userEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, "Cannot modify super admin")
		return
	}
	switch request.usermodInfo.newRank {
	case "ADMIN":
		userEdit.UserLevel = models.UserLevelAdmin
		userEdit.Permissions = models.UserPermissionAll
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermReplaceUploads, true)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermManageUsers, true)
	case "USER":
		userEdit.UserLevel = models.UserLevelUser
		userEdit.Permissions = models.UserPermissionNone
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermReplaceUploads, false)
		updateApiKeyPermsOnUserPermChange(userEdit.Id, models.UserPermManageUsers, false)
	default:
		sendError(w, http.StatusBadRequest, "invalid rank sent")
	}
	database.SaveUser(userEdit, false)
}

func updateApiKeyPermsOnUserPermChange(userId int, userPerm uint16, isNewlyGranted bool) {
	var affectedPermission models.ApiPermission
	switch userPerm {
	case models.UserPermManageUsers:
		affectedPermission = models.ApiPermManageUsers
	case models.UserPermReplaceUploads:
		affectedPermission = models.ApiPermReplace
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

func apiResetPassword(w http.ResponseWriter, request apiRequest, user models.User) {
	userToEdit, ok := isValidUserForEditing(w, request)
	if !ok {
		return
	}
	if userToEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, "Cannot reset pw of super admin")
		return
	}
	if userToEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot reset password of yourself")
		return
	}
	userToEdit.ResetPassword = true
	password := ""
	if request.usermodInfo.setNewPw {
		password = helper.GenerateRandomString(configuration.MinLengthPassword + 2)
		userToEdit.Password = configuration.HashPassword(password, false)
	}
	database.DeleteAllSessionsByUser(userToEdit.Id)
	database.SaveUser(userToEdit, false)
	_, _ = w.Write([]byte("{\"Result\":\"ok\",\"password\":\"" + password + "\"}"))
}

func apiDeleteUser(w http.ResponseWriter, request apiRequest, user models.User) {
	userToDelete, ok := isValidUserForEditing(w, request)
	if !ok {
		return
	}
	if userToDelete.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, "Cannot delete super admin")
		return
	}
	if userToDelete.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot delete yourself")
		return
	}
	database.DeleteUser(userToDelete.Id)
	for _, file := range database.GetAllMetadata() {
		if file.UserId == userToDelete.Id {
			if request.usermodInfo.deleteUserFiles {
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

func isAuthorisedForApi(w http.ResponseWriter, request apiRequest, routing apiRoute) (models.User, bool) {
	user, ok := isValidApiKey(request.apiKey, true, routing.ApiPerm)
	if ok {
		return user, true
	}
	sendError(w, http.StatusUnauthorized, "Unauthorized")
	return models.User{}, false
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
	apiInfo     apiModInfo
	filemodInfo fileModInfo
	usermodInfo userModInfo
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

type apiModInfo struct {
	friendlyName     string
	apiKeyToModify   string
	permission       models.ApiPermission
	grantPermission  bool
	basicPermissions bool
}
type userModInfo struct {
	userId           int
	permission       uint16
	grantPermission  bool
	basicPermissions bool
	deleteUserFiles  bool
	setNewPw         bool
	newRank          string
	newUserName      string
}
type fileModInfo struct {
	id               string
	idNewContent     string
	downloads        string
	expiry           string
	password         string
	originalPassword bool
	deleteNewFile    bool
}

func parseRequest(r *http.Request) (apiRequest, error) {
	apiPermission := models.ApiPermNone
	switch r.Header.Get("permission") {
	case "PERM_VIEW":
		apiPermission = models.ApiPermView
	case "PERM_UPLOAD":
		apiPermission = models.ApiPermUpload
	case "PERM_DELETE":
		apiPermission = models.ApiPermDelete
	case "PERM_API_MOD":
		apiPermission = models.ApiPermApiMod
	case "PERM_EDIT":
		apiPermission = models.ApiPermEdit
	case "PERM_REPLACE":
		apiPermission = models.ApiPermReplace
	case "PERM_MANAGE_USERS":
		apiPermission = models.ApiPermManageUsers
	}
	userPermission := models.UserPermissionNone
	switch r.Header.Get("userpermission") {
	case "PERM_REPLACE":
		userPermission = models.UserPermReplaceUploads
	case "PERM_LIST":
		userPermission = models.UserPermListOtherUploads
	case "PERM_EDIT":
		userPermission = models.UserPermEditOtherUploads
	case "PERM_REPLACE_OTHER":
		userPermission = models.UserPermReplaceOtherUploads
	case "PERM_DELETE":
		userPermission = models.UserPermDeleteOtherUploads
	case "PERM_LOGS":
		userPermission = models.UserPermManageLogs
	case "PERM_API":
		userPermission = models.UserPermManageApiKeys
	case "PERM_USERS":
		userPermission = models.UserPermManageUsers
	}
	userId := -1
	userIdString := r.Header.Get("userid")
	if userIdString != "" {
		var err error
		userId, err = strconv.Atoi(userIdString)
		if err != nil {
			return apiRequest{}, err
		}
	}

	return apiRequest{
		apiKey:     r.Header.Get("apikey"),
		requestUrl: strings.Replace(r.URL.String(), "/api", "", 1),
		request:    r,
		fileInfo:   fileInfo{id: r.Header.Get("id")},
		filemodInfo: fileModInfo{
			id:               r.Header.Get("id"),
			idNewContent:     r.Header.Get("idNewContent"),
			downloads:        r.Header.Get("allowedDownloads"),
			expiry:           r.Header.Get("expiryTimestamp"),
			password:         r.Header.Get("password"),
			originalPassword: r.Header.Get("originalPassword") == "true",
			deleteNewFile:    r.Header.Get("deleteOriginal") == "true",
		},
		apiInfo: apiModInfo{
			friendlyName:     r.Header.Get("friendlyName"),
			apiKeyToModify:   publicKeyToApiKey(r.Header.Get("apiKeyToModify")),
			permission:       apiPermission,
			grantPermission:  r.Header.Get("permissionModifier") == "GRANT",
			basicPermissions: r.Header.Get("basicPermissions") == "true",
		},
		usermodInfo: userModInfo{
			userId:           userId,
			permission:       userPermission,
			newRank:          r.Header.Get("newRank"),
			grantPermission:  r.Header.Get("permissionModifier") == "GRANT",
			basicPermissions: r.Header.Get("basicPermissions") == "true",
			deleteUserFiles:  r.Header.Get("deleteFiles") == "true",
			newUserName:      r.Header.Get("username"),
			setNewPw:         r.Header.Get("generateNewPassword") == "true",
		},
	}, nil
}

// publicKeyToApiKey tries to convert a (possible) public key to a private key
// If not a public key or if invalid, the original value is returned
func publicKeyToApiKey(publicKey string) string {
	if len(publicKey) == lengthPublicId {
		privateApiKey, ok := database.GetApiKeyByPublicKey(publicKey)
		if ok {
			return privateApiKey
		}
	}
	return publicKey
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

// isValidApiKey checks if the API key provides is valid. If modifyTime is true, it also automatically updates
// the lastUsed timestamp
func isValidApiKey(key string, modifyTime bool, requiredPermissionApiKey models.ApiPermission) (models.User, bool) {
	if key == "" {
		return models.User{}, false
	}
	savedKey, ok := database.GetApiKey(key)
	if ok && savedKey.Id != "" && (savedKey.Expiry == 0 || savedKey.Expiry > time.Now().Unix()) {
		if modifyTime {
			savedKey.LastUsed = time.Now().Unix()
			database.UpdateTimeApiKey(savedKey)
		}
		if !savedKey.HasPermission(requiredPermissionApiKey) {
			return models.User{}, false
		}
		user, ok := database.GetUser(savedKey.UserId)
		if !ok {
			return models.User{}, false
		}
		return user, true
	}
	return models.User{}, false
}
