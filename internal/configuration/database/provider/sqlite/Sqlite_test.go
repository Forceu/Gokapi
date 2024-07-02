//go:build test

package sqlite

import (
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"math"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

var config = models.DbConnection{
	HostUrl: "./test/newfolder/gokapi.sqlite",
	Type:    0, // dbabstraction.TypeSqlite
}

func TestMain(m *testing.M) {
	_ = os.Mkdir("test", 0777)
	exitVal := m.Run()
	_ = os.RemoveAll("test")
	os.Exit(exitVal)
}

var dbInstance DatabaseProvider

func TestInit(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	test.FolderExists(t, "./test/newfolder")
	instance.Close()
	err = os.WriteFile("./test/newfolder/gokapi2.sqlite", []byte("invalid"), 0700)
	test.IsNil(t, err)
	instance, err = New(models.DbConnection{
		HostUrl: "./test/newfolder/gokapi2.sqlite",
		Type:    0, // dbabstraction.TypeSqlite
	})
	test.IsNotNil(t, err)
}

func TestClose(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	instance.Close()
	instance, err = New(config)
	test.IsNil(t, err)
	dbInstance = instance
}

func TestMetaData(t *testing.T) {
	files := dbInstance.GetAllMetadata()
	test.IsEqualInt(t, len(files), 0)

	dbInstance.SaveMetaData(models.File{Id: "testfile", Name: "test.txt", ExpireAt: time.Now().Add(time.Hour).Unix()})
	files = dbInstance.GetAllMetadata()
	test.IsEqualInt(t, len(files), 1)
	test.IsEqualString(t, files["testfile"].Name, "test.txt")

	file, ok := dbInstance.GetMetaDataById("testfile")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Id, "testfile")
	_, ok = dbInstance.GetMetaDataById("invalid")
	test.IsEqualBool(t, ok, false)

	test.IsEqualInt(t, len(dbInstance.GetAllMetadata()), 1)
	dbInstance.DeleteMetaData("invalid")
	test.IsEqualInt(t, len(dbInstance.GetAllMetadata()), 1)

	test.IsEqualBool(t, file.UnlimitedDownloads, false)
	test.IsEqualBool(t, file.UnlimitedTime, false)

	dbInstance.DeleteMetaData("testfile")
	test.IsEqualInt(t, len(dbInstance.GetAllMetadata()), 0)

	dbInstance.SaveMetaData(models.File{
		Id:                 "test2",
		Name:               "test2",
		UnlimitedDownloads: true,
		UnlimitedTime:      false,
	})

	file, ok = dbInstance.GetMetaDataById("test2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedDownloads, true)
	test.IsEqualBool(t, file.UnlimitedTime, false)

	dbInstance.SaveMetaData(models.File{
		Id:                 "test3",
		Name:               "test3",
		UnlimitedDownloads: false,
		UnlimitedTime:      true,
	})
	file, ok = dbInstance.GetMetaDataById("test3")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, file.UnlimitedDownloads, false)
	test.IsEqualBool(t, file.UnlimitedTime, true)
	dbInstance.Close()
	defer test.ExpectPanic(t)
	_ = dbInstance.GetAllMetadata()
}

func TestDatabaseProvider_GetType(t *testing.T) {
	test.IsEqualInt(t, dbInstance.GetType(), 0)
}

func TestGetAllMetaDataIds(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	dbInstance = instance

	ids := dbInstance.GetAllMetaDataIds()
	test.IsEqualString(t, ids[0], "test2")
	test.IsEqualString(t, ids[1], "test3")

	dbInstance.Close()
	defer test.ExpectPanic(t)
	_ = dbInstance.GetAllMetaDataIds()
}

func TestHotlink(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	dbInstance = instance

	dbInstance.SaveHotlink(models.File{Id: "testfile", Name: "test.txt", HotlinkId: "testlink", ExpireAt: time.Now().Add(time.Hour).Unix()})

	hotlink, ok := dbInstance.GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testfile")
	_, ok = dbInstance.GetHotlink("invalid")
	test.IsEqualBool(t, ok, false)

	dbInstance.DeleteHotlink("invalid")
	_, ok = dbInstance.GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	dbInstance.DeleteHotlink("testlink")
	_, ok = dbInstance.GetHotlink("testlink")
	test.IsEqualBool(t, ok, false)

	dbInstance.SaveHotlink(models.File{Id: "testfile", Name: "test.txt", HotlinkId: "testlink", ExpireAt: 0, UnlimitedTime: true})
	hotlink, ok = dbInstance.GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testfile")

	dbInstance.SaveHotlink(models.File{Id: "file2", Name: "file2.txt", HotlinkId: "link2", ExpireAt: time.Now().Add(time.Hour).Unix()})
	dbInstance.SaveHotlink(models.File{Id: "file3", Name: "file3.txt", HotlinkId: "link3", ExpireAt: time.Now().Add(time.Hour).Unix()})

	hotlinks := dbInstance.GetAllHotlinks()
	test.IsEqualInt(t, len(hotlinks), 3)
	test.IsEqualBool(t, slices.Contains(hotlinks, "testlink"), true)
	test.IsEqualBool(t, slices.Contains(hotlinks, "link2"), true)
	test.IsEqualBool(t, slices.Contains(hotlinks, "link3"), true)
	dbInstance.DeleteHotlink("")
	hotlinks = dbInstance.GetAllHotlinks()
	test.IsEqualInt(t, len(hotlinks), 3)
}

