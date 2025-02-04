package redis

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	redigo "github.com/gomodule/redigo/redis"
	"log"
	"os"
	"slices"
	"testing"
	"time"
)

var config = models.DbConnection{
	RedisPrefix: "test_",
	HostUrl:     "127.0.0.1:16379",
	Type:        1, // dbabstraction.TypeRedis
}

var mRedis *miniredis.Miniredis

func TestMain(m *testing.M) {

	mRedis = miniredis.NewMiniRedis()
	err := mRedis.StartAddr("127.0.0.1:16379")
	if err != nil {
		log.Fatal("Could not start miniredis")
	}
	defer mRedis.Close()
	exitVal := m.Run()
	os.Exit(exitVal)
}

var dbInstance DatabaseProvider

func TestDatabaseProvider_Init(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	instance.Close()
	_, err = New(models.DbConnection{})
	test.IsNotNil(t, err)
	defer test.ExpectPanic(t)
	_, err = New(models.DbConnection{
		RedisPrefix: "test_",
		HostUrl:     "invalid:11",
		Type:        1, // dbabstraction.TypeRedis
	})
	test.IsNotNil(t, err)
}

func TestDatabaseProvider_GetType(t *testing.T) {
	test.IsEqualInt(t, dbInstance.GetType(), 1)
}

func TestDatabaseProvider_GetSchemaVersion(t *testing.T) {
	test.IsEqualInt(t, dbInstance.GetSchemaVersion(), DatabaseSchemeVersion)
}

func TestDatabaseProvider_Upgrade(t *testing.T) {
	var err error
	dbInstance, err = New(config)
	test.IsNil(t, err)
	dbInstance.Upgrade(19)
}

func TestDatabaseProvider_GetDbVersion(t *testing.T) {
	version := dbInstance.GetDbVersion()
	test.IsEqualInt(t, version, DatabaseSchemeVersion)
	dbInstance.SetDbVersion(99)
	test.IsEqualInt(t, dbInstance.GetDbVersion(), 99)
	dbInstance.SetDbVersion(DatabaseSchemeVersion)
}

func TestDatabaseProvider_RunGarbageCollection(t *testing.T) {
	dbInstance.RunGarbageCollection()
}

func TestGetDialOptions(t *testing.T) {
	result := getDialOptions(config)
	test.IsEqualInt(t, len(result), 1)
	newConfig := config
	newConfig.Username = "123"
	newConfig.Password = "456"
	newConfig.RedisUseSsl = true
	result = getDialOptions(newConfig)
	test.IsEqualInt(t, len(result), 4)
}

func TestGetKey(t *testing.T) {
	key, ok := dbInstance.getKeyString("test1")
	test.IsEqualString(t, key, "")
	test.IsEqualBool(t, ok, false)
	dbInstance.setKey("test1", "content")
	key, ok = dbInstance.getKeyString("test1")
	test.IsEqualString(t, key, "content")
	test.IsEqualBool(t, ok, true)
	dbInstance.deleteKey("test1")
	key, ok = dbInstance.getKeyString("test1")
	test.IsEqualString(t, key, "")

	keyInt, ok := dbInstance.getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 0)
	test.IsEqualBool(t, ok, false)
	dbInstance.setKey("test2", 2)
	keyInt, ok = dbInstance.getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 2)
	test.IsEqualBool(t, ok, true)
	dbInstance.setKey("test2", 0)
	keyInt, ok = dbInstance.getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 0)
	test.IsEqualBool(t, ok, true)

	bytes, ok := dbInstance.getKeyBytes("test3")
	test.IsEqualInt(t, len(bytes), 0)
	test.IsEqualBool(t, ok, false)
	dbInstance.setKey("test3", []byte("test"))
	bytes, ok = dbInstance.getKeyBytes("test3")
	test.IsEqualString(t, string(bytes), "test")
	test.IsEqualBool(t, ok, true)
}

func TestExpiration(t *testing.T) {
	dbInstance.setKey("expTest", "test")
	dbInstance.setKey("expTest2", "test2")
	_, ok := dbInstance.getKeyString("expTest")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.getKeyString("expTest2")
	test.IsEqualBool(t, ok, true)
	dbInstance.setExpiryInSeconds("expTest", 1)
	dbInstance.setExpiryAt("expTest2", time.Now().Add(1*time.Second).Unix())
	_, ok = dbInstance.getKeyString("expTest")
	test.IsEqualBool(t, ok, true)
	_, ok = dbInstance.getKeyString("expTest2")
	test.IsEqualBool(t, ok, true)
	mRedis.FastForward(2 * time.Second)
	_, ok = dbInstance.getKeyString("expTest")
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.getKeyString("expTest2")
	test.IsEqualBool(t, ok, false)
}

