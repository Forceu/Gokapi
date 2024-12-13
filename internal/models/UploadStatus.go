package models

// UploadStatus contains information about the current status of a file upload
type UploadStatus struct {
	// ChunkId is the identifier for the chunk
	ChunkId string `json:"chunkid"`
	// CurrentStatus indicates if the chunk is currently being processed (e.g. encrypting or
	// hashing) or being moved/uploaded to the file storage
	// See processingstatus for definition
	CurrentStatus int `json:"currentstatus"`
	// FileId is populated, once a file has been created from a chunk
	FileId string
	// ErrorMessage is not saved to the database, only sent through SSE
	ErrorMessage string `json:"errormessage"`
}
