package database

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"math"
	"os"
	"sync"
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
	Init("./test", "gokapi.sqlite")
	test.IsEqualBool(t, sqliteDb != nil, true)
	// Test that second init doesn't raise an error
	Init("./test", "gokapi.sqlite")
}

func TestClose(t *testing.T) {
	test.IsEqualBool(t, sqliteDb != nil, true)
	Close()
	test.IsEqualBool(t, sqliteDb == nil, true)
	Init("./test", "gokapi.sqlite")
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

	SaveHotlink(models.File{Id: "testhfile", Name: "testh.txt", HotlinkId: "testlink", ExpireAt: 0, UnlimitedTime: true})
	hotlink, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testhfile")
}

func TestApiKey(t *testing.T) {
	SaveApiKey(models.ApiKey{
		Id:             "newkey",
		FriendlyName:   "New Key",
		LastUsedString: "LastUsed",
		LastUsed:       100,
		Permissions:    20,
	})
	SaveApiKey(models.ApiKey{
		Id:             "newkey2",
		FriendlyName:   "New Key2",
		LastUsedString: "LastUsed2",
		LastUsed:       200,
		Permissions:    40,
	})

	keys := GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 2)
	test.IsEqualString(t, keys["newkey"].FriendlyName, "New Key")
	test.IsEqualString(t, keys["newkey"].Id, "newkey")
	test.IsEqualString(t, keys["newkey"].LastUsedString, "LastUsed")
	test.IsEqualInt64(t, keys["newkey"].LastUsed, 100)
	test.IsEqualInt(t, keys["newkey"].Permissions, 20)

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
	})
	key, ok = GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Old Key")
}

func TestSession(t *testing.T) {
	renewAt := time.Now().Add(1 * time.Hour).Unix()
	SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})

	session, ok := GetSession("newsession")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, session.RenewAt == renewAt, true)

	DeleteSession("newsession")
	_, ok = GetSession("newsession")
	test.IsEqualBool(t, ok, false)

	SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})

	SaveSession("anothersession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})
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

func TestGarbageCollectionUploads(t *testing.T) {
	orgiginalFunc := currentTime
	currentTime = func() time.Time {
		return time.Now().Add(-25 * time.Hour)
	}
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete1",
		CurrentStatus: 0,
		LastUpdate:    time.Now().Add(-24 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete2",
		CurrentStatus: 1,
		LastUpdate:    time.Now().Add(-24 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete3",
		CurrentStatus: 0,
		LastUpdate:    0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete4",
		CurrentStatus: 0,
		LastUpdate:    time.Now().Add(-20 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete5",
		CurrentStatus: 1,
		LastUpdate:    time.Now().Add(40 * time.Hour).Unix(),
	})
	currentTime = orgiginalFunc

	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep1",
		CurrentStatus: 0,
		LastUpdate:    time.Now().Add(-24 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep2",
		CurrentStatus: 1,
		LastUpdate:    time.Now().Add(-24 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep3",
		CurrentStatus: 0,
		LastUpdate:    0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep4",
		CurrentStatus: 0,
		LastUpdate:    time.Now().Add(-20 * time.Hour).Unix(),
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep5",
		CurrentStatus: 1,
		LastUpdate:    time.Now().Add(40 * time.Hour).Unix(),
	})
	for _, item := range []string{"ctodelete1", "ctodelete2", "ctodelete3", "ctodelete4", "ctokeep1", "ctokeep2", "ctokeep3", "ctokeep4"} {
		_, result := GetUploadStatus(item)
		test.IsEqualBool(t, result, true)
	}
	RunGarbageCollection()
	for _, item := range []string{"ctodelete1", "ctodelete2", "ctodelete3", "ctodelete4"} {
		_, result := GetUploadStatus(item)
		test.IsEqualBool(t, result, false)
	}
	for _, item := range []string{"ctokeep1", "ctokeep2", "ctokeep3", "ctokeep4"} {
		_, result := GetUploadStatus(item)
		test.IsEqualBool(t, result, true)
	}
}

func TestGarbageCollectionSessions(t *testing.T) {
	SaveSession("todelete1", models.Session{
		RenewAt:    time.Now().Add(-10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(-10 * time.Second).Unix(),
	})
	SaveSession("todelete2", models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(-10 * time.Second).Unix(),
	})
	SaveSession("tokeep1", models.Session{
		RenewAt:    time.Now().Add(-10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(10 * time.Second).Unix(),
	})
	SaveSession("tokeep2", models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(10 * time.Second).Unix(),
	})
	for _, item := range []string{"todelete1", "todelete2", "tokeep1", "tokeep2"} {
		_, result := GetSession(item)
		test.IsEqualBool(t, result, true)
	}
	RunGarbageCollection()
	for _, item := range []string{"todelete1", "todelete2"} {
		_, result := GetSession(item)
		test.IsEqualBool(t, result, false)
	}
	for _, item := range []string{"tokeep1", "tokeep2"} {
		_, result := GetSession(item)
		test.IsEqualBool(t, result, true)
	}
}

func TestEnd2EndInfo(t *testing.T) {
	info := GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 0)
	test.IsEqualBool(t, info.HasBeenSetUp(), false)

	SaveEnd2EndInfo(models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("testNonce1"),
		Content:        []byte("testContent1"),
		AvailableFiles: []string{"file1_0", "file1_1"},
	})

	info = GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 1)
	test.IsEqualBool(t, info.HasBeenSetUp(), true)
	test.IsEqualByteSlice(t, info.Nonce, []byte("testNonce1"))
	test.IsEqualByteSlice(t, info.Content, []byte("testContent1"))
	test.IsEqualBool(t, len(info.AvailableFiles) == 0, true)

	SaveEnd2EndInfo(models.E2EInfoEncrypted{
		Version:        2,
		Nonce:          []byte("testNonce2"),
		Content:        []byte("testContent2"),
		AvailableFiles: []string{"file2_0", "file2_1"},
	})

	info = GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 2)
	test.IsEqualBool(t, info.HasBeenSetUp(), true)
	test.IsEqualByteSlice(t, info.Nonce, []byte("testNonce2"))
	test.IsEqualByteSlice(t, info.Content, []byte("testContent2"))
	test.IsEqualBool(t, len(info.AvailableFiles) == 0, true)

	DeleteEnd2EndInfo()
	info = GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 0)
	test.IsEqualBool(t, info.HasBeenSetUp(), false)
}

