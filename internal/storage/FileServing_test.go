package storage

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/cloudconfig"
	"Gokapi/internal/models"
	"Gokapi/internal/storage/cloudstorage/aws"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	var testserver *httptest.Server
	if testconfiguration.UseMockS3Server() {
		testserver = testconfiguration.StartS3TestServer()
	}
	exitVal := m.Run()
	testconfiguration.Delete()
	if testserver != nil {
		testserver.Close()
	}
	os.Exit(exitVal)
}

var idNewFile string

func TestGetFile(t *testing.T) {
	_, result := GetFile("invalid")
	test.IsEqualBool(t, result, false)
	file, result := GetFile("Wzol7LyY2QVczXynJtVo")
	test.IsEqualBool(t, result, true)
	test.IsEqualString(t, file.Id, "Wzol7LyY2QVczXynJtVo")
	test.IsEqualString(t, file.Name, "smallfile2")
	test.IsEqualString(t, file.Size, "8 B")
	test.IsEqualInt(t, file.DownloadsRemaining, 1)
	_, result = GetFile("deletedfile1234")
	test.IsEqualBool(t, result, false)

}

func TestGetFileByHotlink(t *testing.T) {
	_, result := GetFileByHotlink("invalid")
	test.IsEqualBool(t, result, false)
	_, result = GetFileByHotlink("")
	test.IsEqualBool(t, result, false)
	file, result := GetFileByHotlink("PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg")
	test.IsEqualBool(t, result, true)
	test.IsEqualString(t, file.Id, "n1tSTAGj8zan9KaT4u6p")
	test.IsEqualString(t, file.Name, "picture.jpg")
	test.IsEqualString(t, file.Size, "4 B")
	test.IsEqualInt(t, file.DownloadsRemaining, 1)
}

func TestAddHotlink(t *testing.T) {
	file := models.File{Name: "test.dat", Id: "testIdE"}
	addHotlink(&file)
	test.IsEqualString(t, file.HotlinkId, "")
	file = models.File{Name: "test.jpg", Id: "testId"}
	addHotlink(&file)
	test.IsEqualInt(t, len(file.HotlinkId), 44)
	lastCharacters := file.HotlinkId[len(file.HotlinkId)-4:]
	test.IsEqualBool(t, lastCharacters == ".jpg", true)
	settings := configuration.GetServerSettings()
	test.IsEqualString(t, settings.Hotlinks[file.HotlinkId].FileId, "testId")
	test.IsEqualString(t, settings.Hotlinks[file.HotlinkId].Id, file.HotlinkId)
	configuration.Release()
}

func TestNewFile(t *testing.T) {
	os.Setenv("TZ", "UTC")
	content := []byte("This is a file for testing purposes")
	mimeHeader := make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Disposition", "form-data; name=\"file\"; filename=\"test.dat\"")
	mimeHeader.Set("Content-Type", "text/plain")
	header := multipart.FileHeader{
		Filename: "test.dat",
		Header:   mimeHeader,
		Size:     int64(len(content)),
	}
	request := models.UploadRequest{
		AllowedDownloads: 1,
		Expiry:           999,
		ExpiryTimestamp:  2147483600,
		MaxMemory:        10,
		DataDir:          "test/data",
	}
	file, err := NewFile(bytes.NewReader(content), &header, request)
	test.IsNil(t, err)
	test.IsEqualString(t, file.Name, "test.dat")
	test.IsEqualString(t, file.SHA256, "f1474c19eff0fc8998fa6e1b1f7bf31793b103a6")
	test.IsEqualString(t, file.HotlinkId, "")
	test.IsEqualString(t, file.PasswordHash, "")
	test.IsEqualString(t, file.Size, "35 B")
	test.IsEqualString(t, file.ExpireAtString, "2038-01-19 03:13")
	test.IsEqualInt(t, file.DownloadsRemaining, 1)
	test.IsEqualInt(t, len(file.Id), 20)
	test.IsEqualInt(t, int(file.ExpireAt), 2147483600)
	idNewFile = file.Id

	createBigFile("bigfile", 20)
	bigFile, _ := os.Open("bigfile")
	mimeHeader = make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Disposition", "form-data; name=\"file\"; filename=\"bigfile\"")
	mimeHeader.Set("Content-Type", "application/binary")
	header = multipart.FileHeader{
		Filename: "bigfile",
		Header:   mimeHeader,
		Size:     int64(20) * 1024 * 1024,
	}
	request = models.UploadRequest{
		AllowedDownloads: 1,
		Expiry:           999,
		ExpiryTimestamp:  2147483600,
		MaxMemory:        10,
		DataDir:          "test/data",
	}
	// Also testing renaming of temp file
	file, err = NewFile(bigFile, &header, request)
	test.IsNil(t, err)
	test.IsEqualString(t, file.Name, "bigfile")
	test.IsEqualString(t, file.SHA256, "9674344c90c2f0646f0b78026e127c9b86e3ad77")
	test.IsEqualString(t, file.Size, "20.0 MB")
	_, err = bigFile.Seek(0, io.SeekStart)
	test.IsNil(t, err)
	// Testing removal of temp file
	test.IsEqualString(t, file.Name, "bigfile")
	test.IsEqualString(t, file.SHA256, "9674344c90c2f0646f0b78026e127c9b86e3ad77")
	test.IsEqualString(t, file.Size, "20.0 MB")
	bigFile.Close()
	os.Remove("bigfile")

	if aws.IsIncludedInBuild {
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		file, err = NewFile(bytes.NewReader(content), &header, request)
		test.IsNil(t, err)
		test.IsEqualString(t, file.Name, "bigfile")
		test.IsEqualString(t, file.SHA256, "f1474c19eff0fc8998fa6e1b1f7bf31793b103a6")
		test.IsEqualString(t, file.Size, "20.0 MB")
		testconfiguration.DisableS3()
	}
}

