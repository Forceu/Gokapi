package database

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_DATA_DIR", "test")
	os.Mkdir("test", 0777)
	exitVal := m.Run()
	os.RemoveAll("test")
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	Init("./test/filestorage.db")
	test.IsEqualBool(t, bitcaskDb != nil, true)
	// Test that second init doesn't raise an error
	Init("./test/filestorage.db")
}

func TestClose(t *testing.T) {
	test.IsEqualBool(t, bitcaskDb != nil, true)
	Close()
	test.IsEqualBool(t, bitcaskDb == nil, true)
	Init("./test/filestorage.db")
}

func TestMetaData(t *testing.T) {
	files := GetAllMetadata()
	test.IsEqualInt(t, len(files), 0)

	SaveMetaData(models.File{Id: "testfile", Name: "test.txt", ExpireAt: time.Now().Add(time.Hour).Unix()})
	files = GetAllMetadata()
	test.IsEqualInt(t, len(files), 1)
	test.IsEqualString(t, files["testfile"].Name, "test.txt")

	file, ok := GetMetaDataById("testfile")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Id, "testfile")
	_, ok = GetMetaDataById("invalid")
	test.IsEqualBool(t, ok, false)

	test.IsEqualInt(t, len(GetAllMetadata()), 1)
	DeleteMetaData("invalid")
	test.IsEqualInt(t, len(GetAllMetadata()), 1)
	DeleteMetaData("testfile")
	test.IsEqualInt(t, len(GetAllMetadata()), 0)
}

func TestHotlink(t *testing.T) {
	SaveHotlink(models.File{Id: "testhfile", Name: "testh.txt", HotlinkId: "testlink", ExpireAt: time.Now().Add(time.Hour).Unix()})

	hotlink, ok := GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testhfile")
	_, ok = GetHotlink("invalid")
	test.IsEqualBool(t, ok, false)

	DeleteHotlink("invalid")
	_, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	DeleteHotlink("testlink")
	_, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, false)
}

func TestApiKey(t *testing.T) {
	SaveApiKey(models.ApiKey{
		Id:             "newkey",
		FriendlyName:   "New Key",
		LastUsed:       100,
		LastUsedString: "LastUsed",
	}, false)
	SaveApiKey(models.ApiKey{
		Id:             "newkey2",
		FriendlyName:   "New Key2",
		LastUsed:       200,
		LastUsedString: "LastUsed2",
	}, true)

	keys := GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 2)
	test.IsEqualString(t, keys["newkey"].FriendlyName, "New Key")
	test.IsEqualString(t, keys["newkey"].Id, "newkey")
	test.IsEqualString(t, keys["newkey"].LastUsedString, "LastUsed")
	test.IsEqualBool(t, keys["newkey"].LastUsed == 100, true)

	test.IsEqualInt(t, len(GetAllApiKeys()), 2)
	DeleteApiKey("newkey2")
	test.IsEqualInt(t, len(GetAllApiKeys()), 1)

	key, ok := GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "New Key")
	_, ok = GetApiKey("newkey2")
	test.IsEqualBool(t, ok, false)

	SaveApiKey(models.ApiKey{
		Id:             "newkey",
		FriendlyName:   "Old Key",
		LastUsed:       100,
		LastUsedString: "LastUsed",
	}, false)
	key, ok = GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Old Key")
}

func TestSession(t *testing.T) {
	renewAt := time.Now().Add(1 * time.Hour).Unix()
	SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	}, 2*time.Hour)

	session, ok := GetSession("newsession")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, session.RenewAt == renewAt, true)

	DeleteSession("newsession")
	_, ok = GetSession("newsession")
	test.IsEqualBool(t, ok, false)

	SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	}, 2*time.Hour)

	SaveSession("anothersession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	}, 2*time.Hour)
	_, ok = GetSession("newsession")
	test.IsEqualBool(t, ok, true)
	_, ok = GetSession("anothersession")
	test.IsEqualBool(t, ok, true)

	DeleteAllSessions()
	_, ok = GetSession("newsession")
	test.IsEqualBool(t, ok, false)
	_, ok = GetSession("anothersession")
	test.IsEqualBool(t, ok, false)
}

func TestUploadDefaults(t *testing.T) {
	defaults := GetUploadDefaults()
	test.IsEqualInt(t, defaults.Downloads, 1)
	test.IsEqualInt(t, defaults.TimeExpiry, 14)
	test.IsEqualString(t, defaults.Password, "")
	test.IsEqualBool(t, defaults.UnlimitedDownload, false)
	test.IsEqualBool(t, defaults.UnlimitedTime, false)

	SaveUploadDefaults(models.LastUploadValues{
		Downloads:         20,
		TimeExpiry:        30,
		Password:          "abcd",
		UnlimitedDownload: true,
		UnlimitedTime:     true,
	})
	defaults = GetUploadDefaults()
	test.IsEqualInt(t, defaults.Downloads, 20)
	test.IsEqualInt(t, defaults.TimeExpiry, 30)
	test.IsEqualString(t, defaults.Password, "abcd")
	test.IsEqualBool(t, defaults.UnlimitedDownload, true)
	test.IsEqualBool(t, defaults.UnlimitedTime, true)
}

func TestBinaryConversion(t *testing.T) {
	test.IsEqualInt(t, byteToInt(intToByte(0)), 0)
	test.IsEqualInt(t, byteToInt(intToByte(-100)), -100)
	test.IsEqualInt(t, byteToInt(intToByte(100)), 100)
	test.IsEqualInt(t, byteToInt(intToByte(10000)), 10000)
	test.IsEqualInt(t, byteToInt(intToByte(2147483647)), 2147483647)
	test.IsEqualInt(t, byteToInt(intToByte(-2147483647)), -2147483647)
}

func TestRunGc(t *testing.T) {
	items := bitcaskDb.Len()
	bitcaskDb.PutWithTTL([]byte("test"), []byte("value"), 500*time.Millisecond)
	test.IsEqualInt(t, bitcaskDb.Len(), items+1)
	time.Sleep(501 * time.Millisecond)
	RunGarbageCollection()
	test.IsEqualInt(t, bitcaskDb.Len(), items)
}

func TestGetLengthAvailable(t *testing.T) {
	test.IsEqualInt(t, GetLengthAvailable(), 85)
}
