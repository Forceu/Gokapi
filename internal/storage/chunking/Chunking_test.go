package chunking

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestParseChunkInfo(t *testing.T) {
	data := url.Values{}
	data.Set("dztotalfilesize", "100000")
	data.Set("dzchunkbyteoffset", "10")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r := test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	info, err := ParseChunkInfo(r)
	test.IsNil(t, err)
	test.IsEqualInt64(t, info.TotalFilesizeBytes, 100000)
	test.IsEqualInt64(t, info.Offset, 10)
	test.IsEqualString(t, info.UUID, "fweflwfejkfwejf-wekjefwjfwej")

	data.Set("dzuuid", "23432")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dzuuid", "!\"§$%&/()=?abc-")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	info, err = ParseChunkInfo(r)
	test.IsNil(t, err)
	test.IsEqualInt64(t, info.TotalFilesizeBytes, 100000)
	test.IsEqualInt64(t, info.Offset, 10)
	test.IsEqualString(t, info.UUID, "___________abc-")

	data.Set("dzchunkbyteoffset", "invalid")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "-1")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "0")
	data.Set("dztotalfilesize", "invalid")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dztotalfilesize", "-1")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	data.Set("dztotalfilesize", "")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)

	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("invalid§%&§"))
	_, err = ParseChunkInfo(r)
	test.IsNotNil(t, err)
}

func TestParseFileHeader(t *testing.T) {
	data := url.Values{}
	data.Set("filename", "testfile")
	data.Set("filecontenttype", "test/content")
	data.Set("filesize", "1000")
	_, r := test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	header, err := ParseFileHeader(r)
	test.IsNil(t, err)
	test.IsEqualString(t, header.Filename, "testfile")
	test.IsEqualString(t, header.ContentType, "test/content")
	test.IsEqualInt64(t, header.Size, 1000)

	data.Set("filename", "")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	data.Set("filename", "testfile")
	data.Set("filecontenttype", "")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	data.Set("filecontenttype", "test/content")
	data.Set("filesize", "invalid")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	data.Set("filesize", "")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	data.Set("filesize", "-5")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)

	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("invalid§%&§"))
	_, err = ParseFileHeader(r)
	test.IsNotNil(t, err)
}

func TestParseMultipartHeader(t *testing.T) {
	mimeHeader := make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Type", "test/type")
	multipartHeader := multipart.FileHeader{
		Filename: "testfile",
		Size:     100,
		Header:   mimeHeader,
	}

	header, err := ParseMultipartHeader(&multipartHeader)
	test.IsNil(t, err)
	test.IsEqualInt64(t, header.Size, 100)
	test.IsEqualString(t, header.Filename, "testfile")
	test.IsEqualString(t, header.ContentType, "test/type")

	multipartHeader.Filename = ""
	_, err = ParseMultipartHeader(&multipartHeader)
	test.IsNotNil(t, err)

	multipartHeader.Filename = "testfile"
	multipartHeader.Header.Del("Content-Type")
	_, err = ParseMultipartHeader(&multipartHeader)
	test.IsNotNil(t, err)
}

func TestGetChunkFilePath(t *testing.T) {
	test.IsEqualString(t, getChunkFilePath("test"), "test/data/chunk-test")
}

func TestGetFileByChunkId(t *testing.T) {
	test.FileDoesNotExist(t, "testchunk")
	_, err := GetFileByChunkId("testchunk")
	test.IsNotNil(t, err)
	err = os.WriteFile("test/data/chunk-testchunk", []byte("conent"), 0777)
	test.IsNil(t, err)
	file, err := GetFileByChunkId("testchunk")
	test.IsEqualString(t, file.Name(), "test/data/chunk-testchunk")
	test.IsNil(t, err)
	err = os.Remove(file.Name())
	test.IsNil(t, err)
}

func TestNewChunk(t *testing.T) {
	info := ChunkInfo{
		TotalFilesizeBytes: 100,
		Offset:             0,
		UUID:               "testuuid12345",
	}
	header := multipart.FileHeader{
		Size: 21,
	}
	err := NewChunk(strings.NewReader("This is a test content"), &header, info)
	test.IsNil(t, err)
	test.IsEqualString(t, sha1sumFile("test/data/chunk-testuuid12345"), "a69ec3c3a031e3540d0c2a864ca931f3d54e2c13")

	info.Offset = 52
	header = multipart.FileHeader{
		Size: 11,
	}
	err = NewChunk(strings.NewReader("More content"), &header, info)
	test.IsNil(t, err)
	test.IsEqualString(t, sha1sumFile("test/data/chunk-testuuid12345"), "8794d8352fae46b83bab83d3e613dde8f0244ded")

	info.Offset = 99
	err = NewChunk(strings.NewReader("More content"), &header, info)
	test.IsNotNil(t, err)

	err = os.Remove("test/data/chunk-testuuid12345")
	test.IsNil(t, err)

	info.TotalFilesizeBytes = -4
	err = NewChunk(strings.NewReader("More content"), &header, info)
	test.IsNotNil(t, err)

	info.TotalFilesizeBytes = 100
	info.UUID = "../../../../../../../../../../invalid"
	err = NewChunk(strings.NewReader("More content"), &header, info)
	test.IsNotNil(t, err)

}

func sha1sumFile(filename string) string {
	sha := sha1.New()
	filecontent, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	sha.Write(filecontent)
	return hex.EncodeToString(sha.Sum(nil))
}
