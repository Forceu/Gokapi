package filerequest

import (
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
)

// New creates a new file request object. It is not stored yet,
// and an API key has to be generated manually
func New(user models.User) models.FileRequest {
	return models.FileRequest{
		Id:           helper.GenerateRandomString(15),
		UserId:       user.Id,
		CreationDate: time.Now().Unix(),
		Name:         "Unnamed file request",
	}
}

func Get(id string) (models.FileRequest, bool) {
	result, ok := database.GetFileRequest(id)
	if !ok {
		return models.FileRequest{}, false
	}
	result.Populate(database.GetAllMetadata(), configuration.Get().MaxFileSizeMB)
	return result, true
}

func GetAll() []models.FileRequest {
	result := database.GetAllFileRequests()
	if len(result) == 0 {
		return result
	}
	allFiles := database.GetAllMetadata()
	maxServerSize := configuration.Get().MaxFileSizeMB
	for i, request := range result {
		request.Populate(allFiles, maxServerSize)
		result[i] = request
	}
	return result
}

// Delete all files associated with a file request and the request itself
func Delete(request models.FileRequest) {
	files := GetAllFiles(request)
	storage.DeleteFiles(files, true)
	database.DeleteFileRequest(request)
}

// GetAllFiles returns a list of all files associated with a file request
func GetAllFiles(request models.FileRequest) []models.File {
	var result []models.File
	files := database.GetAllMetadata()
	for _, file := range files {
		if file.UploadRequestId == request.Id {
			result = append(result, file)
		}
	}
	return result
}
