package models

import (
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestToJsonResult(t *testing.T) {
	file := File{
		Id:                 "testId",
		Name:               "testName",
		Size:               "10 B",
		SHA256:             "sha256",
		ExpireAt:           50,
		ExpireAtString:     "future",
		DownloadsRemaining: 1,
		PasswordHash:       "pwhash",
		HotlinkId:          "hotlinkid",
		ContentType:        "text/html",
		AwsBucket:          "test",
		DownloadCount:      3,
		Encryption: EncryptionInfo{
			IsEncrypted:   true,
			DecryptionKey: []byte{0x01},
			Nonce:         []byte{0x02},
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	test.IsEqualString(t, file.ToJsonResult("serverurl/"), `{"Result":"OK","FileInfo":{"Id":"testId","Name":"testName","Size":"10 B","HotlinkId":"hotlinkid","ContentType":"text/html","ExpireAt":50,"ExpireAtString":"future","DownloadsRemaining":1,"DownloadCount":3,"UnlimitedDownloads":true,"UnlimitedTime":true,"RequiresClientSideDecryption":false,"IsEncrypted":true,"IsPasswordProtected":true,"IsSavedOnLocalStorage":false},"Url":"serverurl/d?id=","HotlinkUrl":"serverurl/hotlink/","GenericHotlinkUrl":"serverurl/downloadFile?id="}`)
}
