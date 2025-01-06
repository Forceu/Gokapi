package models

import (
	"errors"
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestToJsonResult(t *testing.T) {
	file := File{
		Id:                 "testId",
		Name:               "testName",
		Size:               "10 B",
		SizeBytes:          10,
		SHA1:               "sha256",
		ExpireAt:           50,
		ExpireAtString:     "future",
		DownloadsRemaining: 1,
		PasswordHash:       "pwhash",
		HotlinkId:          "hotlinkid",
		ContentType:        "text/html",
		AwsBucket:          "test",
		UserId:             2,
		DownloadCount:      3,
		Encryption: EncryptionInfo{
			IsEncrypted:   true,
			DecryptionKey: []byte{0x01},
			Nonce:         []byte{0x02},
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	test.IsEqualString(t, file.ToJsonResult("serverurl/", false), `{"Result":"OK","FileInfo":{"Id":"testId","Name":"testName","Size":"10 B","HotlinkId":"hotlinkid","ContentType":"text/html","ExpireAtString":"future","UrlDownload":"serverurl/d?id=testId","UrlHotlink":"","ExpireAt":50,"SizeBytes":10,"DownloadsRemaining":1,"DownloadCount":3,"UnlimitedDownloads":true,"UnlimitedTime":true,"RequiresClientSideDecryption":true,"IsEncrypted":true,"IsEndToEndEncrypted":false,"IsPasswordProtected":true,"IsSavedOnLocalStorage":false,"UploaderId":2},"IncludeFilename":false}`)
	test.IsEqualString(t, file.ToJsonResult("serverurl/", true), `{"Result":"OK","FileInfo":{"Id":"testId","Name":"testName","Size":"10 B","HotlinkId":"hotlinkid","ContentType":"text/html","ExpireAtString":"future","UrlDownload":"serverurl/d/testId/testName","UrlHotlink":"","ExpireAt":50,"SizeBytes":10,"DownloadsRemaining":1,"DownloadCount":3,"UnlimitedDownloads":true,"UnlimitedTime":true,"RequiresClientSideDecryption":true,"IsEncrypted":true,"IsEndToEndEncrypted":false,"IsPasswordProtected":true,"IsSavedOnLocalStorage":false,"UploaderId":2},"IncludeFilename":true}`)
}

func TestIsLocalStorage(t *testing.T) {
	file := File{AwsBucket: "123"}
	test.IsEqualBool(t, file.IsLocalStorage(), false)
	file.AwsBucket = ""
	test.IsEqualBool(t, file.IsLocalStorage(), true)
}

func TestErrorAsJson(t *testing.T) {
	result := errorAsJson(errors.New("testerror"))
	test.IsEqualString(t, result, "{\"Result\":\"error\",\"ErrorMessage\":\"testerror\"}")
}

func TestRequiresClientDecryption(t *testing.T) {
	file := File{
		Id:        "test",
		AwsBucket: "bucket",
		Encryption: EncryptionInfo{
			IsEncrypted: true,
		},
	}
	test.IsEqualBool(t, file.RequiresClientDecryption(), true)
	file.Encryption.IsEncrypted = false
	test.IsEqualBool(t, file.RequiresClientDecryption(), false)
	file.AwsBucket = ""
	test.IsEqualBool(t, file.RequiresClientDecryption(), false)
	file.Encryption.IsEncrypted = true
	test.IsEqualBool(t, file.RequiresClientDecryption(), false)
}

func TestGetHolinkUrl(t *testing.T) {
	file := FileApiOutput{
		Id:                           "testfile",
		Name:                         "name",
		Size:                         "1 B",
		HotlinkId:                    "test",
		RequiresClientSideDecryption: true,
	}
	url := getHotlinkUrl(file, "testserver/", false)
	test.IsEqualString(t, url, "")
	file.RequiresClientSideDecryption = false
	url = getHotlinkUrl(file, "testserver/", false)
	test.IsEqualString(t, url, "testserver/hotlink/test")
	file.HotlinkId = ""
	url = getHotlinkUrl(file, "testserver/", false)
	test.IsEqualString(t, url, "testserver/downloadFile?id=testfile")
	url = getHotlinkUrl(file, "testserver/", true)
	test.IsEqualString(t, url, "testserver/dh/testfile/name")
}
