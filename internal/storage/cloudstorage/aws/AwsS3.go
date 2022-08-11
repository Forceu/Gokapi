//go:build !noaws && !awsmock

package aws

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
	if config.Endpoint != "" && !strings.HasPrefix(config.Endpoint, "http") {
		config.Endpoint = "https://" + config.Endpoint
	}
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
	_, _, err := FileExists(models.File{AwsBucket: awsConfig.Bucket, SHA1: "invalid"})
	awsConfig = tempConfig
	if err != nil {
		return false, err
	}
	return true, nil
}

func createClient() (*s3.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if awsConfig.Endpoint != "" && service == s3.ServiceID && region == awsConfig.Region {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           awsConfig.Endpoint,
				SigningRegion: awsConfig.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsConfig.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsConfig.KeyId, awsConfig.KeySecret, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.Region = awsConfig.Region
	})
	return client, nil
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	client, err := createClient()
	if err != nil {
		return "", err
	}

	uploader := manager.NewUploader(client)
	result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
		Body:   input,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}

// Download downloads a file from AWS, used for encrypted files and testing
func Download(writer io.WriterAt, file models.File) (int64, error) {
	client, err := createClient()
	if err != nil {
		return 0, err
	}
	downloader := manager.NewDownloader(client)

	size, err := downloader.Download(context.TODO(), writer, &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// RedirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File, forceDownload bool) error {
	client, err := createClient()
	if err != nil {
		return err
	}

	contentDisposition := "inline; filename=\"" + file.Name + "\""
	if forceDownload {
		contentDisposition = "Attachment; filename=\"" + file.Name + "\""
	}
	presignClient := s3.NewPresignClient(client)
	presignResult, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket:                     aws.String(file.AwsBucket),
		Key:                        aws.String(file.SHA1),
		ResponseContentDisposition: aws.String(contentDisposition),
		ResponseCacheControl:       aws.String("no-store"),
		ResponseContentType:        aws.String(file.ContentType),
	}, s3.WithPresignExpires(15*time.Second))
	if err != nil {
		return err
	}

	http.Redirect(w, r, presignResult.URL, http.StatusTemporaryRedirect)
	return nil
}

func getTimeoutContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	rContext, rCancel := context.WithTimeout(ctx, 5*time.Second)
	return rContext, func() {
		if rCancel != nil {
			rCancel()
		}
	}
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, int64, error) {
	client, err := createClient()
	if err != nil {
		return false, 0, err
	}

	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()

	info, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})

	if err != nil {
		var errorNotFound *types.NotFound
		if errors.As(err, &errorNotFound) {
			return false, 0, nil
		}
		fmt.Println(err.Error()) // TODO
		if true {                // TODO
			return false, 0, errors.New("Timeout - could not connect to " + awsConfig.Endpoint)
		}
		return false, 0, err
	}
	return true, info.ContentLength, nil
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	client, err := createClient()
	if err != nil {
		return false, err
	}
	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

// IsCorsCorrectlySet returns true if CORS rules allow download from Gokapi
func IsCorsCorrectlySet(bucket, gokapiUrl string) (bool, error) {
	client, err := createClient()
	if err != nil {
		return false, err
	}
	input := &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	}
	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()

	result, err := client.GetBucketCors(ctx, input)
	if err != nil {
		if err.Error() == "NoSuchCorsConfiguration" { // TODO
			return false, nil
		}
		return false, err
	}

	for _, rule := range result.CORSRules {
		for _, origin := range rule.AllowedOrigins {
			if origin == "*" {
				return true, nil
			}
			if strings.HasPrefix(gokapiUrl, origin) {
				return true, nil
			}
		}
	}
	return false, nil
}
