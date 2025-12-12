package models

type FileRequest struct {
	Id       int    `json:"id" redis:"id"`             // The internal ID of the file request
	Owner    int    `json:"owner" redis:"owner"`       // The user ID of the owner
	MaxFiles int    `json:"maxfiles" redis:"maxfiles"` // The maximum number of files allowed
	MaxSize  int    `json:"maxsize" redis:"maxsize"`   // The maximum file size allowed in MB
	Expiry   int64  `json:"expiry" redis:"expiry"`     // The expiry time of the file request
	Name     string `json:"name" redis:"name"`         // The given name for the file request
}
