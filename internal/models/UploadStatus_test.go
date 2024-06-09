package models

import (
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestUploadStatus_ToJson(t *testing.T) {
	status := UploadStatus{}
	output, err := status.ToJson()
	test.IsNil(t, err)
	test.IsEqualString(t, string(output), "{\"chunkid\":\"\",\"currentstatus\":0,\"lastupdate\":0,\"type\":\"uploadstatus\"}")
}
