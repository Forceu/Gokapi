package filerequest

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
)

func Get(id int) (models.FileRequest, bool) {
	result, ok := database.GetFileRequest(id)
	if !ok {
		return models.FileRequest{}, false
	}
	result.Populate(database.GetAllMetadata())
	return result, true
}

func GetAll() []models.FileRequest {
	result := database.GetAllFileRequests()
	if len(result) == 0 {
		return result
	}
	allFiles := database.GetAllMetadata()
	for _, request := range result {
		request.Populate(allFiles)
	}
	return result
}

// Delete all files associated with a file request and the request itself
func Delete(request models.FileRequest) {
	files := GetAllFiles(request)
	for _, file := range files {
		storage.DeleteFile(file.Id, true)
	}
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
