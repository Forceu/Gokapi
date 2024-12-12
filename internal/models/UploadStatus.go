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
	// This is only used for saving in the database and thereafter requesting via /metadataFromChunk
	// The FileId is not propagated through SSE
	FileId string
}
