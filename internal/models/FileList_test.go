//go:build test
// +build test

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
	}
	test.IsEqualString(t, file.ToJsonResult("serverurl/"), `{"Result":"OK","FileInfo":{"Id":"testId","Name":"testName","Size":"10 B","SHA256":"sha256","ExpireAt":50,"ExpireAtString":"future","DownloadsRemaining":1,"PasswordHash":"pwhash","HotlinkId":"hotlinkid","ContentType":"text/html","AwsBucket":"","UnlimitedDownloads":false,"UnlimitedTime":false},"Url":"serverurl/d?id=","HotlinkUrl":"serverurl/hotlink/"}`)
}
