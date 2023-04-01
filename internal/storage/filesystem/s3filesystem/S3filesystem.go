package s3filesystem

import (
	"fmt"
	"github.com/forceu/gokapi/internal/models"
	fileInterfaces "github.com/forceu/gokapi/internal/storage/filesystem/interfaces"
	"github.com/forceu/gokapi/internal/storage/filesystem/s3filesystem/aws"
	"os"
)

// GetDriver returns a driver for the AWS file system
func GetDriver() fileInterfaces.System {
	return &s3StorageDriver{}
}

type s3StorageDriver struct {
	Bucket string
}

// Config is the required configuration for the driver
type Config struct {
	// Bucket is the name of the bucket to store new files
	Bucket string
}

// MoveToFilesystem uploads a file from the local filesystem to the bucket specified in the metadata
func (d *s3StorageDriver) MoveToFilesystem(sourceFile *os.File, metaData models.File) error {
	_, err := aws.Upload(sourceFile, metaData)
	if err != nil {
		return err
	}
	err = sourceFile.Close()
	if err != nil {
		return err
	}
	return os.Remove(sourceFile.Name())
}

// Init sets the driver configurations and returns true if successful
// Requires a Config struct as input
func (d *s3StorageDriver) Init(input any) bool {
	config, ok := input.(Config)
	if !ok {
		panic("runtime exception: input for aws filesystem is not a config object")
	}
	if config.Bucket == "" {
		panic("empty bucket has been passed")
	}
	d.Bucket = config.Bucket
	return aws.IsAvailable()
}

// IsAvailable returns true if AWS is available and login was successful once
func (d *s3StorageDriver) IsAvailable() bool {
	return aws.IsAvailable()
}

// GetFile returns a File struct for the corresponding filename
func (d *s3StorageDriver) GetFile(filename string) fileInterfaces.File {
	return &awsFile{Bucket: d.Bucket, Filename: filename}
}

// FileExists returns true if the system contains a file with the given relative filepath in the bucket
func (d *s3StorageDriver) FileExists(filename string) (bool, error) {
	exists, _, err := aws.FileExists(models.File{AwsBucket: d.Bucket, SHA1: filename})
	return exists, err
}

// GetSystemName returns the name of the driver
func (d *s3StorageDriver) GetSystemName() string {
	return fileInterfaces.DriverAws
}

type awsFile struct {
	Bucket   string
	Filename string
}

func (f *awsFile) Exists() bool {
	exists, _, err := aws.FileExists(models.File{AwsBucket: f.Bucket, SHA1: f.Filename})
	if err != nil {
		fmt.Println(err)
		return false
	}
	return exists
}

func (f *awsFile) GetName() string {
	return f.Filename
}