func TestDeleteAll(t *testing.T) {
	dbInstance.setKey("delTest", "test")
	dbInstance.setKey("delTest2", "test2")
	dbInstance.setKey("delTest3", "test2")

	keys := dbInstance.getAllKeysWithPrefix("delTest")
	test.IsEqualInt(t, len(keys), 3)
	dbInstance.deleteAllWithPrefix("delTest")
	keys = dbInstance.getAllKeysWithPrefix("delTest")
	test.IsEqualInt(t, len(keys), 0)
}

func TestGetAllValuesWithPrefix(t *testing.T) {
	content := make(map[string]string)
	content["alTest"] = "test"
	content["alTest2"] = "test2"
	content["alTest3"] = "test3"
	content["alTest4"] = "test4"
	for k, v := range content {
		dbInstance.setKey(k, v)
	}
	keys := dbInstance.getAllValuesWithPrefix("alTest")
	test.IsEqualInt(t, len(keys), 4)
	for k, v := range keys {
		result, err := redigo.String(v, nil)
		test.IsNil(t, err)
		test.IsEqualString(t, result, content[k])
	}
}

func TestGetHashmap(t *testing.T) {
	hmap, ok := dbInstance.getHashMap("newmap")
	test.IsEqualBool(t, hmap == nil, true)
	test.IsEqualBool(t, ok, false)

	content := make(map[string]string)
	content["alTest1"] = "test"
	content["alTest2"] = "test2"
	content["alTest3"] = "test3"
	content["alTest4"] = "test4"
	dbInstance.setHashMap(dbInstance.buildArgs("newmap").AddFlat(content))
	hmap, ok = dbInstance.getHashMap("newmap")
	test.IsEqualBool(t, ok, true)
	hmapString, err := redigo.StringMap(hmap, nil)
	test.IsNil(t, err)
	for k, v := range content {
		test.IsEqualString(t, hmapString[k], v)
	}

	content2 := make(map[string]string)
	content2["alTest4"] = "test4"
	content2["alTest5"] = "test5"
	content2["alTest6"] = "test6"
	content2["alTest7"] = "test7"
	dbInstance.setHashMap(dbInstance.buildArgs("newmap2").AddFlat(content2))

	maps := dbInstance.getAllHashesWithPrefix("newmap")
	test.IsEqualInt(t, len(maps), 2)
}

func TestApiKeys(t *testing.T) {
	keys := dbInstance.GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 0)
	_, ok := dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, false)

	newKey := models.ApiKey{
		Id:           "newkey",
		FriendlyName: "New Key",
		LastUsed:     1234,
		Permissions:  1,
	}
	dbInstance.SaveApiKey(newKey)
	retrievedKey, ok := dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedKey.Id == newKey.Id, true)
	test.IsEqualBool(t, retrievedKey.FriendlyName == newKey.FriendlyName, true)
	test.IsEqualBool(t, retrievedKey.LastUsed == newKey.LastUsed, true)
	test.IsEqualBool(t, retrievedKey.Permissions == newKey.Permissions, true)

	dbInstance.SaveApiKey(models.ApiKey{
		Id:           "123",
		FriendlyName: "34",
		LastUsed:     0,
		Permissions:  0,
	})

	keys = dbInstance.GetAllApiKeys()
	test.IsEqualInt(t, len(keys), 2)

	dbInstance.DeleteApiKey("newkey")
	_, ok = dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, false)

	newKey.LastUsed = 10
	dbInstance.UpdateTimeApiKey(newKey)
	key, ok := dbInstance.GetApiKey("newkey")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, key.LastUsed == 10, true)

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

	_, ok = dbInstance.GetSystemKey(4)
	test.IsEqualBool(t, ok, false)
	dbInstance.SaveApiKey(models.ApiKey{
		Id:          "sysKey1",
		PublicId:    "publicSysKey1",
		IsSystemKey: true,
		UserId:      5,
		Expiry:      time.Now().Add(time.Hour).Unix(),
	})
	_, ok = dbInstance.GetSystemKey(4)
	test.IsEqualBool(t, ok, false)
	dbInstance.SaveApiKey(models.ApiKey{
		Id:          "sysKey2",
		PublicId:    "publicSysKey2",
		IsSystemKey: true,
		UserId:      4,
		Expiry:      time.Now().Add(-1 * time.Hour).Unix(),
	})
	_, ok = dbInstance.GetSystemKey(4)
	test.IsEqualBool(t, ok, false)
	_, ok = dbInstance.GetSystemKey(5)
	test.IsEqualBool(t, ok, true)
	dbInstance.SaveApiKey(models.ApiKey{
		Id:          "sysKey3",
		PublicId:    "publicSysKey2",
		IsSystemKey: true,
		UserId:      4,
		Expiry:      time.Now().Add(2 * time.Hour).Unix(),
	})
	dbInstance.SaveApiKey(models.ApiKey{
		Id:          "sysKey4",
		PublicId:    "publicSysKey4",
		IsSystemKey: true,
		UserId:      4,
		Expiry:      time.Now().Add(4 * time.Hour).Unix(),
	})
	key, ok = dbInstance.GetSystemKey(4)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, key.Id, "sysKey4")
	test.IsEqualBool(t, key.IsSystemKey, true)
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

