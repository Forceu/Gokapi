//go:build test

package sqlite

import (
	"math"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
)

var config = models.DbConnection{
	HostUrl: "./test/newfolder/gokapi.sqlite",
	Type:    0, // dbabstraction.TypeSqlite
}
var configUpgrade = models.DbConnection{
	HostUrl: "./test/newfolder/gokapi_old.sqlite",
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
	_, err = New(models.DbConnection{
		HostUrl: "",
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

func TestDatabaseProvider_GetDbVersion(t *testing.T) {
	version := dbInstance.GetDbVersion()
	test.IsEqualInt(t, version, DatabaseSchemeVersion)
	dbInstance.SetDbVersion(99)
	test.IsEqualInt(t, dbInstance.GetDbVersion(), 99)
	dbInstance.SetDbVersion(DatabaseSchemeVersion)
}

func TestDatabaseProvider_GetSchemaVersion(t *testing.T) {
	test.IsEqualInt(t, dbInstance.GetSchemaVersion(), DatabaseSchemeVersion)
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

func TestDatabaseProvider_IncreaseDownloadCount(t *testing.T) {
	newFile := models.File{
		Id:                 "newFileId",
		Name:               "newFileName",
		Size:               "3GB",
		SHA1:               "newSHA1",
		PasswordHash:       "newPassword",
		HotlinkId:          "newHotlink",
		ContentType:        "newContent",
		AwsBucket:          "newAws",
		ExpireAt:           123456,
		SizeBytes:          456789,
		DownloadsRemaining: 11,
		DownloadCount:      2,
		Encryption: models.EncryptionInfo{
			IsEncrypted:         true,
			IsEndToEndEncrypted: true,
			DecryptionKey:       []byte("newDecryptionKey"),
			Nonce:               []byte("newDecryptionNonce"),
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	dbInstance.SaveMetaData(newFile)
	dbInstance.IncreaseDownloadCount(newFile.Id, false)
	retrievedFile, ok := dbInstance.GetMetaDataById(newFile.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, retrievedFile.DownloadCount, 3)
	test.IsEqualInt(t, retrievedFile.DownloadsRemaining, 11)
	newFile.DownloadCount = 3
	test.IsEqual(t, retrievedFile, newFile)

	dbInstance.IncreaseDownloadCount(newFile.Id, true)
	retrievedFile, ok = dbInstance.GetMetaDataById(newFile.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, retrievedFile.DownloadCount, 4)
	test.IsEqualInt(t, retrievedFile.DownloadsRemaining, 10)
	newFile.DownloadCount = 4
	newFile.DownloadsRemaining = 10
	test.IsEqual(t, retrievedFile, newFile)
	dbInstance.DeleteMetaData(newFile.Id)
}

func TestApiKey(t *testing.T) {
	key1 := models.ApiKey{
		Id:           "newkey",
		FriendlyName: "New Key",
		LastUsed:     100,
		Permissions:  20,
		PublicId:     "_n3wkey",
		Expiry:       0,
		IsSystemKey:  false,
		UserId:       5,
	}
	key2 := models.ApiKey{
		Id:           "newkey2",
		FriendlyName: "New Key2",
		PublicId:     "_n3wkey2",
		Expiry:       17362039396,
		LastUsed:     200,
		Permissions:  40,
		IsSystemKey:  true,
		UserId:       10,
	}
	dbInstance.SaveApiKey(key1)
	dbInstance.SaveApiKey(key2)
	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "expiredKey",
		PublicId:     "expiredKey",
		FriendlyName: "expiredKey",
		Expiry:       1,
	})

	keys := dbInstance.GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 2)
	test.IsEqual(t, keys["newkey"], key1)
	test.IsEqual(t, keys["newkey2"], key2)
	dbInstance.DeleteApiKey("newkey2")
	test.IsEqualInt(t, len(dbInstance.GetAllApiKeys()), 1)

	key, ok := dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, key, key1)
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

	session = models.Session{
		RenewAt:    2147483645,
		ValidUntil: 2147483645,
		UserId:     20,
	}
	dbInstance.SaveSession("sess_user1", session)
	dbInstance.SaveSession("sess_user2", session)
	dbInstance.SaveSession("sess_user3", session)
	session.UserId = 40
	dbInstance.SaveSession("sess_user4", session)
	_, ok = dbInstance.GetSession("sess_user1")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.GetSession("sess_user2")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.GetSession("sess_user3")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.GetSession("sess_user4")
	test.IsEqualBool(t, ok, true)
	dbInstance.DeleteAllSessionsByUser(20)
	_, ok = dbInstance.GetSession("sess_user1")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetSession("sess_user2")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetSession("sess_user3")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetSession("sess_user4")
	test.IsEqualBool(t, ok, true)
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
	info := dbInstance.GetEnd2EndInfo(4)
	test.IsEqualInt(t, info.Version, 0)
	test.IsEqualBool(t, info.HasBeenSetUp(), false)

	dbInstance.SaveEnd2EndInfo(models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("testNonce1"),
		Content:        []byte("testContent1"),
		AvailableFiles: nil,
	}, 4)

	info = dbInstance.GetEnd2EndInfo(4)
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
	}, 4)

	info = dbInstance.GetEnd2EndInfo(4)
	test.IsEqualInt(t, info.Version, 2)
	test.IsEqualBool(t, info.HasBeenSetUp(), true)
	test.IsEqualByteSlice(t, info.Nonce, []byte("testNonce2"))
	test.IsEqualByteSlice(t, info.Content, []byte("testContent2"))
	test.IsEqualBool(t, len(info.AvailableFiles) == 0, true)

	dbInstance.DeleteEnd2EndInfo(4)
	info = dbInstance.GetEnd2EndInfo(4)
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
		PublicId:     "key1",
		LastUsed:     100,
	}
	dbInstance.SaveApiKey(key)
	key = models.ApiKey{
		Id:           "key2",
		FriendlyName: "key2",
		PublicId:     "key2",
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

	dbInstance.SaveApiKey(models.ApiKey{
		Id:       "publicTest",
		PublicId: "publicId",
	})
	_, ok = dbInstance.GetApiKey("publicTest")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.GetApiKey("publicId")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetApiKeyByPublicKey("publicTest")
	test.IsEqualBool(t, ok, false)
	keyName, ok := dbInstance.GetApiKeyByPublicKey("publicId")
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, keyName, "publicTest")
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

