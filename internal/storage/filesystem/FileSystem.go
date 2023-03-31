package filesystem

import (
	"github.com/forceu/gokapi/internal/storage/filesystem/interfaces"
	"github.com/forceu/gokapi/internal/storage/filesystem/localstorage"
	"github.com/forceu/gokapi/internal/storage/filesystem/s3filesystem"
	"github.com/forceu/gokapi/internal/storage/filesystem/s3filesystem/aws"
	"log"
)

var dataFilesystem interfaces.System
var s3FileSystem interfaces.System

// ActiveStorageSystem is a driver for the storage system that is in use currently. Can be either
// the local filesystem or S3, depending on the configuration
var ActiveStorageSystem interfaces.System

// Init initializes the filesystems and must be called on start
func Init(pathData string) {
	dataFilesystem = localstorage.GetDriver()
	dataFilesystem.Init(localstorage.Config{
		DataPath: pathData,
	})
	ActiveStorageSystem = dataFilesystem
}

// SetAws sets the AWS filesystem as the default storage
func SetAws() {
	if aws.IsIncludedInBuild {
		s3FileSystem = s3filesystem.GetDriver()
		ok := s3FileSystem.Init(s3filesystem.Config{Bucket: aws.GetDefaultBucketName()})
		if !ok && !isUnitTesting {
			log.Println("Unable to set AWS S3 as filesystem")
			return
		}
		ActiveStorageSystem = s3FileSystem
	}
}

// SetLocal sets the local filesystem as the default storage
func SetLocal() {
	ActiveStorageSystem = dataFilesystem
}

// isUnitTesting is only set to true when testing, to avoid login with aws
var isUnitTesting = false