func TestApiKey(t *testing.T) {
	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "newkey",
		FriendlyName: "New Key",
		LastUsed:     100,
		Permissions:  20,
	})
	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "newkey2",
		FriendlyName: "New Key2",
		LastUsed:     200,
		Permissions:  40,
	})

	keys := dbInstance.GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 2)
	test.IsEqualString(t, keys["newkey"].FriendlyName, "New Key")
	test.IsEqualString(t, keys["newkey"].Id, "newkey")
	test.IsEqualInt64(t, keys["newkey"].LastUsed, 100)
	test.IsEqualBool(t, keys["newkey"].Permissions == 20, true)

	test.IsEqualInt(t, len(dbInstance.GetAllApiKeys()), 2)
	dbInstance.DeleteApiKey("newkey2")
	test.IsEqualInt(t, len(dbInstance.GetAllApiKeys()), 1)

	key, ok := dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "New Key")
	_, ok = dbInstance.GetApiKey("newkey2")
	test.IsEqualBool(t, ok, false)

	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "newkey",
		FriendlyName: "Old Key",
		LastUsed:     100,
	})
	key, ok = dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.FriendlyName, "Old Key")
}

func TestSession(t *testing.T) {
	renewAt := time.Now().Add(1 * time.Hour).Unix()
	dbInstance.SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})

	session, ok := dbInstance.GetSession("newsession")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, session.RenewAt == renewAt, true)

	dbInstance.DeleteSession("newsession")
	_, ok = dbInstance.GetSession("newsession")
	test.IsEqualBool(t, ok, false)

	dbInstance.SaveSession("newsession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})

	dbInstance.SaveSession("anothersession", models.Session{
		RenewAt:    renewAt,
		ValidUntil: time.Now().Add(2 * time.Hour).Unix(),
	})
	_, ok = dbInstance.GetSession("newsession")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.GetSession("anothersession")
	test.IsEqualBool(t, ok, true)

	dbInstance.DeleteAllSessions()
	_, ok = dbInstance.GetSession("newsession")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetSession("anothersession")
	test.IsEqualBool(t, ok, false)
}

func TestUploadDefaults(t *testing.T) {
	defaults, ok := dbInstance.GetUploadDefaults()
	test.IsEqualBool(t, ok, false)
	dbInstance.SaveUploadDefaults(models.LastUploadValues{
		Downloads:         20,
		TimeExpiry:        30,
		Password:          "abcd",
		UnlimitedDownload: true,
		UnlimitedTime:     true,
	})
	defaults, ok = dbInstance.GetUploadDefaults()
	test.IsEqualBool(t, ok, true)
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
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete1",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete2",
		CurrentStatus: 1,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete3",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete4",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete5",
		CurrentStatus: 1,
	})
	currentTime = orgiginalFunc

	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep1",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep2",
		CurrentStatus: 1,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep3",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep4",
		CurrentStatus: 0,
	})
	dbInstance.SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep5",
		CurrentStatus: 1,
	})
	for _, item := range []string{"ctodelete1", "ctodelete2", "ctodelete3", "ctodelete4", "ctokeep1", "ctokeep2", "ctokeep3", "ctokeep4"} {
		_, result := dbInstance.GetUploadStatus(item)
		test.IsEqualBool(t, result, true)
	}
	dbInstance.RunGarbageCollection()
	for _, item := range []string{"ctodelete1", "ctodelete2", "ctodelete3", "ctodelete4"} {
		_, result := dbInstance.GetUploadStatus(item)
		test.IsEqualBool(t, result, false)
	}
	for _, item := range []string{"ctokeep1", "ctokeep2", "ctokeep3", "ctokeep4"} {
		_, result := dbInstance.GetUploadStatus(item)
		test.IsEqualBool(t, result, true)
	}
}

func TestGarbageCollectionSessions(t *testing.T) {
	dbInstance.SaveSession("todelete1", models.Session{
		RenewAt:    time.Now().Add(-10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(-10 * time.Second).Unix(),
	})
	dbInstance.SaveSession("todelete2", models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(-10 * time.Second).Unix(),
	})
	dbInstance.SaveSession("tokeep1", models.Session{
		RenewAt:    time.Now().Add(-10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(10 * time.Second).Unix(),
	})
	dbInstance.SaveSession("tokeep2", models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(10 * time.Second).Unix(),
	})
	for _, item := range []string{"todelete1", "todelete2", "tokeep1", "tokeep2"} {
		_, result := dbInstance.GetSession(item)
		test.IsEqualBool(t, result, true)
	}
	dbInstance.RunGarbageCollection()
	for _, item := range []string{"todelete1", "todelete2"} {
		_, result := dbInstance.GetSession(item)
		test.IsEqualBool(t, result, false)
	}
	for _, item := range []string{"tokeep1", "tokeep2"} {
		_, result := dbInstance.GetSession(item)
		test.IsEqualBool(t, result, true)
	}
}

