package errorcodes

const (
	UnspecifiedError = iota
	InvalidUrl
	InvalidApiKey
	AdminOnly
	CannotParse
	NotFound
	NoPermission
	AlreadyExists
	InternalServer
	FileTooLarge
	InvalidUserInput
	ChunkTooSmall
	InvalidChunkReservation
	CannotAllocateFile
	RequestExpired
	CannotUploadMoreFiles
	RateLimited
	EndToEndNotSupported
	UnsupportedFile
	ResourceCanNotBeEdited
)
