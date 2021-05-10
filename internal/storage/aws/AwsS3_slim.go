// +build noaws
// +build !awsmock

package aws

import (
	"Gokapi/internal/models"
	"errors"
	"io"
	"net/http"
)

const errorString = "AWS not supported in this build"

// IsCredentialProvided returns true if all credentials are provided, however does not check them to be valid
func IsCredentialProvided() bool {
	return false
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
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File) error {
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
