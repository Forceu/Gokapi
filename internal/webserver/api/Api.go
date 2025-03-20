package api

import (
	"encoding/json"
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"io"
	"net/http"
	"strings"
	"time"
)

const lengthPublicId = 35
const lengthApiKey = 30
const minLengthUser = 4

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	requestUrl := parseRequestUrl(r)

	routing, ok := getRouting(requestUrl)
	if !ok {
		sendError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	var user models.User
	user, ok = isAuthorisedForApi(r, routing)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if routing.RequestParser == nil {
		routing.Continue(w, nil, user)
		return
	}
	parser := routing.RequestParser.New()
	err := parser.ParseRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
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
		sendError(w, http.StatusNotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermEditOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to edit file.")
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
			file.ExpireAtString = storage.FormatTimestamp(request.ExpiryTimestamp)
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
	outputFileInfo(w, file)
}

// generateNewKey generates and saves a new API key
func generateNewKey(defaultPermissions bool, userId int, friendlyName string) models.ApiKey {
	if friendlyName == "" {
		friendlyName = "Unnamed key"
	}
	newKey := models.ApiKey{
		Id:           helper.GenerateRandomString(lengthApiKey),
		PublicId:     helper.GenerateRandomString(lengthPublicId),
		FriendlyName: friendlyName,
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
	if !user.HasPermissionManageLogs() {
		tempKey.RemovePermission(models.ApiPermManageLogs)
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

func apiDeleteKey(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthDelete)
	if !ok {
		panic("invalid parameter passed")
	}
	apiKeyOwner, apiKey, ok := isValidKeyForEditing(request.KeyId)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to delete this API key")
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
		sendError(w, http.StatusNotFound, "Invalid key ID provided.")
		return
	}
	if apiKeyOwner.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to delete this API key")
		return
	}

	switch request.Permission {
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
	case models.ApiPermManageLogs:
		if !apiKeyOwner.HasPermissionManageLogs() {
			sendError(w, http.StatusUnauthorized, "Insufficient user permission for owner to set this API permission")
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
		sendError(w, http.StatusNotFound, "Invalid user id provided.")
		return models.User{}, false
	}
	return user, true
}

func apiCreateApiKey(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramAuthCreate)
	if !ok {
		panic("invalid parameter passed")
	}
	key := generateNewKey(request.BasicPermissions, user.Id, request.FriendlyName)
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
	if len(request.Username) < minLengthUser {
		sendError(w, http.StatusBadRequest, "Invalid username provided.")
		return
	}
	_, ok = database.GetUserByName(request.Username)
	if ok {
		sendError(w, http.StatusConflict, "User already exists.")
		return
	}
	newUser := models.User{
		Name:      request.Username,
		UserLevel: models.UserLevelUser,
	}
	database.SaveUser(newUser, true)
	newUser, ok = database.GetUserByName(request.Username)
	if !ok {
		sendError(w, http.StatusInternalServerError, "Could not save user")
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
		sendError(w, http.StatusNotFound, "Invalid key ID provided.")
		return
	}
	if ownerApiKey.Id != user.Id && !user.HasPermission(models.UserPermManageApiKeys) {
		sendError(w, http.StatusUnauthorized, "No permission to edit this key")
		return
	}
	err := renameApiKeyFriendlyName(apiKey.Id, request.FriendlyName)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
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
		sendError(w, http.StatusNotFound, "Invalid file ID provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermDeleteOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to delete this file")
		return
	}
	logging.LogDelete(file, user)
	_ = storage.DeleteFile(request.Id, true)
}

func apiChunkAdd(w http.ResponseWriter, r requestParser, _ models.User) {
	request, ok := r.(*paramChunkAdd)
	if !ok {
		panic("invalid parameter passed")
	}
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.Request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, storage.ErrorFileTooLarge.Error())
		return
	}

	request.Request.Body = http.MaxBytesReader(w, request.Request.Body, maxUpload)
	err := fileupload.ProcessNewChunk(w, request.Request, true)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
}

func apiChunkComplete(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramChunkComplete)
	if !ok {
		panic("invalid parameter passed")
	}
	if request.IsNonBlocking {
		go doBlockingPartCompleteChunk(nil, request, user)
		_, _ = io.WriteString(w, "{\"result\":\"OK\"}")
		return
	} else {
		doBlockingPartCompleteChunk(w, request, user)
	}

}

