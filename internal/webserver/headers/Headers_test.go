package headers

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestWriteDownloadHeaders(t *testing.T) {
	file := models.File{Name: "testname", ContentType: "testtype"}
	w, _ := test.GetRecorder("GET", "/test", nil, nil, nil)
	Write(file, w, true)
	test.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "attachment; filename=\"testname\"")
	w, _ = test.GetRecorder("GET", "/test", nil, nil, nil)
	Write(file, w, false)
	test.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "inline; filename=\"testname\"")
	test.IsEqualString(t, w.Result().Header.Get("Content-Type"), "testtype")
	file.Encryption.IsEncrypted = true
	w, _ = test.GetRecorder("GET", "/test", nil, nil, nil)
	Write(file, w, false)
	test.IsEqualString(t, w.Result().Header.Get("Accept-Ranges"), "bytes")
}
