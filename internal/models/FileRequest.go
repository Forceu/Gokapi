package models

import "time"

type FileRequest struct {
	Id            int    `json:"id" redis:"id"`                     // The internal ID of the file request
	Owner         int    `json:"owner" redis:"owner"`               // The user ID of the owner
	MaxFiles      int    `json:"maxfiles" redis:"maxfiles"`         // The maximum number of files allowed
	MaxSize       int    `json:"maxsize" redis:"maxsize"`           // The maximum file size allowed in MB
	Expiry        int64  `json:"expiry" redis:"expiry"`             // The expiry time of the file request
	CreationDate  int64  `json:"creationdate" redis:"creationdate"` // The timestamp of the creation of the file request
	Name          string `json:"name" redis:"name"`                 // The given name for the file request
	UploadedFiles int    `json:"uploadedfiles" redis:"-"`           // Contains the number of uploaded files for this request. Needs to be calculated with Populate()
	LastUpload    int64  `json:"lastupload" redis:"-"`              // Contains the timestamp of the last upload for this request. Needs to be calculated with Populate()

}

func (f *FileRequest) Populate(files map[string]File) {
	for _, file := range files {
		if file.UploadRequestId == f.Id {
			f.UploadedFiles++
			if file.UploadDate > f.LastUpload {
				f.LastUpload = file.UploadDate
			}
		}
	}
}

// GetReadableDateExpiry returns the expiry date as YYYY-MM-DD HH:MM:SS
func (f *FileRequest) GetReadableDateExpiry() string {
	if f.Expiry == 0 {
		return "Never"
	}
	if time.Now().Unix() > f.Expiry {
		return "Expired"
	}
	return time.Unix(f.Expiry, 0).Format("2006-01-02 15:04:05")
}

// GetReadableDateLastUpdate returns the last update date as YYYY-MM-DD HH:MM:SS
func (f *FileRequest) GetReadableDateLastUpdate() string {
	if f.LastUpload == 0 {
		return "None"
	}
	return time.Unix(f.LastUpload, 0).Format("2006-01-02 15:04:05")
}
