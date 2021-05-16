// +build !noaws,!awsmock

package aws

import (
	"Gokapi/internal/configuration/cloudconfig"
	"Gokapi/internal/environment"
	"Gokapi/internal/models"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"net/http"
	"time"
)

var awsConfig models.AwsConfig
var environmentHolder environment.Environment

// IsAvailable is true if Gokapi has been compiled with AWS support or the API is being mocked
const IsAvailable = true

// IsMockApi is true if the API is being mocked and therefore can only be used for testing purposes
const IsMockApi = false

// IsCredentialProvided returns true if all credentials are provided
func IsCredentialProvided(checkIfValid bool) bool {
	e := getEnvironment()
	isProvided := e.IsAwsProvided()
	if !isProvided {
		return false
	}
	if checkIfValid {
		return isValidLogin()
	}
	return true
}

func getEnvironment() *environment.Environment {
	if environmentHolder == (environment.Environment{}) {
		environmentHolder = environment.New()
	}
	return &environmentHolder
}

// Init reads the credentials for AWS
func Init() {
	config, ok := cloudconfig.Load()
	if ok {
		awsConfig = config.Aws
	}
}

func isValidLogin() bool {
	sess := createSession()
	svc := s3.New(sess)
	_, err := svc.Config.Credentials.Get()
	if err != nil {
		fmt.Println("WARNING: AWS login not successful: " + err.Error())
		return false
	}
	fmt.Println("AWS login successful")
	return true
}

func createSession() *session.Session {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(awsConfig.KeyId, awsConfig.KeySecret, ""),
		Endpoint:         aws.String(awsConfig.Endpoint),
		Region:           aws.String(awsConfig.Region),
		S3ForcePathStyle: aws.Bool(true),
	}
	return session.Must(session.NewSession(s3Config))
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	sess := createSession()
	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
		Body:   input,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}

// Download downloads a file from AWS
func Download(writer io.WriterAt, file models.File) (int64, error) {
	sess := createSession()
	downloader := s3manager.NewDownloader(sess)

	size, err := downloader.Download(writer, &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// RedirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File) error {
	sess := createSession()
	s3svc := s3.New(sess)

	req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:                     aws.String(file.AwsBucket),
		Key:                        aws.String(file.SHA256),
		ResponseContentDisposition: aws.String("filename=" + file.Name),
	})

	url, err := req.Presign(15 * time.Second)
	if err != nil {
		return err
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, error) {
	sess := createSession()
	svc := s3.New(sess)

	_, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})

	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	sess := createSession()
	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})

	if err != nil {
		return false, err
	}
	return true, nil
}
