//  +build test

package testconfiguration

import (
	"Gokapi/internal/models"
	"Gokapi/internal/storage/aws"
	"os"
)

const (
	dataDir    = "test"
	configFile = dataDir + "/config.json"
)

// Create creates a configuration for unit testing
func Create(initFiles bool) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_DATA_DIR", "test")
	os.Mkdir(dataDir, 0777)
	os.WriteFile(configFile, configTestFile, 0777)
	if initFiles {
		os.Mkdir("test/data", 0777)
		os.WriteFile("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0", []byte("123"), 0777)
		os.WriteFile("test/data/c4f9375f9834b4e7f0a528cc65c055702bf5f24a", []byte("456"), 0777)
		os.WriteFile("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7", []byte("789"), 0777)
		os.WriteFile("test/data/2341354656543213246465465465432456898794", []byte("abc"), 0777)
		os.WriteFile("test/fileupload.jpg", []byte("abc"), 0777)
	}
}

// WriteUpgradeConfigFile writes a Gokapi v1.1.0 config file
func WriteUpgradeConfigFile() {
	os.Mkdir(dataDir, 0777)
	os.WriteFile(configFile, configUpgradeTestFile, 0777)
}

// Delete deletes the configuration for unit testing
func Delete() {
	os.RemoveAll(dataDir)
}

// EnableS3 sets env variables for mock S3
func EnableS3() {
	os.Setenv("GOKAPI_AWS_BUCKET", "gokapi-test")
	os.Setenv("AWS_REGION", "mock-region-1")
	os.Setenv("AWS_ACCESS_KEY_ID", "accId")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "accKey")
	aws.Upload(nil, models.File{
		Id:        "awsTest1234567890123",
		Name:      "aws Test File",
		Size:      "20 MB",
		SHA256:    "x341354656543213246465465465432456898794",
		AwsBucket: "gokapi-test",
	})
}

// DisableS3 unsets env variables for mock S3
func DisableS3() {
	os.Unsetenv("GOKAPI_AWS_BUCKET")
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}

// StartMockInputStdin simulates a user input on stdin. Call StopMockInputStdin afterwards!
func StartMockInputStdin(input string) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	_, err = w.Write([]byte(input))
	if err != nil {
		panic(err)
	}
	w.Close()

	stdin := os.Stdin
	os.Stdin = r
	return stdin
}

// StopMockInputStdin needs to be called after StartMockInputStdin
func StopMockInputStdin(stdin *os.File) {
	os.Stdin = stdin
}

