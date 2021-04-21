package filestructure

import (
	"Gokapi/pkg/test"
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
		ContentType:        "test/html",
	}
	test.IsEqualString(t, file.ToJsonResult("serverurl/"), `{"Result":"OK","FileInfo":{"Id":"testId","Name":"testName","Size":"10 B","SHA256":"sha256","ExpireAt":50,"ExpireAtString":"future","DownloadsRemaining":1,"PasswordHash":"pwhash","HotlinkId":"hotlinkid","ContentType":"test/html"},"Url":"serverurl/d?id=","HotlinkUrl":"serverurl/hotlink/"}`)
}
