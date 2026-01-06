package models

import (
	"strings"
	"time"

	"github.com/forceu/gokapi/internal/helper"
)

type FileRequest struct {
	Id            string   `json:"id" redis:"id"`                     // The internal ID of the file request
	UserId        int      `json:"userid" redis:"userid"`             // The user ID of the owner
	MaxFiles      int      `json:"maxfiles" redis:"maxfiles"`         // The maximum number of files allowed
	MaxSize       int      `json:"maxsize" redis:"maxsize"`           // The maximum file size allowed in MB
	Expiry        int64    `json:"expiry" redis:"expiry"`             // The expiry time of the file request
	CreationDate  int64    `json:"creationdate" redis:"creationdate"` // The timestamp of the file request creation
	Name          string   `json:"name" redis:"name"`                 // The given name for the file request
	ApiKey        string   `json:"apikey" redis:"apikey"`             // The API key related to the file request
	UploadedFiles int      `json:"uploadedfiles" redis:"-"`           // Contains the number of uploaded files for this request. Needs to be calculated with Populate()
	LastUpload    int64    `json:"lastupload" redis:"-"`              // Contains the timestamp of the last upload for this request. Needs to be calculated with Populate()
	TotalFileSize int64    `json:"totalfilesize" redis:"-"`           // Contains the file size of all uploaded files. Needs to be calculated with Populate()
	FileIdList    []string `json:"fileidlist" redis:"-"`              // Contains an array of the IDs of all uploaded files. Needs to be calculated with Populate()
	Files         []File   `json:"-" redis:"-"`                       // Contains an array of the IDs of all uploaded files. Needs to be calculated with Populate()
}

// Populate inserts the number of uploaded files and the last upload date
func (f *FileRequest) Populate(files map[string]File) {
	f.FileIdList = make([]string, 0)
	f.Files = make([]File, 0)
	for _, file := range files {
		if file.UploadRequestId == f.Id {
			f.UploadedFiles++
			f.TotalFileSize = f.TotalFileSize + file.SizeBytes
			f.FileIdList = append(f.FileIdList, file.Id)
			f.Files = append(f.Files, file)
			if file.UploadDate > f.LastUpload {
				f.LastUpload = file.UploadDate
			}
		}
	}
}

// GetReadableDateLastUpdate returns the last update date as YYYY-MM-DD HH:MM:SS
func (f *FileRequest) GetReadableDateLastUpdate() string {
	if f.LastUpload == 0 {
		return "None"
	}
	return time.Unix(f.LastUpload, 0).Format("2006-01-02 15:04:05")
}

func (f *FileRequest) GetReadableTotalSize() string {
	return helper.ByteCountSI(f.TotalFileSize)
}

func (f *FileRequest) GetFilesAsString() string {
	return strings.Join(f.FileIdList, ",")
}

func (f *FileRequest) IsUnlimitedSize() bool {
	return f.MaxSize == 0
}
func (f *FileRequest) IsUnlimitedFiles() bool {
	return f.MaxFiles == 0
}