var configTestFile = []byte(`{
   "Port":"127.0.0.1:53843",
   "AdminName":"test",
   "AdminPassword":"10340aece68aa4fb14507ae45b05506026f276cf",
   "ServerUrl":"http://127.0.0.1:53843/",
   "DefaultDownloads":3,
   "DefaultExpiry":20,
   "DefaultPassword":"123",
   "RedirectUrl":"https://test.com/",
   "Sessions":{
      "validsession":{
         "RenewAt":2147483645,
         "ValidUntil":2147483646
      },
      "logoutsession":{
         "RenewAt":2147483645,
         "ValidUntil":2147483646
      },
      "needsRenewal":{
         "RenewAt":0,
         "ValidUntil":2147483646
      },
      "expiredsession":{
         "RenewAt":0,
         "ValidUntil":0
      }
   },
   "Files":{
      "Wzol7LyY2QVczXynJtVo":{
         "Id":"Wzol7LyY2QVczXynJtVo",
         "Name":"smallfile2",
         "Size":"8 B",
         "SHA256":"e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":1,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":""
      },
      "e4TjE7CokWK0giiLNxDL":{
         "Id":"e4TjE7CokWK0giiLNxDL",
         "Name":"smallfile2",
         "Size":"8 B",
         "SHA256":"e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
         "ExpireAt":2147483645,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":2,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":""
      },
      "wefffewhtrhhtrhtrhtr":{
         "Id":"wefffewhtrhhtrhtrhtr",
         "Name":"smallfile3",
         "Size":"8 B",
         "SHA256":"e017693e4a04a59d0b0f400fe98177fe7ee13cf7",
         "ExpireAt":2147483645,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":1,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":""
      },
      "deletedfile123456789":{
         "Id":"deletedfile123456789",
         "Name":"DeletedFile",
         "Size":"8 B",
         "SHA256":"invalid",
         "ExpireAt":2147483645,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":2,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":""
      },
      "jpLXGJKigM4hjtA6T6sN":{
         "Id":"jpLXGJKigM4hjtA6T6sN",
         "Name":"smallfile",
         "Size":"7 B",
         "SHA256":"c4f9375f9834b4e7f0a528cc65c055702bf5f24a",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:18",
         "DownloadsRemaining":1,
         "ContentType":"text/html",
         "PasswordHash":"7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7",
         "HotlinkId":""
      },
      "jpLXGJKigM4hjtA6T6sN2":{
         "Id":"jpLXGJKigM4hjtA6T6sN2",
         "Name":"smallfile",
         "Size":"7 B",
         "SHA256":"c4f9375f9834b4e7f0a528cc65c055702bf5f24a",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:18",
         "DownloadsRemaining":1,
         "ContentType":"text/html",
         "PasswordHash":"7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7",
         "HotlinkId":""
      },
      "n1tSTAGj8zan9KaT4u6p":{
         "Id":"n1tSTAGj8zan9KaT4u6p",
         "Name":"picture.jpg",
         "Size":"4 B",
         "SHA256":"a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":1,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":"PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg"
      },
      "cleanuptest123456789":{
         "Id":"cleanuptest123456789",
         "Name":"cleanup",
         "Size":"4 B",
         "SHA256":"2341354656543213246465465465432456898794",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":0,
         "PasswordHash":"",
         "ContentType":"text/html",
         "HotlinkId":""
      },
      "awsTest1234567890123":{
         "Id":"awsTest1234567890123",
         "Name":"Aws Test File",
         "Size":"20 MB",
         "SHA256":"x341354656543213246465465465432456898794",
         "ExpireAt":2147483646,
         "ExpireAtString":"2021-05-04 15:19",
         "DownloadsRemaining":4,
         "PasswordHash":"",
         "ContentType":"application/octet-stream",
         "AwsBucket":"gokapi-test",
         "HotlinkId":""
      }
   },
   "Hotlinks":{
      "PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg":{
         "Id":"PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
         "FileId":"n1tSTAGj8zan9KaT4u6p"
      }
   },
   "DownloadStatus":{
      "69JCbLVxx2KxfvB6FYkrDn3oCU7BWT":{
         "Id":"69JCbLVxx2KxfvB6FYkrDn3oCU7BWT",
         "FileId":"cleanuptest123456789",
         "ExpireAt":2147483646
      }
   },
   "ApiKeys":{
      "validkey":{
         "Id":"validkey",
         "FriendlyName":"Unnamed Key",
         "LastUsed":0,
         "LastUsedString":""
      },
      "GAh1IhXDvYnqfYLazWBqMB9HSFmNPO":{
         "Id":"GAh1IhXDvYnqfYLazWBqMB9HSFmNPO",
         "FriendlyName":"Unnamed Key",
         "LastUsed":0,
         "LastUsedString":""
      },
      "jiREglQJW0bOqJakfjdVfe8T1EM8n8":{
         "Id":"jiREglQJW0bOqJakfjdVfe8T1EM8n8",
         "FriendlyName":"Unnamed Key",
         "LastUsed":0,
         "LastUsedString":""
      },
      "okeCMWqhVMZSpt5c1qpCWhKvJJPifb":{
         "Id":"okeCMWqhVMZSpt5c1qpCWhKvJJPifb",
         "FriendlyName":"Unnamed Key",
         "LastUsed":0,
         "LastUsedString":""
      }
   },
   "ConfigVersion":6,
   "SaltAdmin":"LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C",
   "SaltFiles":"lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE",
   "LengthId":20,
   "DataDir":"test/data"
}`)

var configUpgradeTestFile = []byte(`{
   "Port":"127.0.0.1:53842",
   "AdminName":"admin",
   "AdminPassword":"7450c2403ab85f0e8d5436818b66b99fdd287ac6",
   "ServerUrl":"https://gokapi.url/",
   "DefaultDownloads":1,
   "DefaultExpiry":14,
   "DefaultPassword":"123",
   "RedirectUrl":"https://github.com/Forceu/Gokapi/",
   "Sessions":{
      "y0t-OQGF5UPFHyFOLab38SNjrc_a4xdIHTsZclkLpxuSwwTzS_qEETsinkgVIdWNMnQjhcaZtgCoJdpu":{
         "RenewAt":1619774155,
         "ValidUntil":1622362555
      }
   },
   "Files":{
      "MgXJLe4XLfpXcL12ec4i":{
         "Id":"MgXJLe4XLfpXcL12ec4i",
         "Name":"gokapi-linux_amd64",
         "Size":"10.2 MB",
         "SHA256":"b08f5989e1c6d57b45fffe39a8edc5da715799b7",
         "ExpireAt":1620980170,
         "ExpireAtString":"2021-05-14 10:16",
         "DownloadsRemaining":1,
         "PasswordHash":"e143a1801faba4c5c6fdc2e823127c988940f72e"
      },
      "doLN1pgbb945DfhGottx":{
         "Id":"doLN1pgbb945DfhGottx",
         "Name":"config.json",
         "Size":"945 B",
         "SHA256":"d2d6fd5fbf4a4bb1b1ae2f19130dd75b5adc0a0b",
         "ExpireAt":1620980181,
         "ExpireAtString":"2021-05-14 10:16",
         "DownloadsRemaining":1,
         "PasswordHash":"e143a1801faba4c5c6fdc2e823127c988940f72e"
      },
      "q06tcBco9gdJTf_pZ8xf":{
         "Id":"q06tcBco9gdJTf_pZ8xf",
         "Name":"gokapi-linux_amd64",
         "Size":"10.2 MB",
         "SHA256":"b08f5989e1c6d57b45fffe39a8edc5da715799b7",
         "ExpireAt":1620980160,
         "ExpireAtString":"2021-05-14 10:16",
         "DownloadsRemaining":1,
         "PasswordHash":""
      }
   }
}`)
