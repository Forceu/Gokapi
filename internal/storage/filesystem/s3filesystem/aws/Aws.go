//go:build !noaws && !awsmock

package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/headers"
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
	_, _, err := FileExists(models.File{AwsBucket: awsConfig.Bucket, SHA1: "invalid"})
	awsConfig = tempConfig
	if err != nil {
		return false, err
	}
	return true, nil
}

// createClient builds a fresh S3 client from the current awsConfig, pointing it at the
// (potentially S3-compatible, e.g. Backblaze B2) endpoint using forced path-style addressing.
func createClient() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsConfig.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsConfig.KeyId, awsConfig.KeySecret, "")),
	)
	if err != nil {
		panic(err)
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(awsConfig.Endpoint)
		o.UsePathStyle = true
	})
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	uploader := manager.NewUploader(createClient())

	result, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
		Body:   input,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}

// download downloads a file from AWS, used for testing
func download(writer io.WriterAt, file models.File) (int64, error) {
	downloader := manager.NewDownloader(createClient())
	size, err := downloader.Download(context.Background(), writer, &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// Stream downloads a file from AWS sequentially, used for saving to a Zip file
func Stream(writer io.Writer, file models.File) error {
	svc := createClient()

	obj, err := svc.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})
	if err != nil {
		return err
	}
	defer obj.Body.Close()

	if file.Encryption.IsEncrypted {
		return encryption.DecryptReader(file.Encryption, obj.Body, writer)
	}
	_, err = io.Copy(writer, obj.Body)
	return err
}

// ServeFile either redirects the user to a pre-signed download url (default) or downloads the file and serves it as a proxy (depending
// on configuration). Returns true if blocking operation (to set download status) or false if non-blocking.
func ServeFile(w http.ResponseWriter, r *http.Request, file models.File, forceDownload, forceDecryption bool) (bool, error) {
	if forceDecryption {
		return true, serveDecryptedFile(w, file)
	}
	if awsConfig.ProxyDownload {
		return true, proxyDownload(w, file, forceDownload)
	}
	return false, redirectToDownload(w, r, file, forceDownload)
}

func getPresignedUrl(file models.File, forceDownload bool) (string, error) {
	svc := createClient()
	presignClient := s3.NewPresignClient(svc)

	dispositionType := "inline"
	if forceDownload {
		dispositionType = "Attachment"
	}
	// Use RFC 6266 format to support UTF-8 filenames and avoid encoding errors.
	contentDisposition := fmt.Sprintf("%s; filename*=UTF-8''%s", dispositionType, url.PathEscape(file.Name))

	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket:                     aws.String(file.AwsBucket),
		Key:                        aws.String(file.SHA1),
		ResponseContentDisposition: aws.String(contentDisposition),
		ResponseCacheControl:       aws.String("no-store"),
		ResponseContentType:        aws.String(file.ContentType),
	}, s3.WithPresignExpires(15*time.Second))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// redirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func redirectToDownload(w http.ResponseWriter, r *http.Request, file models.File, forceDownload bool) error {
	url, err := getPresignedUrl(file, forceDownload)
	if err != nil {
		return err
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// proxyDownload streams the file from S3 as a proxy, by downloading a presigned url
func proxyDownload(w http.ResponseWriter, file models.File, forceDownload bool) error {
	url, err := getPresignedUrl(file, forceDownload)
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	headers.Write(file, w, forceDownload, false)
	_, _ = io.Copy(w, resp.Body)
	return nil
}

func serveDecryptedFile(w http.ResponseWriter, file models.File) error {
	svc := createClient()

	// 1. Get the object from S3
	obj, err := svc.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA1),
	})
	if err != nil {
		return err
	}
	defer obj.Body.Close()

	headers.Write(file, w, true, true)
	if file.Encryption.IsEncrypted {
		if !encryption.IsDecryptionAvailable() {
			return errors.New("file is encrypted but server-side decryption key is not available")
		}
		return encryption.DecryptReader(file.Encryption, obj.Body, w)
	}
	_, err = io.Copy(w, obj.Body)
	return err
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
	return fileExists(file.AwsBucket, file.SHA1)
}

func fileExists(bucket, filename string) (bool, int64, error) {
	svc := createClient()

	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()

	info, err := svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, 0, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return false, 0, errors.New("Timeout - could not connect to " + awsConfig.Endpoint)
		}
		return false, 0, err
	}
	return true, *info.ContentLength, nil
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	svc := createClient()

	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()

	_, err := svc.DeleteObject(ctx, &s3.DeleteObjectInput{
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
	svc := createClient()
	input := &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	}

	ctx, cancelCtx := getTimeoutContext()
	defer cancelCtx()

	result, err := svc.GetBucketCors(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchCORSConfiguration" ||
			apiErr.ErrorCode() == "NoSuchCorsConfiguration") {
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

// GetDefaultBucketName returns the default bucketname where new files are stored
func GetDefaultBucketName() string {
	return awsConfig.Bucket
}