func TestE2EConfig(t *testing.T) {
	e2econfig := models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("testnonce"),
		Content:        []byte("testcontent"),
		AvailableFiles: nil,
	}
	dbInstance.SaveEnd2EndInfo(e2econfig, 2)
	retrieved := dbInstance.GetEnd2EndInfo(2)
	test.IsEqualInt(t, retrieved.Version, 1)
	test.IsEqualString(t, string(retrieved.Nonce), "testnonce")
	test.IsEqualString(t, string(retrieved.Content), "testcontent")
	dbInstance.DeleteEnd2EndInfo(2)
	retrieved = dbInstance.GetEnd2EndInfo(2)
	test.IsEqualInt(t, retrieved.Version, 0)
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

func TestGetAllMetaDataIds(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)

	ids := instance.GetAllMetaDataIds()
	test.IsEqualString(t, ids[0], "test2")
	test.IsEqualString(t, ids[1], "test3")

	instance.Close()
	defer test.ExpectPanic(t)
	_ = instance.GetAllMetaDataIds()
}

func TestUsers(t *testing.T) {
	instance, err := New(config)
	test.IsNil(t, err)
	users := instance.GetAllUsers()
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
	instance.SaveUser(user, false)
	retrievedUser, ok := instance.GetUser(2)
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, retrievedUser, user)
	users = instance.GetAllUsers()
	test.IsEqualInt(t, len(users), 1)
	test.IsEqualInt(t, retrievedUser.Id, 2)

	_, ok = instance.GetUser(0)
	test.IsEqualBool(t, ok, false)
	_, ok = instance.GetUserByName("invalid")
	test.IsEqualBool(t, ok, false)
	retrievedUser, ok = instance.GetUserByName("test")
	test.IsEqualBool(t, ok, true)
	test.IsEqual(t, retrievedUser, user)

	instance.DeleteUser(2)
	_, ok = instance.GetUser(2)
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
	instance.SaveUser(user, true)
	_, ok = instance.GetUser(1000)
	test.IsEqualBool(t, ok, false)
	retrievedUser, ok = instance.GetUserByName("test2")
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, retrievedUser.Id == 1000, false)
	user.Id = retrievedUser.Id
	test.IsEqual(t, retrievedUser, user)

	instance.UpdateUserLastOnline(retrievedUser.Id)
	retrievedUser, ok = instance.GetUser(retrievedUser.Id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, time.Now().Unix()-retrievedUser.LastOnline < 5, true)
	test.IsEqualBool(t, time.Now().Unix()-retrievedUser.LastOnline > -1, true)

	user.Name = "test1"
	instance.SaveUser(user, true)
	user.Name = "test3"
	instance.SaveUser(user, true)
	user.Name = "test99"
	user.UserLevel = models.UserLevelSuperAdmin
	instance.SaveUser(user, true)
	user.Name = "test0"
	user.UserLevel = models.UserLevelUser
	instance.SaveUser(user, true)

	users = instance.GetAllUsers()
	test.IsEqualInt(t, len(users), 5)
	test.IsEqualString(t, users[0].Name, "test99")
	test.IsEqualString(t, users[1].Name, "test2")
	test.IsEqualString(t, users[2].Name, "test1")
	test.IsEqualString(t, users[3].Name, "test3")
	test.IsEqualString(t, users[4].Name, "test0")

	_, err = dbToUser([]any{"invalid"})
	test.IsNotNil(t, err)
	defer instance.Close()
}
