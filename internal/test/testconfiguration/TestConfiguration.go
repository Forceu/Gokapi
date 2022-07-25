//go:build test

package testconfiguration

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"net/http/httptest"
	"os"
	"strings"
	"time"
)

const (
	dataDir    = "test"
	configFile = dataDir + "/config.json"
)

func SetDirEnv() {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_DATA_DIR", "test")
	os.Mkdir(dataDir, 0777)

}

// Create creates a configuration for unit testing
func Create(initFiles bool) {
	SetDirEnv()
	os.WriteFile(configFile, configTestFile, 0777)
	database.Init("./test/filestorage.db")
	writeTestSessions()
	database.SaveUploadDefaults(models.LastUploadValues{
		Downloads:         3,
		TimeExpiry:        20,
		Password:          "123",
		UnlimitedDownload: false,
		UnlimitedTime:     false,
	})
	writeTestFiles()
	database.SaveHotlink(models.File{Id: "n1tSTAGj8zan9KaT4u6p", HotlinkId: "PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg", ExpireAt: time.Now().Add(time.Hour).Unix()})
	writeApiKeyys()
	database.Close()

	if initFiles {
		os.Mkdir("test/data", 0777)
		os.WriteFile("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0", []byte("123"), 0777)
		os.WriteFile("test/data/c4f9375f9834b4e7f0a528cc65c055702bf5f24a", []byte("456"), 0777)
		os.WriteFile("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7", []byte("789"), 0777)
		os.WriteFile("test/data/2341354656543213246465465465432456898794", []byte("abc"), 0777)
		os.WriteFile("test/data/unlimtedtest", []byte("def"), 0777)
		os.WriteFile("test/fileupload.jpg", []byte("abc"), 0777)
	}
}

// WriteEncryptedFile writes metadata for an encrypted file and returns the id
func WriteEncryptedFile() string {
	name := helper.GenerateRandomString(10)
	database.SaveMetaData(models.File{
		Id:   name,
		Name: name,
		SHA1: name,
		Encryption: models.EncryptionInfo{
			IsEncrypted: true,
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	})
	return name
}

// WriteSslCertificates writes a valid or invalid SSL certificate
func WriteSslCertificates(valid bool) {
	os.Mkdir(dataDir, 0777)
	if valid {
		os.WriteFile("test/ssl.crt", sslCertValid, 0700)
		os.WriteFile("test/ssl.key", sslKeyValid, 0700)
	} else {
		os.WriteFile("test/ssl.crt", sslCertExpired, 0700)
		os.WriteFile("test/ssl.key", sslKeyExpired, 0700)
	}
}

// WriteCloudConfigFile writes a valid or invalid AWS config file
func WriteCloudConfigFile(valid bool) {
	os.Mkdir(dataDir, 0777)
	if valid {
		os.WriteFile("test/cloudconfig.yml", cloudConfigTestFile, 0700)
	} else {
		os.WriteFile("test/cloudconfig.yml", []byte("invalid"), 0700)
	}
}

// Delete deletes the configuration for unit testing
func Delete() {
	os.RemoveAll(dataDir)
}

var testServer *httptest.Server

// EnableS3 sets env variables for mock S3
func EnableS3() {
	if !aws.IsMockApi {
		return
	}
	os.Setenv("GOKAPI_AWS_BUCKET", "gokapi-test")
	os.Setenv("GOKAPI_AWS_REGION", "mock-region-1")
	os.Setenv("GOKAPI_AWS_KEY", "accId")
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "accKey")
}

func UseMockS3Server() bool {
	if os.Getenv("REAL_AWS_CREDENTIALS") != "true" {
		fmt.Println("Using MOCK S3 SERVER! To test real credentials, pass REAL_AWS_CREDENTIALS=true")
		fmt.Println("To mock the API, run test with --tags test,awsmock")
		return true
	}
	fmt.Println("Warning, using REAL AWS S3 API! This test will fail if no valid credentials have been provided.")
	fmt.Println("To mock the API, run test with --tags test,awsmock or pass REAL_AWS_CREDENTIALS=false")
	return false
}

func StartS3TestServer() *httptest.Server {
	backend := s3mem.New()
	_ = backend.CreateBucket("gokapi")
	_ = backend.CreateBucket("gokapi-test")
	_, _ = backend.PutObject("gokapi-test", "x341354656543213246465465465432456898794", nil, strings.NewReader("content"), 7)
	faker := gofakes3.New(backend)
	server := httptest.NewServer(faker.Server())
	os.Setenv("GOKAPI_AWS_ENDPOINT", server.URL)
	return server
}

