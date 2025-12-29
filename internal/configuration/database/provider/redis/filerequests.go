package redis

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixFileRequests       = "frq:"
	prefixFileRequestCounter = "frq_max"
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
func (p DatabaseProvider) GetFileRequest(id int) (models.FileRequest, bool) {
	result, ok := p.getHashMap(prefixFileRequests + strconv.Itoa(id))
	if !ok {
		return models.FileRequest{}, false
	}
	request, err := dbToFileRequest(result)
	helper.Check(err)
	return request, true
}

// GetAllFileRequests returns an array with all file requests
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

// SaveFileRequest stores the hotlink associated with the file in the database
// Returns the ID of the new request
func (p DatabaseProvider) SaveFileRequest(request models.FileRequest) int {
	if request.Id == 0 {
		id := p.getIncreasedInt(prefixFileRequestCounter)
		request.Id = id
	} else {
		counter, _ := p.getKeyInt(prefixFileRequestCounter)
		if counter < request.Id {
			p.setKey(prefixFileRequestCounter, request.Id)
		}
	}
	p.setHashMap(p.buildArgs(prefixUsers + strconv.Itoa(request.Id)).AddFlat(request))
	return request.Id
}

// DeleteFileRequest deletes a file request with the given ID
func (p DatabaseProvider) DeleteFileRequest(request models.FileRequest) {
	p.deleteKey(prefixFileRequests + strconv.Itoa(request.Id))
}