func TestEnd2EndInfo(t *testing.T) {
	info := dbInstance.GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 0)
	test.IsEqualBool(t, info.HasBeenSetUp(), false)

	dbInstance.SaveEnd2EndInfo(models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("testNonce1"),
		Content:        []byte("testContent1"),
		AvailableFiles: nil,
	})

	info = dbInstance.GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 1)
	test.IsEqualBool(t, info.HasBeenSetUp(), true)
	test.IsEqualByteSlice(t, info.Nonce, []byte("testNonce1"))
	test.IsEqualByteSlice(t, info.Content, []byte("testContent1"))
	test.IsEqualBool(t, len(info.AvailableFiles) == 0, true)

	dbInstance.SaveEnd2EndInfo(models.E2EInfoEncrypted{
		Version:        2,
		Nonce:          []byte("testNonce2"),
		Content:        []byte("testContent2"),
		AvailableFiles: nil,
	})

	info = dbInstance.GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 2)
	test.IsEqualBool(t, info.HasBeenSetUp(), true)
	test.IsEqualByteSlice(t, info.Nonce, []byte("testNonce2"))
	test.IsEqualByteSlice(t, info.Content, []byte("testContent2"))
	test.IsEqualBool(t, len(info.AvailableFiles) == 0, true)

	dbInstance.DeleteEnd2EndInfo()
	info = dbInstance.GetEnd2EndInfo()
	test.IsEqualInt(t, info.Version, 0)
	test.IsEqualBool(t, info.HasBeenSetUp(), false)
}

func TestUpdateTimeApiKey(t *testing.T) {
	retrievedKey, ok := dbInstance.GetApiKey("key1")
	test.IsEqualBool(t, ok, false)
	test.IsEqualString(t, retrievedKey.Id, "")

	key := models.ApiKey{
		Id:           "key1",
		FriendlyName: "key1",
		LastUsed:     100,
	}
	dbInstance.SaveApiKey(key)
	key = models.ApiKey{
		Id:           "key2",
		FriendlyName: "key2",
		LastUsed:     200,
	}
	dbInstance.SaveApiKey(key)

	retrievedKey, ok = dbInstance.GetApiKey("key1")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key1")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 100)
	retrievedKey, ok = dbInstance.GetApiKey("key2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key2")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 200)

	key.LastUsed = 300
	dbInstance.UpdateTimeApiKey(key)

	retrievedKey, ok = dbInstance.GetApiKey("key1")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key1")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 100)
	retrievedKey, ok = dbInstance.GetApiKey("key2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedKey.Id, "key2")
	test.IsEqualInt64(t, retrievedKey.LastUsed, 300)
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
		dbInstance.SaveMetaData(file)
		retrievedFile, ok := dbInstance.GetMetaDataById(file.Id)
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, retrievedFile.Name, file.Name)
		dbInstance.DeleteMetaData(file.Id)
		_, ok = dbInstance.GetMetaDataById(file.Id)
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

	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "readtest",
		FriendlyName: "readtest",
		LastUsed:     40000,
	})
	simulatedConnection := func(t *testing.T) {
		_, ok := dbInstance.GetApiKey("readtest")
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

func TestUploadStatus(t *testing.T) {
	allStatus := dbInstance.GetAllUploadStatus()
	found := false
	test.IsEqualInt(t, len(allStatus), 5)
	for _, status := range allStatus {
		if status.ChunkId == "ctokeep5" {
			found = true
		}
	}
	test.IsEqualBool(t, found, true)
	newStatus := models.UploadStatus{
		ChunkId:       "testid",
		CurrentStatus: 1,
	}
	retrievedStatus, ok := dbInstance.GetUploadStatus("testid")
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, retrievedStatus == models.UploadStatus{}, true)
	dbInstance.SaveUploadStatus(newStatus)
	retrievedStatus, ok = dbInstance.GetUploadStatus("testid")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedStatus.ChunkId, "testid")
	test.IsEqualInt(t, retrievedStatus.CurrentStatus, 1)
	allStatus = dbInstance.GetAllUploadStatus()
	test.IsEqualInt(t, len(allStatus), 6)
}

func TestDatabaseProvider_Upgrade(t *testing.T) {
	dbInstance.Upgrade(19)
}

func TestRawSql(t *testing.T) {
	dbInstance.Close()
	dbInstance.sqliteDb = nil
	defer test.ExpectPanic(t)
	_ = dbInstance.rawSqlite("Select * from Sessions")
}
