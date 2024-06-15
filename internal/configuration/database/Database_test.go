//go:build test

package database

import (
	"database/sql"
	"errors"
	"math"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/exp/slices"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
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
	test.IsEqualBool(t, sqliteDb == nil, true)
	Init("./test/newfolder", "gokapi.sqlite")
	test.IsEqualBool(t, sqliteDb != nil, true)
	test.FolderExists(t, "./test/newfolder")
	Close()
	test.IsEqualBool(t, sqliteDb == nil, true)
	err := os.WriteFile("./test/newfolder/gokapi2.sqlite", []byte("invalid"), 0700)
	test.IsNil(t, err)
	Init("./test/newfolder", "gokapi2.sqlite")
}

func TestClose(t *testing.T) {
	test.IsEqualBool(t, sqliteDb != nil, true)
	Close()
	test.IsEqualBool(t, sqliteDb == nil, true)
	mock := setMockDb(t)
	mock.ExpectClose().WillReturnError(errors.New("test"))
	Close()
	restoreDb()
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
	SaveHotlink(models.File{Id: "testfile", Name: "test.txt", HotlinkId: "testlink", ExpireAt: time.Now().Add(time.Hour).Unix()})

	hotlink, ok := GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testfile")
	_, ok = GetHotlink("invalid")
	test.IsEqualBool(t, ok, false)

	DeleteHotlink("invalid")
	_, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	DeleteHotlink("testlink")
	_, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, false)

	SaveHotlink(models.File{Id: "testfile", Name: "test.txt", HotlinkId: "testlink", ExpireAt: 0, UnlimitedTime: true})
	hotlink, ok = GetHotlink("testlink")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, hotlink, "testfile")

	SaveHotlink(models.File{Id: "file2", Name: "file2.txt", HotlinkId: "link2", ExpireAt: time.Now().Add(time.Hour).Unix()})
	SaveHotlink(models.File{Id: "file3", Name: "file3.txt", HotlinkId: "link3", ExpireAt: time.Now().Add(time.Hour).Unix()})

	hotlinks := GetAllHotlinks()
	test.IsEqualInt(t, len(hotlinks), 3)
	test.IsEqualBool(t, slices.Contains(hotlinks, "testlink"), true)
	test.IsEqualBool(t, slices.Contains(hotlinks, "link2"), true)
	test.IsEqualBool(t, slices.Contains(hotlinks, "link3"), true)
	DeleteHotlink("")
	hotlinks = GetAllHotlinks()
	test.IsEqualInt(t, len(hotlinks), 3)

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
	test.IsEqualBool(t, keys["newkey"].Permissions == 20, true)

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

func TestGuestUploadToken(t *testing.T) {
	SaveUploadToken(models.UploadToken{
		Id:             "newtoken",
		LastUsedString: "LastUsed",
		LastUsed:       100,
	})
	SaveUploadToken(models.UploadToken{
		Id:             "newtoken2",
		LastUsedString: "LastUsed2",
		LastUsed:       200,
	})

	tokens := GetAllUploadTokens()
	test.IsEqualInt(t, len(tokens), 2)
	test.IsEqualString(t, tokens["newtoken"].Id, "newtoken")
	test.IsEqualString(t, tokens["newtoken"].LastUsedString, "LastUsed")
	test.IsEqualInt64(t, tokens["newtoken"].LastUsed, 100)

	test.IsEqualInt(t, len(GetAllUploadTokens()), 2)
	DeleteUploadToken("newtoken2")
	test.IsEqualInt(t, len(GetAllUploadTokens()), 1)

	token, ok := GetUploadToken("newtoken")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, token.Id, "newtoken")
	_, ok = GetUploadToken("newtoken2")
	test.IsEqualBool(t, ok, false)

	SaveUploadToken(models.UploadToken{
		Id:             "newtoken",
		LastUsed:       100,
		LastUsedString: "RecentlyUsed",
	})
	token, ok = GetUploadToken("newtoken")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, token.LastUsedString, "RecentlyUsed")
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

func TestColumnExists(t *testing.T) {
	exists, err := ColumnExists("invalid", "invalid")
	test.IsEqualBool(t, exists, false)
	test.IsNil(t, err)
	exists, err = ColumnExists("FileMetaData", "invalid")
	test.IsEqualBool(t, exists, false)
	test.IsNil(t, err)
	exists, err = ColumnExists("FileMetaData", "ExpireAt")
	test.IsEqualBool(t, exists, true)
	test.IsNil(t, err)
	setMockDb(t).ExpectQuery(regexp.QuoteMeta("PRAGMA table_info(error)")).WillReturnError(errors.New("error"))
	exists, err = ColumnExists("error", "error")
	test.IsEqualBool(t, exists, false)
	test.IsNotNil(t, err)
	restoreDb()
	mock := setMockDb(t)

	rows := mock.NewRows([]string{"invalid"}).
		AddRow(0).
		AddRow(1)
	mock.ExpectQuery(regexp.QuoteMeta("PRAGMA table_info(error)")).WillReturnRows(rows)
	exists, err = ColumnExists("error", "error")
	test.IsEqualBool(t, exists, false)
	test.IsNotNil(t, err)
	restoreDb()
}

func TestGarbageCollectionUploads(t *testing.T) {
	orgiginalFunc := currentTime
	currentTime = func() time.Time {
		return time.Now().Add(-25 * time.Hour)
	}
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete1",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete2",
		CurrentStatus: 1,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete3",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete4",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctodelete5",
		CurrentStatus: 1,
	})
	currentTime = orgiginalFunc

	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep1",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep2",
		CurrentStatus: 1,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep3",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep4",
		CurrentStatus: 0,
	})
	SaveUploadStatus(models.UploadStatus{
		ChunkId:       "ctokeep5",
		CurrentStatus: 1,
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

func TestUploadStatus(t *testing.T) {
	allStatus := GetAllUploadStatus()
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
	retrievedStatus, ok := GetUploadStatus("testid")
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, retrievedStatus == models.UploadStatus{}, true)
	SaveUploadStatus(newStatus)
	retrievedStatus, ok = GetUploadStatus("testid")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, retrievedStatus.ChunkId, "testid")
	test.IsEqualInt(t, retrievedStatus.CurrentStatus, 1)
	allStatus = GetAllUploadStatus()
	test.IsEqualInt(t, len(allStatus), 6)
}

var originalDb *sql.DB

func setMockDb(t *testing.T) sqlmock.Sqlmock {
	originalDb = sqliteDb
	db, mock, err := sqlmock.New()
	test.IsNil(t, err)
	sqliteDb = db
	return mock
}
func restoreDb() {
	sqliteDb = originalDb
}