func TestServeFile(t *testing.T) {
	file, result := GetFile(idNewFile)
	test.IsEqualBool(t, result, true)
	r := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	ServeFile(file, w, r, true)
	_, result = GetFile(idNewFile)
	test.IsEqualBool(t, result, false)

	test.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "attachment; filename=\"test.dat\"")
	test.IsEqualString(t, w.Result().Header.Get("Content-Length"), "35")
	test.IsEqualString(t, w.Result().Header.Get("Content-Type"), "text/plain")
	content, err := ioutil.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(content), "This is a file for testing purposes")

	if aws.IsIncludedInBuild {
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		r = httptest.NewRequest("GET", "/upload", nil)
		w = httptest.NewRecorder()
		file, result = GetFile("awsTest1234567890123")
		test.IsEqualBool(t, result, true)
		ServeFile(file, w, r, false)
		if aws.IsMockApi {
			test.ResponseBodyContains(t, w, "https://redirect.url")
		} else {
			test.ResponseBodyContains(t, w, "<a href=\"http")
		}
		testconfiguration.DisableS3()
	}
}

func TestCleanUp(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.Files["cleanuptest123456789"].Name, "cleanup")
	test.IsEqualString(t, settings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	test.IsEqualString(t, settings.Files["deletedfile123456789"].Name, "DeletedFile")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")

	CleanUp(false)
	test.IsEqualString(t, settings.Files["cleanuptest123456789"].Name, "cleanup")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")
	test.IsEqualString(t, settings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, settings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")

	file, _ := GetFile("n1tSTAGj8zan9KaT4u6p")
	file.DownloadsRemaining = 0
	settings.Files["n1tSTAGj8zan9KaT4u6p"] = file

	CleanUp(false)
	test.FileDoesNotExist(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, settings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, settings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("Wzol7LyY2QVczXynJtVo")
	file.DownloadsRemaining = 0
	settings.Files["Wzol7LyY2QVczXynJtVo"] = file

	CleanUp(false)
	test.FileExists(t, "test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7")
	test.IsEqualString(t, settings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, settings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, settings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, settings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("e4TjE7CokWK0giiLNxDL")
	file.DownloadsRemaining = 0
	settings.Files["e4TjE7CokWK0giiLNxDL"] = file
	file, _ = GetFile("wefffewhtrhhtrhtrhtr")
	file.DownloadsRemaining = 0
	settings.Files["wefffewhtrhhtrhtrhtr"] = file

	CleanUp(false)
	test.FileDoesNotExist(t, "test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7")
	test.IsEqualString(t, settings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, settings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, settings.Files["e4TjE7CokWK0giiLNxDL"].Name, "")
	test.IsEqualString(t, settings.Files["wefffewhtrhhtrhtrhtr"].Name, "")

	test.IsEqualString(t, settings.Files["cleanuptest123456789"].Name, "cleanup")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")
	settings.DownloadStatus = make(map[string]models.DownloadStatus)
	CleanUp(false)
	test.IsEqualString(t, settings.Files["cleanuptest123456789"].Name, "")
	test.FileDoesNotExist(t, "test/data/2341354656543213246465465465432456898794")

	if aws.IsIncludedInBuild {
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, settings.Files["awsTest1234567890123"].Name, "Aws Test File")
		testconfiguration.DisableS3()
	}
}

func TestDeleteFile(t *testing.T) {
	testconfiguration.Create(true)
	configuration.Load()
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	test.FileExists(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	result := DeleteFile("n1tSTAGj8zan9KaT4u6p")
	test.IsEqualBool(t, result, true)
	test.IsEqualString(t, settings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.FileDoesNotExist(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	result = DeleteFile("invalid")
	test.IsEqualBool(t, result, false)
	result = DeleteFile("")
	test.IsEqualBool(t, result, false)

	if aws.IsIncludedInBuild {
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		awsFile := models.File{
			Id:        "awsTest1234567890123",
			Name:      "aws Test File",
			Size:      "20 MB",
			SHA256:    "x341354656543213246465465465432456898794",
			AwsBucket: "gokapi-test",
		}
		settings.Files["awsTest1234567890123"] = awsFile
		result, err := aws.FileExists(settings.Files["awsTest1234567890123"])
		test.IsEqualBool(t, result, true)
		test.IsNil(t, err)
		DeleteFile("awsTest1234567890123")
		result, err = aws.FileExists(awsFile)
		test.IsEqualBool(t, result, false)
		test.IsNil(t, err)
		testconfiguration.DisableS3()
	}
}

func createBigFile(name string, megabytes int64) {
	size := megabytes * 1024 * 1024
	file, _ := os.Create(name)
	_, _ = file.Seek(size-1, 0)
	_, _ = file.Write([]byte{0})
	_ = file.Close()
}