func TestUpdateTimeApiKey(t *testing.T) {

	retrievedKey, ok := GetApiKey("key1")
	test.IsEqualBool(t, ok, false)
	test.IsEqualString(t, retrievedKey.Id, "")

	key := models.ApiKey{
		Id:             "key1",
		FriendlyName:   "key1",
		LastUsed:       100,
		LastUsedString: "last1",
	}
	SaveApiKey(key)
	key = models.ApiKey{
		Id:             "key2",
		FriendlyName:   "key2",
		LastUsed:       200,
		LastUsedString: "last2",
	}
	SaveApiKey(key)

	retrievedKey, ok = GetApiKey("key1")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key1")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 100)
	test.IsEqualString(t, retrievedKey.LastUsedString, "last1")
	retrievedKey, ok = GetApiKey("key2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key2")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 200)
	test.IsEqualString(t, retrievedKey.LastUsedString, "last2")

	key.LastUsed = 300
	key.LastUsedString = "last2_1"
	UpdateTimeApiKey(key)

	retrievedKey, ok = GetApiKey("key1")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key1")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 100)
	test.IsEqualString(t, retrievedKey.LastUsedString, "last1")
	retrievedKey, ok = GetApiKey("key2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key2")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 300)
	test.IsEqualString(t, retrievedKey.LastUsedString, "last2_1")
}

func TestParallelConnectionsWritingAndReading(t *testing.T) {
	var wg sync.WaitGroup

	simulatedConnection := func(t *testing.T) {
		file := models.File{
			Id:                 helper.GenerateRandomString(10),
			Name:               helper.GenerateRandomString(10),
			Size:               "10B",
			SHA1:               "1289423794287598237489",
			ExpireAt:           math.MaxInt,
			SizeBytes:          10,
			ExpireAtString:     "Never",
			DownloadsRemaining: 10,
			DownloadCount:      10,
			PasswordHash:       "",
			HotlinkId:          "",
			ContentType:        "",
			AwsBucket:          "",
			Encryption:         models.EncryptionInfo{},
			UnlimitedDownloads: false,
			UnlimitedTime:      false,
		}
		SaveMetaData(file)
		retrievedFile, ok := GetMetaDataById(file.Id)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, retrievedFile.Name, file.Name)
		DeleteMetaData(file.Id)
		_, ok = GetMetaDataById(file.Id)
		test.IsEqualBool(t, ok, false)
	}

	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			simulatedConnection(t)
		}()
	}
	wg.Wait()
}

func TestParallelConnectionsReading(t *testing.T) {
	var wg sync.WaitGroup

	SaveApiKey(models.ApiKey{
		Id:             "readtest",
		FriendlyName:   "readtest",
		LastUsed:       40000,
		LastUsedString: "readtest",
	})
	simulatedConnection := func(t *testing.T) {
		_, ok := GetApiKey("readtest")
		test.IsEqualBool(t, ok, true)
	}

	for i := 1; i <= 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			simulatedConnection(t)
		}()
	}
	wg.Wait()
}
