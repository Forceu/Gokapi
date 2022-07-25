package chunking

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/juju/ratelimit"
	"golang.org/x/sync/errgroup"
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
	info, err := ParseChunkInfo(r, false)
	test.IsNil(t, err)
	test.IsEqualInt64(t, info.TotalFilesizeBytes, 100000)
	test.IsEqualInt64(t, info.Offset, 10)
	test.IsEqualString(t, info.UUID, "fweflwfejkfwejf-wekjefwjfwej")

	data.Set("dzuuid", "23432")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dzuuid", "!\"§$%&/()=?abc-")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	info, err = ParseChunkInfo(r, false)
	test.IsNil(t, err)
	test.IsEqualInt64(t, info.TotalFilesizeBytes, 100000)
	test.IsEqualInt64(t, info.Offset, 10)
	test.IsEqualString(t, info.UUID, "___________abc-")

	data.Set("dzchunkbyteoffset", "invalid")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "-1")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dzchunkbyteoffset", "0")
	data.Set("dztotalfilesize", "invalid")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dztotalfilesize", "-1")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data.Set("dztotalfilesize", "")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader("invalid§%&§"))
	_, err = ParseChunkInfo(r, false)
	test.IsNotNil(t, err)

	data = url.Values{}
	data.Set("filesize", "100000")
	data.Set("offset", "10")
	data.Set("uuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	info, err = ParseChunkInfo(r, true)
	test.IsNil(t, err)
	test.IsEqualInt64(t, info.TotalFilesizeBytes, 100000)
	test.IsEqualInt64(t, info.Offset, 10)
	test.IsEqualString(t, info.UUID, "fweflwfejkfwejf-wekjefwjfwej")
}

func TestParseContentType(t *testing.T) {
	var imageFileExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".tiff", ".tif", ".ico"}

	data := url.Values{}
	data.Set("filename", "test.unknown")
	data.Set("filecontenttype", "test/unknown")
	_, r := test.GetRecorder("POST", "/uploadComplete", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	err := r.ParseForm()
	test.IsNil(t, err)
	contentType := parseContentType(r)
	test.IsEqualString(t, contentType, "test/unknown")

	data.Set("filecontenttype", "")
	_, r = test.GetRecorder("POST", "/uploadComplete", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	err = r.ParseForm()
	test.IsNil(t, err)
	contentType = parseContentType(r)
	test.IsEqualString(t, contentType, "application/octet-stream")

	for _, imageExt := range imageFileExtensions {
		data.Set("filename", "test"+imageExt)
		_, r = test.GetRecorder("POST", "/uploadComplete", nil, []test.Header{
			{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
			strings.NewReader(data.Encode()))
		err = r.ParseForm()
		test.IsNil(t, err)
		contentType = parseContentType(r)
		test.IsNotEqualString(t, contentType, "application/octet-stream")
		test.IsNotEqualString(t, contentType, "")
		test.IsEqualBool(t, strings.Contains(contentType, "image/"), true)
	}
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
	header, err = ParseFileHeader(r)
	test.IsNil(t, err)
	test.IsEqualString(t, header.ContentType, "application/octet-stream")

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

	data = url.Values{}
	data.Set("dztotalfilesize", "100000")
	data.Set("dzchunkbyteoffset", "10")
	data.Set("dzuuid", "fweflwfejkfwejf-wekjefwjfwej")
	_, r = test.GetRecorder("POST", "/uploadChunk", nil, []test.Header{
		{Name: "Content-type", Value: "application/x-www-form-urlencoded"}},
		strings.NewReader(data.Encode()))
	_, err = ParseChunkInfo(r, true)
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

	// Testing simultaneous writes
	egroup := new(errgroup.Group)
	egroup.Go(func() error {
		return writeRateLimitedChunk(true)
	})
	egroup.Go(func() error {
		return writeRateLimitedChunk(false)
	})
	err = egroup.Wait()
	test.IsNil(t, err)
}

func writeRateLimitedChunk(firstHalf bool) error {
	var offset int64
	if !firstHalf {
		offset = 500 * 1024
	}
	info := ChunkInfo{
		TotalFilesizeBytes: 1000 * 1024,
		Offset:             offset,
		UUID:               "multiplewrites",
	}
	header := multipart.FileHeader{
		Size: 500 * 1024,
	}
	content := []byte(helper.GenerateRandomString(500 * 1024))
	bucket := ratelimit.NewBucketWithRate(400*1024, 400*1024)
	err := NewChunk(ratelimit.Reader(bytes.NewReader(content), bucket), &header, info)
	return err
}

func TestWriteChunk(t *testing.T) {
	err := writeChunk(nil, &multipart.FileHeader{Size: 10}, ChunkInfo{
		UUID:               "",
		TotalFilesizeBytes: 10,
	})
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
