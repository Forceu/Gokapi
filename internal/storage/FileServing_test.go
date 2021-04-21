package storage

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/filestructure"
	testconfiguration "Gokapi/internal/test"
	"Gokapi/pkg/test"
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
	exitVal := m.Run()
	testconfiguration.Delete()
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
	file := filestructure.File{Name: "test.dat", Id: "testIdE"}
	addHotlink(&file)
	test.IsEqualString(t, file.HotlinkId, "")
	file = filestructure.File{Name: "test.jpg", Id: "testId"}
	addHotlink(&file)
	test.IsEqualInt(t, len(file.HotlinkId), 44)
	lastCharacters := file.HotlinkId[len(file.HotlinkId)-4:]
	test.IsEqualBool(t, lastCharacters == ".jpg", true)
	test.IsEqualString(t, configuration.ServerSettings.Hotlinks[file.HotlinkId].FileId, "testId")
	test.IsEqualString(t, configuration.ServerSettings.Hotlinks[file.HotlinkId].Id, file.HotlinkId)
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
	file, err := processUpload(&content, &header, 2147483600, 1, "")
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
	test.IsEqualString(t, w.Result().Header.Get("Content-Type"), "text")
	content, err := ioutil.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(content), "This is a file for testing purposes")

}

func TestCleanUp(t *testing.T) {
	test.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	test.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	test.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "DeletedFile")
	test.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)

	CleanUp(false)
	test.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	test.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)
	test.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")

	file, _ := GetFile("n1tSTAGj8zan9KaT4u6p")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"] = file

	CleanUp(false)
	test.IsEqualBool(t, helper.FileExists("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0"), false)
	test.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("Wzol7LyY2QVczXynJtVo")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"] = file

	CleanUp(false)
	test.IsEqualBool(t, helper.FileExists("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7"), true)
	test.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("e4TjE7CokWK0giiLNxDL")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"] = file
	file, _ = GetFile("wefffewhtrhhtrhtrhtr")
	file.DownloadsRemaining = 0
	configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"] = file

	CleanUp(false)
	test.IsEqualBool(t, helper.FileExists("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7"), false)
	test.IsEqualString(t, configuration.ServerSettings.Files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["e4TjE7CokWK0giiLNxDL"].Name, "")
	test.IsEqualString(t, configuration.ServerSettings.Files["wefffewhtrhhtrhtrhtr"].Name, "")

	test.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "cleanup")
	test.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), true)
	configuration.ServerSettings.DownloadStatus = make(map[string]filestructure.DownloadStatus)
	CleanUp(false)
	test.IsEqualString(t, configuration.ServerSettings.Files["cleanuptest123456789"].Name, "")
	test.IsEqualBool(t, helper.FileExists("test/data/2341354656543213246465465465432456898794"), false)
}