// DisableS3 unsets env variables for mock S3
func DisableS3() {
	aws.LogOut()
	if !aws.IsMockApi {
		return
	}
	os.Unsetenv("GOKAPI_AWS_BUCKET")
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}

func writeTestSessions() {
	database.SaveSession("validsession", models.Session{
		RenewAt:    2147483645,
		ValidUntil: 2147483646,
	}, 1*time.Hour)
	database.SaveSession("logoutsession", models.Session{
		RenewAt:    2147483645,
		ValidUntil: 2147483646,
	}, 1*time.Hour)
	database.SaveSession("needsRenewal", models.Session{
		RenewAt:    0,
		ValidUntil: 2147483646,
	}, 1*time.Hour)
	database.SaveSession("expiredsession", models.Session{
		RenewAt:    0,
		ValidUntil: 0,
	}, 1*time.Hour)
}

func writeApiKeyys() {
	database.SaveApiKey(models.ApiKey{
		Id:           "validkey",
		FriendlyName: "First Key",
	}, false)
	database.SaveApiKey(models.ApiKey{
		Id:             "GAh1IhXDvYnqfYLazWBqMB9HSFmNPO",
		FriendlyName:   "Second Key",
		LastUsed:       1620671580,
		LastUsedString: "used",
	}, false)
	database.SaveApiKey(models.ApiKey{
		Id:           "jiREglQJW0bOqJakfjdVfe8T1EM8n8",
		FriendlyName: "Unnamed Key",
	}, false)
	database.SaveApiKey(models.ApiKey{
		Id:           "okeCMWqhVMZSpt5c1qpCWhKvJJPifb",
		FriendlyName: "Unnamed Key",
	}, false)
}

func writeTestFiles() {
	database.SaveMetaData(models.File{
		Id:                 "Wzol7LyY2QVczXynJtVo",
		Name:               "smallfile2",
		Size:               "8 B",
		SHA1:               "e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
	})
	database.SaveMetaData(models.File{
		Id:                 "e4TjE7CokWK0giiLNxDL",
		Name:               "smallfile2",
		Size:               "8 B",
		SHA1:               "e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
		ExpireAt:           2147483645,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 2,
		ContentType:        "text/html",
	})
	database.SaveMetaData(models.File{
		Id:                 "wefffewhtrhhtrhtrhtr",
		Name:               "smallfile3",
		Size:               "8 B",
		SHA1:               "e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
		ExpireAt:           2147483645,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
	})
	database.SaveMetaData(models.File{
		Id:                 "deletedfile123456789",
		Name:               "DeletedFile",
		Size:               "8 B",
		SHA1:               "invalid",
		ExpireAt:           2147483645,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 2,
		ContentType:        "text/html",
	})
	database.SaveMetaData(models.File{
		Id:                 "jpLXGJKigM4hjtA6T6sN",
		Name:               "smallfile",
		Size:               "7 B",
		SHA1:               "c4f9375f9834b4e7f0a528cc65c055702bf5f24a",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:18",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
		PasswordHash:       "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7",
	})
	database.SaveMetaData(models.File{
		Id:                 "jpLXGJKigM4hjtA6T6sN2",
		Name:               "smallfile",
		Size:               "7 B",
		SHA1:               "c4f9375f9834b4e7f0a528cc65c055702bf5f24a",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:18",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
		PasswordHash:       "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7",
	})
	database.SaveMetaData(models.File{
		Id:                 "n1tSTAGj8zan9KaT4u6p",
		Name:               "picture.jpg",
		Size:               "4 B",
		SHA1:               "a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
		HotlinkId:          "PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
	})
	database.SaveMetaData(models.File{
		Id:                 "cleanuptest123456789",
		Name:               "cleanup",
		Size:               "4 B",
		SHA1:               "2341354656543213246465465465432456898794",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 0,
		ContentType:        "text/html",
	})
	database.SaveMetaData(models.File{
		Id:                 "awsTest1234567890123",
		Name:               "Aws Test File",
		Size:               "20 MB",
		SHA1:               "x341354656543213246465465465432456898794",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 4,
		ContentType:        "application/octet-stream",
		AwsBucket:          "gokapi-test",
	})
	database.SaveMetaData(models.File{
		Id:                 "unlimitedDownload",
		Name:               "unlimitedDownload",
		Size:               "8 B",
		SHA1:               "unlimtedtest",
		ExpireAt:           2147483646,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 0,
		ContentType:        "text/html",
		UnlimitedDownloads: true,
	})
	database.SaveMetaData(models.File{
		Id:                 "unlimitedTime",
		Name:               "unlimitedTime",
		Size:               "8 B",
		SHA1:               "unlimtedtest",
		ExpireAt:           0,
		ExpireAtString:     "2021-05-04 15:19",
		DownloadsRemaining: 1,
		ContentType:        "text/html",
		UnlimitedTime:      true,
	})
}