func doBlockingPartCompleteChunk(w http.ResponseWriter, request *paramChunkComplete, user models.User) {
	uploadRequest := fileupload.CreateUploadConfig(request.AllowedDownloads,
		request.ExpiryDays,
		request.Password,
		request.UnlimitedTime,
		request.UnlimitedDownloads,
		request.IsE2E,
		request.FileSize)
	file, err := fileupload.CompleteChunk(request.Uuid, request.FileHeader, user.Id, uploadRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	logging.LogUpload(file, user)
	if w != nil {
		config := configuration.Get()
		_, _ = io.WriteString(w, file.ToJsonResult(config.ServerUrl, config.IncludeFilename))
	}
}

func apiList(w http.ResponseWriter, _ requestParser, user models.User) {
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

func apiListSingle(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesListSingle)
	if !ok {
		panic("invalid parameter passed")
	}
	id := strings.TrimPrefix(request.RequestUrl, "/files/list/")
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

func apiUploadFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesAdd)
	if !ok {
		panic("invalid parameter passed")
	}
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	if request.Request.ContentLength > maxUpload {
		sendError(w, http.StatusBadRequest, storage.ErrorFileTooLarge.Error())
		return
	}

	request.Request.Body = http.MaxBytesReader(w, request.Request.Body, maxUpload)
	err := fileupload.ProcessCompleteFile(w, request.Request, user.Id, configuration.Get().MaxMemory)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
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
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if file.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to duplicate this file")
		return
	}
	uploadRequest := fileupload.CreateUploadConfig(request.AllowedDownloads,
		request.ExpiryDays,
		request.Password,
		request.UnlimitedTime,
		request.UnlimitedDownloads,
		false, // is not being used by storage.DuplicateFile
		0)     // is not being used by storage.DuplicateFile
	newFile, err := storage.DuplicateFile(file, request.RequestedChanges, request.FileName, uploadRequest)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	outputFileInfo(w, newFile)
}

func apiReplaceFile(w http.ResponseWriter, r requestParser, user models.User) {
	request, ok := r.(*paramFilesReplace)
	if !ok {
		panic("invalid parameter passed")
	}
	fileOriginal, ok := storage.GetFile(request.Id)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if fileOriginal.UserId != user.Id && !user.HasPermission(models.UserPermReplaceOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to replace this file")
		return
	}

	fileNewContent, ok := storage.GetFile(request.IdNewContent)
	if !ok {
		sendError(w, http.StatusNotFound, "Invalid id provided.")
		return
	}
	if fileNewContent.UserId != user.Id && !user.HasPermission(models.UserPermListOtherUploads) {
		sendError(w, http.StatusUnauthorized, "No permission to duplicate this file")
		return
	}

	modifiedFile, err := storage.ReplaceFile(request.Id, request.IdNewContent, request.Delete)
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
	logging.LogReplace(fileOriginal, modifiedFile, user)
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
		sendError(w, http.StatusBadRequest, "Cannot modify super admin")
		return
	}
	if userEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot modify yourself")
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
		sendError(w, http.StatusBadRequest, "Cannot modify yourself")
		return
	}
	if userEdit.IsSuperAdmin() {
		sendError(w, http.StatusBadRequest, "Cannot modify super admin")
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
		sendError(w, http.StatusBadRequest, "invalid rank sent")
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
		sendError(w, http.StatusBadRequest, "Cannot reset pw of super admin")
		return
	}
	if userToEdit.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot reset password of yourself")
		return
	}
	userToEdit.ResetPassword = true
	password := ""
	if request.NewPassword {
		password = helper.GenerateRandomString(configuration.MinLengthPassword + 2)
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
		sendError(w, http.StatusBadRequest, "Cannot delete super admin")
		return
	}
	if userToDelete.IsSameUser(user.Id) {
		sendError(w, http.StatusBadRequest, "Cannot delete yourself")
		return
	}
	logging.LogUserDeletion(userToDelete, user)
	database.DeleteUser(userToDelete.Id)
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

func isAuthorisedForApi(r *http.Request, routing apiRoute) (models.User, bool) {
	apiKey := r.Header.Get("apikey")
	user, _, ok := isValidApiKey(apiKey, true, routing.ApiPerm)
	if !ok {
		return models.User{}, false
	}
	return user, true
}

// Probably from new API permission system
func sendError(w http.ResponseWriter, errorInt int, errorMessage string) {
	if w == nil {
		return
	}
	w.WriteHeader(errorInt)
	_, _ = w.Write([]byte("{\"Result\":\"error\",\"ErrorMessage\":\"" + errorMessage + "\"}"))
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
