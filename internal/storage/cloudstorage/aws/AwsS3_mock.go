//go:build !noaws && awsmock

package aws

import (
	"bytes"
	"errors"
	"github.com/forceu/gokapi/internal/models"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var uploadedFiles []models.File
var isCorrectLogin bool

const (
	region     = "mock-region-1"
	bucketName = "gokapi-test"
	accessId   = "accId"
	accessKey  = "accKey"
)

// IsIncludedInBuild is true if Gokapi has been compiled with AWS support or the API is being mocked
const IsIncludedInBuild = true

// IsMockApi is true if the API is being mocked and therefore can only be used for testing purposes
const IsMockApi = true

// Init reads the credentials for AWS
func Init(config models.AwsConfig) bool {
	if !isValidCredentials() {
		return false
	}
	Upload(bytes.NewReader([]byte("test")), models.File{
		Id:        "awsTest1234567890123",
		Name:      "aws Test File",
		Size:      "20 MB",
		SHA256:    "x341354656543213246465465465432456898794",
		AwsBucket: "gokapi-test",
	})
	return true
}

// IsAvailable returns true if valid credentials have been passed
func IsAvailable() bool {
	return isCorrectLogin
}

// IsValidLogin checks if a valid login was provided
func IsValidLogin(config models.AwsConfig) (bool, error) {
	return isValidCredentials(), nil
}

// LogOut resets the credentials
func LogOut() {
	isCorrectLogin = false
}

// AddBucketName adds the bucket name to the file to be stored
func AddBucketName(file *models.File) {
	file.AwsBucket = bucketName
}

func isValidCredentials() bool {
	requiredKeys := []string{"GOKAPI_AWS_BUCKET", "GOKAPI_AWS_REGION", "GOKAPI_AWS_KEY", "GOKAPI_AWS_KEY_SECRET"}
	requiredValues := []string{bucketName, region, accessId, accessKey}
	for i, key := range requiredKeys {
		val, _ := os.LookupEnv(key)
		if val != requiredValues[i] {
			isCorrectLogin = false
			return false
		}
	}
	isCorrectLogin = true
	return true
}

// IsCredentialProvided returns true if all credentials are provided, however does not check them to be valid
func IsCredentialProvided(checkIfValid bool) bool {
	requiredKeys := []string{"GOKAPI_AWS_BUCKET", "AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"}
	for _, key := range requiredKeys {
		if !isValidEnv(key) {
			return false
		}
	}
	return true
}

func isValidEnv(key string) bool {
	val, ok := os.LookupEnv(key)
	return ok && val != ""
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	if !isValidCredentials() {
		return "", errors.New("invalid credentials / invalid bucket / invalid region")
	}

	if !isUploaded(file) {
		uploadedFiles = append(uploadedFiles, file)
	}
	return "", nil
}

// Download downloads a file from AWS
func Download(writer io.WriterAt, file models.File) (int64, error) {
	if !isValidCredentials() {
		return 0, errors.New("invalid credentials / invalid bucket / invalid region")
	}

	if isUploaded(file) {
		return strconv.ParseInt(file.Size, 10, 64)
	}
	return 0, errors.New("file not found")
}

func isUploaded(file models.File) bool {
	for _, element := range uploadedFiles {
		if element.SHA256 == file.SHA256 {
			return true
		}
	}
	return false
}

// RedirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File, forceDownload bool) error {
	if !isValidCredentials() {
		return errors.New("invalid credentials / invalid bucket / invalid region")
	}

	if isUploaded(file) {
		http.Redirect(w, r, "https://redirect.url", http.StatusTemporaryRedirect)
		return nil
	}
	return errors.New("file not found")
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, int64, error) {
	if !isValidCredentials() {
		return false, 0, errors.New("invalid credentials / invalid bucket / invalid region")
	}
	if !isUploaded(file) {
		return false, 0, nil
	}

	return true, 10000, nil
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	if !isValidCredentials() {
		return false, errors.New("invalid credentials / invalid bucket / invalid region")
	}
	var buffer []models.File

	for _, element := range uploadedFiles {
		if element.SHA256 != file.SHA256 {
			buffer = append(buffer, element)
		}
	}
	uploadedFiles = buffer

	return true, nil
}

// IsCorsCorrectlySet returns true if CORS rules allow download from Gokapi
func IsCorsCorrectlySet(bucket, gokapiUrl string) (bool, error) {
	switch bucket {
	case "any":
		return true, nil
	case "none":
		return false, nil
	case "forbidden":
		return false, errors.New("forbidden")
	case "https":
		return strings.HasPrefix(gokapiUrl, "https://"), nil
	case "url":
		return strings.HasPrefix(gokapiUrl, "http://test.com"), nil
	default:
		return false, errors.New("unknown")
	}
}