func TestUsers(t *testing.T) {
	users := dbInstance.GetAllUsers()
	test.IsEqualInt(t, len(users), 0)
	user := models.User{
		Id:            2,
		Name:          "test",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelUser,
		LastOnline:    1337,
		Password:      "123456",
		ResetPassword: true,
	}
	dbInstance.SaveUser(user, false)
	retrievedUser, ok := dbInstance.GetUser(2)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, retrievedUser, user)
	users = dbInstance.GetAllUsers()
	test.IsEqualInt(t, len(users), 1)
	test.IsEqualInt(t, retrievedUser.Id, 2)

	_, ok = dbInstance.GetUser(0)
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetUserByName("invalid")
	test.IsEqualBool(t, ok, false)
	retrievedUser, ok = dbInstance.GetUserByName("test")
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, retrievedUser, user)

	dbInstance.DeleteUser(2)
	_, ok = dbInstance.GetUser(2)
	test.IsEqualBool(t, ok, false)

	user = models.User{
		Id:            1000,
		Name:          "test2",
		Permissions:   models.UserPermissionNone,
		UserLevel:     models.UserLevelAdmin,
		LastOnline:    1338,
		Password:      "1234568",
		ResetPassword: true,
	}
	dbInstance.SaveUser(user, true)
	_, ok = dbInstance.GetUser(1000)
	test.IsEqualBool(t, ok, false)
	retrievedUser, ok = dbInstance.GetUserByName("test2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedUser.Id == 1000, false)
	user.Id = retrievedUser.Id
	test.IsEqual(t, retrievedUser, user)

	dbInstance.UpdateUserLastOnline(retrievedUser.Id)
	retrievedUser, ok = dbInstance.GetUser(retrievedUser.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, time.Now().Unix()-retrievedUser.LastOnline < 5, true)
	test.IsEqualBool(t, time.Now().Unix()-retrievedUser.LastOnline > -1, true)

	user.Name = "test1"
	dbInstance.SaveUser(user, true)
	user.Name = "test3"
	dbInstance.SaveUser(user, true)
	user.Name = "test99"
	user.UserLevel = models.UserLevelSuperAdmin
	dbInstance.SaveUser(user, true)
	user.Name = "test0"
	user.UserLevel = models.UserLevelUser
	dbInstance.SaveUser(user, true)

	users = dbInstance.GetAllUsers()
	test.IsEqualInt(t, len(users), 5)
	test.IsEqualString(t, users[0].Name, "test99")
	test.IsEqualString(t, users[1].Name, "test2")
	test.IsEqualString(t, users[2].Name, "test1")
	test.IsEqualString(t, users[3].Name, "test3")
	test.IsEqualString(t, users[4].Name, "test0")
}

func TestDatabaseProvider_Upgrade(t *testing.T) {
	instance, err := New(configUpgrade)
	test.IsNil(t, err)

	exitCode := 0
	osExit = func(code int) {
		exitCode = code
	}
	instance.SetDbVersion(9)
	instance.Upgrade(instance.GetDbVersion())
	test.IsEqualInt(t, exitCode, 1)

	// exitCode = 0
	// instance.SetDbVersion(6)
	// instance.Upgrade(instance.GetDbVersion())
	// test.IsEqualInt(t, exitCode, 0)

}

func TestRawSql(t *testing.T) {
	dbInstance.Close()
	dbInstance.sqliteDb = nil
	defer test.ExpectPanic(t)
	_ = dbInstance.rawSqlite("Select * from Sessions")
}
