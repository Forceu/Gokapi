package storage

import (
	"bytes"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/downloadstatus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
	"time"
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
	_, result = GetFile("")
	test.IsEqualBool(t, result, false)
	file = models.File{
		Id:                 "testget",
		Name:               "testget",
		SHA256:             "testget",
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	database.SaveMetaData(file)
	_, result = GetFile(file.Id)
	test.IsEqualBool(t, result, false)

}

func TestGetEncInfoFromExistingFile(t *testing.T) {
	configuration.Get().Encryption.Level = 0
	_, result := getEncInfoFromExistingFile("testhash")
	test.IsEqualBool(t, result, true)
	file := models.File{
		Id:     "testhash",
		Name:   "testhash",
		SHA256: "testhash",
		Encryption: models.EncryptionInfo{
			IsEncrypted:   true,
			DecryptionKey: nil,
			Nonce:         nil,
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	database.SaveMetaData(file)
	encinfo, result := getEncInfoFromExistingFile("testhash")
	test.IsEqualBool(t, encinfo.IsEncrypted, false)
	test.IsEqualBool(t, result, true)
	configuration.Get().Encryption.Level = 1
	encinfo, result = getEncInfoFromExistingFile("testhash")
	test.IsEqualBool(t, result, true)
	test.IsEqualBool(t, encinfo.IsEncrypted, true)
	_, result = getEncInfoFromExistingFile("testhashinvalid")
	test.IsEqualBool(t, result, false)
	configuration.Get().Encryption.Level = 0
}

func TestGetFileByHotlink(t *testing.T) {
	_, result := GetFileByHotlink("invalid")
	test.IsEqualBool(t, result, false)
	_, result = GetFileByHotlink("")
	test.IsEqualBool(t, result, false)
	file, ok := GetFileByHotlink("PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Id, "n1tSTAGj8zan9KaT4u6p")
	test.IsEqualString(t, file.Name, "picture.jpg")
	test.IsEqualString(t, file.Size, "4 B")
	test.IsEqualInt(t, file.DownloadsRemaining, 1)
}

func TestAddHotlink(t *testing.T) {
	file := models.File{Name: "test.dat", Id: "testId"}
	addHotlink(&file)
	test.IsEqualString(t, file.HotlinkId, "")
	file = models.File{Name: "test.jpg", Id: "testId", ExpireAt: time.Now().Add(time.Hour).Unix()}
	addHotlink(&file)
	test.IsEqualInt(t, len(file.HotlinkId), 44)
	lastCharacters := file.HotlinkId[len(file.HotlinkId)-4:]
	test.IsEqualBool(t, lastCharacters == ".jpg", true)
	link, ok := database.GetHotlink(file.HotlinkId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, link, "testId")
	file = models.File{Name: "test.jpg", Id: "testId", ExpireAt: time.Now().Add(time.Hour).Unix()}
	file.Encryption.IsEncrypted = true
	file.AwsBucket = "test"
	addHotlink(&file)
	test.IsEqualString(t, file.HotlinkId, "")
}

type testFile struct {
	File    models.File
	Request models.UploadRequest
	Header  multipart.FileHeader
	Content []byte
}

func createTestFile() (testFile, error) {
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
	return testFile{
		File:    file,
		Request: request,
		Header:  header,
		Content: content,
	}, err
}

func TestNewFile(t *testing.T) {

	newFile, err := createTestFile()
	file := newFile.File
	request := newFile.Request
	content := newFile.Content
	header := newFile.Header

	test.IsNil(t, err)
	retrievedFile, ok := database.GetMetaDataById(file.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedFile.Name, "test.dat")
	test.IsEqualString(t, retrievedFile.SHA256, "f1474c19eff0fc8998fa6e1b1f7bf31793b103a6")
	test.IsEqualString(t, retrievedFile.HotlinkId, "")
	test.IsEqualString(t, retrievedFile.PasswordHash, "")
	test.IsEqualString(t, retrievedFile.Size, "35 B")
	test.IsEqualString(t, retrievedFile.ExpireAtString, "2038-01-19 03:13")
	test.IsEqualInt(t, retrievedFile.DownloadsRemaining, 1)
	test.IsEqualInt(t, len(retrievedFile.Id), 20)
	test.IsEqualInt(t, int(retrievedFile.ExpireAt), 2147483600)
	test.IsEqualBool(t, file.UnlimitedTime, false)
	test.IsEqualBool(t, file.UnlimitedDownloads, false)
	idNewFile = file.Id

	request.UnlimitedDownload = true
	file, err = NewFile(bytes.NewReader(content), &header, request)
	test.IsEqualBool(t, file.UnlimitedTime, false)
	test.IsEqualBool(t, file.UnlimitedDownloads, true)
	request.UnlimitedDownload = false
	request.UnlimitedTime = true
	file, err = NewFile(bytes.NewReader(content), &header, request)
	test.IsEqualBool(t, file.UnlimitedTime, true)
	test.IsEqualBool(t, file.UnlimitedDownloads, false)
	request.UnlimitedDownload = true
	file, err = NewFile(bytes.NewReader(content), &header, request)
	test.IsEqualBool(t, file.UnlimitedTime, true)
	test.IsEqualBool(t, file.UnlimitedDownloads, true)

	createBigFile("bigfile", 20)
	bigFile, _ := os.Open("bigfile")
	mimeHeader := make(textproto.MIMEHeader)
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
	retrievedFile, ok = database.GetMetaDataById(file.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedFile.Name, "bigfile")
	test.IsEqualString(t, retrievedFile.SHA256, "9674344c90c2f0646f0b78026e127c9b86e3ad77")
	test.IsEqualString(t, retrievedFile.Size, "20.0 MB")
	_, err = bigFile.Seek(0, io.SeekStart)
	test.IsNil(t, err)
	// Testing removal of temp file
	test.IsEqualString(t, retrievedFile.Name, "bigfile")
	test.IsEqualString(t, retrievedFile.SHA256, "9674344c90c2f0646f0b78026e127c9b86e3ad77")
	test.IsEqualString(t, retrievedFile.Size, "20.0 MB")
	bigFile.Close()
	os.Remove("bigfile")

	createBigFile("bigfile", 50)
	bigFile, _ = os.Open("bigfile")
	mimeHeader = make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Disposition", "form-data; name=\"file\"; filename=\"bigfile\"")
	mimeHeader.Set("Content-Type", "application/binary")
	header = multipart.FileHeader{
		Filename: "bigfile",
		Header:   mimeHeader,
		Size:     int64(50) * 1024 * 1024,
	}
	request = models.UploadRequest{
		AllowedDownloads: 1,
		Expiry:           999,
		ExpiryTimestamp:  2147483600,
		MaxMemory:        10,
		DataDir:          "test/data",
	}
	file, err = NewFile(bigFile, &header, request)
	test.IsNotNil(t, err)
	retrievedFile, ok = database.GetMetaDataById(file.Id)
	test.IsEqualBool(t, ok, false)
	bigFile.Close()
	os.Remove("bigfile")

	configuration.Get().Encryption.Level = 1
	previousSalt := configuration.Get().Authentication.SaltFiles
	configuration.Get().Authentication.SaltFiles = "testsaltfiles"
	cipher, err := encryption.GetRandomCipher()
	test.IsNil(t, err)
	encryption.Init(models.Configuration{Encryption: models.Encryption{
		Level:  encryption.LocalEncryptionStored,
		Cipher: cipher,
	}})

	newFile, err = createTestFile()
	test.IsNil(t, err)
	retrievedFile, ok = database.GetMetaDataById(newFile.File.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedFile.SHA256, "5bbfa18805eb12c678cfd284c956718d57039e37")

	createBigFile("bigfile", 20)
	header.Size = int64(20) * 1024 * 1024
	bigFile, _ = os.Open("bigfile")
	file, err = NewFile(bigFile, &header, request)
	test.IsNil(t, err)
	retrievedFile, ok = database.GetMetaDataById(file.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedFile.Name, "bigfile")
	test.IsEqualString(t, retrievedFile.SHA256, "c1c165c30d0def15ba2bc8f1bd243be13b8c8fe7")

	bigFile.Close()
	database.DeleteMetaData(retrievedFile.Id)

	bigFile, _ = os.Open("bigfile")
	file, err = NewFile(bigFile, &header, request)
	test.IsNil(t, err)
	retrievedFile, ok = database.GetMetaDataById(file.Id)
	test.IsEqualBool(t, ok, true)
	os.Remove("bigfile")

	configuration.Get().Authentication.SaltFiles = previousSalt
	configuration.Get().Encryption.Level = 0

	if aws.IsIncludedInBuild {
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
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		file, err = NewFile(bytes.NewReader(content), &header, request)
		test.IsNil(t, err)
		retrievedFile, ok = database.GetMetaDataById(file.Id)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, retrievedFile.Name, "bigfile")
		test.IsEqualString(t, retrievedFile.SHA256, "f1474c19eff0fc8998fa6e1b1f7bf31793b103a6")
		test.IsEqualString(t, retrievedFile.Size, "20.0 MB")
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
	newFile, err := createTestFile()
	test.IsNil(t, err)
	file = newFile.File
	database.SaveMetaData(file)
	r = httptest.NewRequest("GET", "/upload", nil)
	w = httptest.NewRecorder()
	cipher, err := encryption.GetRandomCipher()
	test.IsNil(t, err)
	nonce, err := encryption.GetRandomNonce()
	test.IsNil(t, err)
	encryption.Init(models.Configuration{Encryption: models.Encryption{
		Level:  encryption.LocalEncryptionStored,
		Cipher: cipher,
	}})
	file.Encryption.IsEncrypted = true
	file.Encryption.DecryptionKey = cipher
	file.Encryption.Nonce = nonce
	defer test.ExpectPanic(t)
	ServeFile(file, w, r, true)
}

func TestCleanUp(t *testing.T) {
	files := database.GetAllMetadata()
	downloadstatus.DeleteAll()
	downloadstatus.SetDownload(files["cleanuptest123456789"])

	test.IsEqualString(t, files["cleanuptest123456789"].Name, "cleanup")
	test.IsEqualString(t, files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	test.IsEqualString(t, files["deletedfile123456789"].Name, "DeletedFile")
	test.IsEqualString(t, files["unlimitedDownload"].Name, "unlimitedDownload")
	test.IsEqualString(t, files["unlimitedTime"].Name, "unlimitedTime")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")

	CleanUp(false)
	files = database.GetAllMetadata()
	test.IsEqualString(t, files["cleanuptest123456789"].Name, "cleanup")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")
	test.IsEqualString(t, files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")
	test.IsEqualString(t, files["unlimitedDownload"].Name, "unlimitedDownload")
	test.IsEqualString(t, files["unlimitedTime"].Name, "unlimitedTime")
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")

	file, _ := GetFile("n1tSTAGj8zan9KaT4u6p")
	file.DownloadsRemaining = 0
	database.SaveMetaData(file)
	files = database.GetAllMetadata()

	CleanUp(false)
	files = database.GetAllMetadata()
	test.FileDoesNotExist(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, files["Wzol7LyY2QVczXynJtVo"].Name, "smallfile2")
	test.IsEqualString(t, files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("Wzol7LyY2QVczXynJtVo")
	file.DownloadsRemaining = 0
	database.SaveMetaData(file)

	CleanUp(false)
	files = database.GetAllMetadata()
	test.FileExists(t, "test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7")
	test.IsEqualString(t, files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, files["e4TjE7CokWK0giiLNxDL"].Name, "smallfile2")
	test.IsEqualString(t, files["wefffewhtrhhtrhtrhtr"].Name, "smallfile3")

	file, _ = GetFile("e4TjE7CokWK0giiLNxDL")
	file.DownloadsRemaining = 0
	database.SaveMetaData(file)
	file, _ = GetFile("wefffewhtrhhtrhtrhtr")
	file.DownloadsRemaining = 0
	database.SaveMetaData(file)

	CleanUp(false)
	files = database.GetAllMetadata()
	test.FileDoesNotExist(t, "test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7")
	test.IsEqualString(t, files["Wzol7LyY2QVczXynJtVo"].Name, "")
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.IsEqualString(t, files["deletedfile123456789"].Name, "")
	test.IsEqualString(t, files["e4TjE7CokWK0giiLNxDL"].Name, "")
	test.IsEqualString(t, files["wefffewhtrhhtrhtrhtr"].Name, "")

	test.IsEqualString(t, files["cleanuptest123456789"].Name, "cleanup")
	test.FileExists(t, "test/data/2341354656543213246465465465432456898794")

	downloadstatus.DeleteAll()
	CleanUp(false)
	files = database.GetAllMetadata()
	test.IsEqualString(t, files["cleanuptest123456789"].Name, "")
	test.FileDoesNotExist(t, "test/data/2341354656543213246465465465432456898794")

	if aws.IsIncludedInBuild {
		testconfiguration.EnableS3()
		config, ok := cloudconfig.Load()
		test.IsEqualBool(t, ok, true)
		ok = aws.Init(config.Aws)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, files["awsTest1234567890123"].Name, "Aws Test File")
		testconfiguration.DisableS3()
	}
	// Doesn't really test anything
	CleanUp(true)
}

func TestDeleteFile(t *testing.T) {
	testconfiguration.Create(true)
	configuration.Load()
	files := database.GetAllMetadata()
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "picture.jpg")
	test.FileExists(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	result := DeleteFile("n1tSTAGj8zan9KaT4u6p", true)
	time.Sleep(time.Second)
	test.IsEqualBool(t, result, true)
	files = database.GetAllMetadata()
	test.IsEqualString(t, files["n1tSTAGj8zan9KaT4u6p"].Name, "")
	test.FileDoesNotExist(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
	result = DeleteFile("invalid", true)
	time.Sleep(time.Second)
	test.IsEqualBool(t, result, false)
	result = DeleteFile("", true)
	time.Sleep(time.Second)
	test.IsEqualBool(t, result, false)

	testfile := models.File{Id: "testfiledownload", DownloadsRemaining: 1, ExpireAt: 2147483646}
	database.SaveMetaData(testfile)
	downloadstatus.SetDownload(testfile)
	file, ok := database.GetMetaDataById("testfiledownload")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.ExpireAt != 0, true)
	DeleteFile(file.Id, false)
	file, ok = database.GetMetaDataById("testfiledownload")
	test.IsEqualInt(t, int(file.ExpireAt), 0)

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
		database.SaveMetaData(awsFile)
		files = database.GetAllMetadata()
		result, size, err := aws.FileExists(files["awsTest1234567890123"])
		test.IsEqualBool(t, result, true)
		test.IsNil(t, err)
		DeleteFile("awsTest1234567890123", true)
		time.Sleep(5 * time.Second)
		result, size, err = aws.FileExists(awsFile)
		test.IsEqualBool(t, result, false)
		test.IsEqualInt(t, int(size), 0)
		test.IsNil(t, err)
		testconfiguration.DisableS3()
	}
}

func TestRequiresClientDecryption(t *testing.T) {
	file := models.File{
		Id:        "test",
		AwsBucket: "bucket",
		Encryption: models.EncryptionInfo{
			IsEncrypted: true,
		},
	}
	result := RequiresClientDecryption(file)
	test.IsEqualBool(t, result, true)
	file.Encryption.IsEncrypted = false
	result = RequiresClientDecryption(file)
	test.IsEqualBool(t, result, false)
	file.AwsBucket = ""
	result = RequiresClientDecryption(file)
	test.IsEqualBool(t, result, false)
	file.Encryption.IsEncrypted = true
	result = RequiresClientDecryption(file)
	test.IsEqualBool(t, result, false)

}

func createBigFile(name string, megabytes int64) {
	size := megabytes * 1024 * 1024
	file, _ := os.Create(name)
	_, _ = file.Seek(size-1, 0)
	_, _ = file.Write([]byte{0})
	_ = file.Close()
}

func TestDeleteAllEncrypted(t *testing.T) {
	file := models.File{
		Id:            "testEncDelEnc",
		UnlimitedTime: true,
		Encryption: models.EncryptionInfo{
			IsEncrypted: true,
		},
	}
	database.SaveMetaData(file)
	file = models.File{
		Id:            "testEncDelUn",
		UnlimitedTime: true,
		Encryption: models.EncryptionInfo{
			IsEncrypted: false,
		},
	}
	database.SaveMetaData(file)
	data, ok := database.GetMetaDataById("testEncDelEnc")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, data.UnlimitedTime, true)
	data, ok = database.GetMetaDataById("testEncDelUn")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, data.UnlimitedTime, true)
	DeleteAllEncrypted()
	data, ok = database.GetMetaDataById("testEncDelEnc")
	test.IsEqualBool(t, data.UnlimitedTime, false)
	data, ok = database.GetMetaDataById("testEncDelUn")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, data.UnlimitedTime, true)
}

func TestWriteDownloadHeaders(t *testing.T) {
	file := models.File{Name: "testname", ContentType: "testtype"}
	w, _ := test.GetRecorder("GET", "/test", nil, nil, nil)
	writeDownloadHeaders(file, w, true)
	test.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "attachment; filename=\"testname\"")
	w, _ = test.GetRecorder("GET", "/test", nil, nil, nil)
	writeDownloadHeaders(file, w, false)
	test.IsEqualString(t, w.Result().Header.Get("Content-Disposition"), "inline; filename=\"testname\"")
	test.IsEqualString(t, w.Result().Header.Get("Content-Type"), "testtype")
	file.Encryption.IsEncrypted = true
	w, _ = test.GetRecorder("GET", "/test", nil, nil, nil)
	writeDownloadHeaders(file, w, false)
	test.IsEqualString(t, w.Result().Header.Get("Accept-Ranges"), "bytes")
}
