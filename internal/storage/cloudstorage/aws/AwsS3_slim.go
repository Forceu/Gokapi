//go:build noaws
// +build noaws

package aws

import (
	"errors"
	"github.com/forceu/gokapi/internal/models"
	"io"
	"net/http"
)

const errorString = "AWS not supported in this build"

// IsIncludedInBuild is true if Gokapi has been compiled with AWS support or the API is being mocked
const IsIncludedInBuild = false

// IsMockApi is true if the API is being mocked and therefore can only be used for testing purposes
const IsMockApi = false

// Init reads the credentials for AWS
func Init(config models.AwsConfig) bool {
	return false
}

// IsAvailable returns true if valid credentials have been passed
func IsAvailable() bool {
	return false
}

// AddBucketName adds the bucket name to the file to be stored
func AddBucketName(file *models.File) {
	return
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	return "", errors.New(errorString)
}

// Download downloads a file from AWS
func Download(writer io.WriterAt, file models.File) (int64, error) {
	return 0, errors.New(errorString)
}

// RedirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File, forceDownload bool) error {
	return errors.New(errorString)
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, error) {
	return true, errors.New(errorString)
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	return false, errors.New(errorString)
}
