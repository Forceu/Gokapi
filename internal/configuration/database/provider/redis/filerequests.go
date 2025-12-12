package redis

import (
	"strconv"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	prefixFileRequests       = "frq:"
	prefixFileRequestCounter = "frq_max"
)

type schemaFileRequests struct {
	Id       int
	Name     string
	Owner    int
	Expiry   int64
	MaxFiles int
	MaxSize  int
}

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

// GetAllFileRequests returns an array with all file requests
func (p DatabaseProvider) GetAllFileRequests() []models.FileRequest {
	var result []models.FileRequest
	maps := p.getAllHashesWithPrefix(prefixFileRequests)
	for _, v := range maps {
		request, err := dbToFileRequest(v)
		helper.Check(err)
		result = append(result, request)
	}
	return result
}

// SaveFileRequest stores the hotlink associated with the file in the database
func (p DatabaseProvider) SaveFileRequest(request models.FileRequest) {
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
}

// DeleteFileRequest deletes a file request with the given ID
func (p DatabaseProvider) DeleteFileRequest(id int) {
	p.deleteKey(prefixFileRequests + strconv.Itoa(id))
}
