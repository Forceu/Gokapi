//go:build !noaws && !awsmock

package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/forceu/gokapi/internal/models"
	"io"
	"net/http"
	"strings"
	"time"
)

var awsConfig models.AwsConfig

var isCorrectLogin bool

// IsIncludedInBuild is true if Gokapi has been compiled with AWS support or the API is being mocked
const IsIncludedInBuild = true

// IsMockApi is true if the API is being mocked and therefore can only be used for testing purposes
const IsMockApi = false

// Init reads the credentials for AWS. Returns true if valid
func Init(config models.AwsConfig) bool {
	awsConfig = config
	ok, err := IsValidLogin(config)
	if err != nil {
		fmt.Println("WARNING: AWS login not successful")
		fmt.Println(err.Error())
		isCorrectLogin = false
		return false
	}
	if ok {
		fmt.Println("AWS login successful")
		isCorrectLogin = true
	}
	return ok
}

// AddBucketName adds the bucket name to the file to be stored
func AddBucketName(file *models.File) {
	file.AwsBucket = awsConfig.Bucket
}

// IsAvailable returns true if valid credentials have been passed
func IsAvailable() bool {
	return isCorrectLogin
}

// LogOut resets the credentials
func LogOut() {
	awsConfig = models.AwsConfig{}
	isCorrectLogin = false
}

// IsValidLogin checks if a valid login was provided
func IsValidLogin(config models.AwsConfig) (bool, error) {
	if !config.IsAllProvided() {
		return false, nil
	}
	tempConfig := awsConfig
	awsConfig = config
	_, _, err := FileExists(models.File{AwsBucket: awsConfig.Bucket, SHA256: "invalid"})
	awsConfig = tempConfig
	if err != nil {
		return false, err
	}
	return true, nil
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

// Download downloads a file from AWS, used for encrypted files and testing
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
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File, forceDownload bool) error {
	sess := createSession()
	s3svc := s3.New(sess)

	contentDisposition := "inline; filename=\"" + file.Name + "\""
	if forceDownload {
		contentDisposition = "Attachment; filename=\"" + file.Name + "\""
	}

	req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:                     aws.String(file.AwsBucket),
		Key:                        aws.String(file.SHA256),
		ResponseContentDisposition: aws.String(contentDisposition),
		ResponseCacheControl:       aws.String("no-store"),
		ResponseContentType:        aws.String(file.ContentType),
	})

	url, err := req.Presign(15 * time.Second)
	if err != nil {
		return err
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, int64, error) {
	sess := createSession()
	svc := s3.New(sess)

	info, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})

	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() == "NotFound" {
				return false, 0, nil
			}
		}
		return false, 0, err
	}
	return true, *info.ContentLength, nil
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

// IsCorsCorrectlySet returns true if CORS rules allow download from Gokapi
func IsCorsCorrectlySet(bucket, gokapiUrl string) (bool, error) {
	sess := createSession()
	svc := s3.New(sess)
	input := &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	}
	result, err := svc.GetBucketCors(input)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == "NoSuchCorsConfiguration" {
			return false, nil
		}
		return false, err
	}

	for _, rule := range result.CORSRules {
		for _, origin := range rule.AllowedOrigins {
			if *origin == "*" {
				return true, nil
			}
			if strings.HasPrefix(gokapiUrl, *origin) {
				return true, nil
			}
		}
	}
	return false, nil
}
