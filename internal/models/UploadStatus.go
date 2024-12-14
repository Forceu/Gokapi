package models

// UploadStatus contains information about the current status of a file upload
type UploadStatus struct {
	// ChunkId is the identifier for the chunk
	ChunkId string
	// CurrentStatus indicates if the chunk is currently being processed (e.g. encrypting or
	// hashing) or being moved/uploaded to the file storage
	// See ProcessingStatus for definition
	CurrentStatus int
	// FileId is populated, once a file has been created from a chunk
	FileId string
	// ErrorMessage is empty, unless an error occurred
	ErrorMessage string `json:"errormessage"`
	// Creation is the unix time when the status was created and is populated automatically
	Creation int64
}
