package storage

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	testconfiguration "Gokapi/internal/test"
	testconfiguration2 "Gokapi/internal/test/testconfiguration"
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration2.Create(true)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration2.Delete()
	os.Exit(exitVal)
}

var idNewFile string

func TestGetFile(t *testing.T) {
	_, result := GetFile("invalid")
	testconfiguration.IsEqualBool(t, result, false)
	file, result := GetFile("Wzol7LyY2QVczXynJtVo")
	testconfiguration.IsEqualBool(t, result, true)
	testconfiguration.IsEqualString(t, file.Id, "Wzol7LyY2QVczXynJtVo")
	testconfiguration.IsEqualString(t, file.Name, "smallfile2")
	testconfiguration.IsEqualString(t, file.Size, "8 B")
	testconfiguration.IsEqualInt(t, file.DownloadsRemaining, 1)
	_, result = GetFile("deletedfile1234")
	testconfiguration.IsEqualBool(t, result, false)

}

func TestGetFileByHotlink(t *testing.T) {
	_, result := GetFileByHotlink("invalid")
	testconfiguration.IsEqualBool(t, result, false)
	_, result = GetFileByHotlink("")
	testconfiguration.IsEqualBool(t, result, false)
	file, result := GetFileByHotlink("PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg")
	testconfiguration.IsEqualBool(t, result, true)
	testconfiguration.IsEqualString(t, file.Id, "n1tSTAGj8zan9KaT4u6p")
	testconfiguration.IsEqualString(t, file.Name, "picture.jpg")
	testconfiguration.IsEqualString(t, file.Size, "4 B")
	testconfiguration.IsEqualInt(t, file.DownloadsRemaining, 1)
}

func TestAddHotlink(t *testing.T) {
	file := models.File{Name: "test.dat", Id: "testIdE"}
	addHotlink(&file)
	testconfiguration.IsEqualString(t, file.HotlinkId, "")
	file = models.File{Name: "test.jpg", Id: "testId"}
	addHotlink(&file)
	testconfiguration.IsEqualInt(t, len(file.HotlinkId), 44)
	lastCharacters := file.HotlinkId[len(file.HotlinkId)-4:]
	testconfiguration.IsEqualBool(t, lastCharacters == ".jpg", true)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Hotlinks[file.HotlinkId].FileId, "testId")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Hotlinks[file.HotlinkId].Id, file.HotlinkId)
}

func TestNewFile(t *testing.T) {
	os.Setenv("TZ", "UTC")
	content := []byte("This is a file for testing purposes")
	mimeHeader := make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Disposition", "form-data; name=\"file\"; filename=\"test.dat\"")
	mimeHeader.Set("Content-Type", "text")
	header := multipart.FileHeader{
		Filename: "test.dat",
		Header:   mimeHeader,
		Size:     int64(len(content)),
	}
	file, err := NewFile(bytes.NewReader(content), &header, 2147483600, 1, "")
	testconfiguration.IsNil(t, err)
	testconfiguration.IsEqualString(t, file.Name, "test.dat")
	testconfiguration.IsEqualString(t, file.SHA256, "f1474c19eff0fc8998fa6e1b1f7bf31793b103a6")
	testconfiguration.IsEqualString(t, file.HotlinkId, "")
	testconfiguration.IsEqualString(t, file.PasswordHash, "")
	testconfiguration.IsEqualString(t, file.Size, "35 B")
	testconfiguration.IsEqualString(t, file.ExpireAtString, "2038-01-19 03:13")
	testconfiguration.IsEqualInt(t, file.DownloadsRemaining, 1)
	testconfiguration.IsEqualInt(t, len(file.Id), 20)
	testconfiguration.IsEqualInt(t, int(file.ExpireAt), 2147483600)
	idNewFile = file.Id
}

func TestServeFile(t *testing.T) {
	file, result := GetFile(idNewFile)
	testconfiguration.IsEqualBool(t, result, true)
	r := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	ServeFile(file, w, r, true)
	_, result = GetFile(idNewFile)
	testconfiguration.IsEqualBool(t, result, false)

	testconfiguration.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "attachment; filename=\"test.dat\"")
	testconfiguration.IsEqualString(t, w.Result().Header.Get("Content-Length"), "35")
	testconfiguration.IsEqualString(t, w.Result().Header.Get("Content-Type"), "text")
	content, err := ioutil.ReadAll(w.Result().Body)
	testconfiguration.IsNil(t, err)
	testconfiguration.IsEqualString(t, string(content), "This is a file for testing purposes")
}

func TestCleanUp(t *testing.T) {
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "DeletedFile")
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)

	CleanUp(false)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")

	file, _ := GetFile("n1tSTAGj8zan9KaT4u6p")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"] = file

	CleanUp(false)
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0"), false)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("Wzol7LyY2QVczXynJtVo")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"] = file

	CleanUp(false)
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7"), true)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("e4TjE7CokWK0giiLNxDL")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"] = file
	file, _ = GetFile("wefffewhtrhhtrhtrhtr")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"] = file

	CleanUp(false)
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7"), false)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "")
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "")

	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)
	configuration.ServerSettings.DownloadStatus = make(map[string]models.DownloadStatus)
	CleanUp(false)
	testconfiguration.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "")
	testconfiguration.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), false)
}