var configTestFile = []byte(`{
"Authentication": {
    "Method": 0,
    "SaltAdmin": "LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C",
    "SaltFiles": "lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE",
    "Username": "test",
    "Password": "10340aece68aa4fb14507ae45b05506026f276cf",
    "HeaderKey": "",
    "OauthProvider": "",
    "OAuthClientId": "",
    "OAuthClientSecret": "",
    "HeaderUsers": null,
    "OauthUsers": null
  },
   "Port":"127.0.0.1:53843",
  "ServerUrl": "http://127.0.0.1:53843/",
  "RedirectUrl": "https://test.com/",
  "ConfigVersion": 12,
  "LengthId": 20,
  "DataDir": "test/data",
  "MaxMemory": 10,
  "UseSsl": false,
  "MaxFileSizeMB": 25
}`)

var sslCertValid = []byte(`-----BEGIN CERTIFICATE-----
MIIBVzCB/aADAgECAgEBMAoGCCqGSM49BAMCMBExDzANBgNVBAoTBkdva2FwaTAe
Fw0yMTA1MTExNzMwMzVaFw0zODAxMTkwMzE0MDVaMBExDzANBgNVBAoTBkdva2Fw
aTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABPVFhEGE9MomZ8jLt011yvDnWx8k
i2jPNG/FzDjXpfgY/PhDWzR+HS3uoMSsAPnxlg/Xqz681ifvO2Ke8tsjZUujRjBE
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMA8GA1UdEQQIMAaHBH8AAAEwCgYIKoZIzj0EAwIDSQAwRgIhAPOAn+51jcMH
tKO1wjI6vA0avJIuZNUh7wxq0y6K22mzAiEAisbOg45sRuD2V3ffsGfY6d3XyQvC
2A69IsVJJwxqr+g=
-----END CERTIFICATE-----`)

var sslKeyValid = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINFOm9o9K15wzt2fHcnujDPPNERk02zYiMGfYChhaS8zoAoGCCqGSM49
AwEHoUQDQgAE9UWEQYT0yiZnyMu3TXXK8OdbHySLaM80b8XMONel+Bj8+ENbNH4d
Le6gxKwA+fGWD9erPrzWJ+87Yp7y2yNlSw==
-----END EC PRIVATE KEY-----`)

var sslCertExpired = []byte(`-----BEGIN CERTIFICATE-----
MIIBVjCB/aADAgECAgEBMAoGCCqGSM49BAMCMBExDzANBgNVBAoTBkdva2FwaTAe
Fw0yMTA1MTExNzU1MDVaFw0yMTA1MTExNzU1MDZaMBExDzANBgNVBAoTBkdva2Fw
aTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABF+tcmF6JjtKhltXTWo9mlLCLJ+4
C2cAi8ahZS9tIaz/QHC1/Gl3i4Nx8QwubYVw9TScAPMUZTZr7TYJ6Gc3vuWjRjBE
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMA8GA1UdEQQIMAaHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIhAI0ZfsFfr/K/
lcHL0rWkwwlCKIe16v74VFob0pzREW1JAiA0hTFSlv12254Lqf5hEUWPXDeQsq+o
tTe2z6xKh0dwkQ==
-----END CERTIFICATE-----`)

var sslKeyExpired = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIG4kCb5tqz0HyRMBY+HItWtZuT2RmH9w1vsyO2XJcHlLoAoGCCqGSM49
AwEHoUQDQgAEX61yYXomO0qGW1dNaj2aUsIsn7gLZwCLxqFlL20hrP9AcLX8aXeL
g3HxDC5thXD1NJwA8xRlNmvtNgnoZze+5Q==
-----END EC PRIVATE KEY-----`)

var cloudConfigTestFile = []byte(`
##
## Example AWS S3 config. Modify this file and save it to config/cloudconfig.yml
##
aws:
  Bucket: "gokapi"
  Region: "test-region"
  Endpoint: "test-endpoint"
  KeyId: "test-keyid"
  KeySecret: "test-secret"
`)
