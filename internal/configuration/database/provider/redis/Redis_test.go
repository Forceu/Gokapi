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

func TestDatabaseProvider_Upgrade(t *testing.T) {
	var err error
	dbInstance, err = New(config)
	test.IsNil(t, err)
	dbInstance.Upgrade(19)
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
}

func TestE2EConfig(t *testing.T) {
	e2econfig := models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("testnonce"),
		Content:        []byte("testcontent"),
		AvailableFiles: nil,
	}
	dbInstance.SaveEnd2EndInfo(e2econfig)
	retrieved := dbInstance.GetEnd2EndInfo()
	test.IsEqualInt(t, retrieved.Version, 1)
	test.IsEqualString(t, string(retrieved.Nonce), "testnonce")
	test.IsEqualString(t, string(retrieved.Content), "testcontent")
	dbInstance.DeleteEnd2EndInfo()
	retrieved = dbInstance.GetEnd2EndInfo()
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

func TestUploadStatus(t *testing.T) {
	allStatus := dbInstance.GetAllUploadStatus()
	test.IsEqualInt(t, len(allStatus), 0)
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
	test.IsEqualInt(t, len(allStatus), 1)
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
