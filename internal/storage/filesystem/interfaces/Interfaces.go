package interfaces

import (
	"github.com/forceu/gokapi/internal/models"
	"os"
)

// DriverLocal is returned as a name for the Local Storage driver
const DriverLocal = "localstorage"

// DriverAws is returned as a name for the AWS Storage driver
const DriverAws = "awss3"

// File contains information about the stored file
type File interface {
	// Exists returns true if the file exists
	Exists() bool
	// GetName returns the name of the file
	GetName() string
}

// System is a driver for storing and retrieving files
type System interface {
	// Init sets the driver configurations and returns true if successful
	Init(input any) bool
	// IsAvailable returns true if the driver can be used
	IsAvailable() bool
	// GetSystemName returns the name of the driver
	GetSystemName() string
	// MoveToFilesystem moves a file from the local filesystem to the driver's filesystem
	MoveToFilesystem(sourceFile *os.File, metaData models.File) error
	// GetFile returns a File struct for the corresponding filename
	GetFile(filename string) File
	// FileExists returns true if the system contains a file with the given relative filepath
	FileExists(filepath string) (bool, error)
}
