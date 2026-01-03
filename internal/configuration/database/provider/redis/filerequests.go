package redis

import (
	"cmp"
	"slices"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixFileRequests = "frq:"
)

func dbToFileRequest(input []any) (models.FileRequest, error) {
	var result models.FileRequest
	err := redigo.ScanStruct(input, &result)
	if err != nil {
		return models.FileRequest{}, err
	}
	return result, nil
}

// GetFileRequest returns the FileRequest or false if not found
func (p DatabaseProvider) GetFileRequest(id string) (models.FileRequest, bool) {
	result, ok := p.getHashMap(prefixFileRequests + id)
	if !ok {
		return models.FileRequest{}, false
	}
	request, err := dbToFileRequest(result)
	helper.Check(err)
	return request, true
}

// GetAllFileRequests returns an array with all file requests, ordered by creation date
func (p DatabaseProvider) GetAllFileRequests() []models.FileRequest {
	var result []models.FileRequest
	maps := p.getAllHashesWithPrefix(prefixFileRequests)
	for _, v := range maps {
		request, err := dbToFileRequest(v)
		helper.Check(err)
		result = append(result, request)
	}
	return sortFilerequests(result)
}

func sortFilerequests(users []models.FileRequest) []models.FileRequest {
	slices.SortFunc(users, func(a, b models.FileRequest) int {
		return cmp.Or(
			cmp.Compare(b.CreationDate, a.CreationDate),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return users
}

// SaveFileRequest stores the file request associated with the file in the database
func (p DatabaseProvider) SaveFileRequest(request models.FileRequest) {
	p.setHashMap(p.buildArgs(prefixUsers + request.Id).AddFlat(request))
}

// DeleteFileRequest deletes a file request with the given ID
func (p DatabaseProvider) DeleteFileRequest(request models.FileRequest) {
	p.deleteKey(prefixFileRequests + request.Id)
}
