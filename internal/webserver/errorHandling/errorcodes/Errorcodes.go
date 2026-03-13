package errorcodes

const (
	// UnspecifiedError is a generic error
	UnspecifiedError = iota
	// InvalidUrl is returned when the URL is invalid
	InvalidUrl
	// InvalidApiKey is returned when the API key is invalid
	InvalidApiKey
	// AdminOnly is returned when the user is not an admin
	AdminOnly
	// CannotParse is returned when the API cannot parse the request
	CannotParse
	// NotFound is returned when a resource does not exist
	NotFound
	// NoPermission is returned when the user or api key does not have the required permission
	NoPermission
	// AlreadyExists is returned when a resource already exists
	AlreadyExists
	// InternalServer is returned when an internal server error occurs
	InternalServer
	// FileTooLarge is returned when an uploaded file is too large
	FileTooLarge
	// InvalidUserInput is returned when the user input is invalid
	InvalidUserInput
	// ChunkTooSmall is returned when a chunk is too small
	ChunkTooSmall
	// InvalidChunkReservation is returned when a chunk has no or an expired reservation
	InvalidChunkReservation
	// CannotAllocateFile is returned when a file cannot be allocated, e.g. with not enough disk space available
	CannotAllocateFile
	// RequestExpired is returned when a request has expired
	RequestExpired
	// CannotUploadMoreFiles is returned when the user has reached the maximum number of files allowed for this file request
	CannotUploadMoreFiles
	//RateLimited is returned when the user has reached the maximum number of file uploads allowed per second
	RateLimited
	// EndToEndNotSupported is returned when action is not possible with end-to-end encrypted files
	EndToEndNotSupported
	// UnsupportedFile is returned when an action is not possible with the given file
	UnsupportedFile
	// ResourceCanNotBeEdited is returned when a resource cannot be edited
	ResourceCanNotBeEdited
)
