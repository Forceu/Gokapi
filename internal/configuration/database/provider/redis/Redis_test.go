package redis

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	redigo "github.com/gomodule/redigo/redis"
	"log"
	"os"
	"testing"
	"time"
)

var config = models.DbConnection{
	RedisPrefix: "test_",
	RedisUrl:    "127.0.0.1:16379",
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
	dbInstance = New()
	err := dbInstance.Init(config)
	test.IsNil(t, err)
	dbInstance.Close()
	defer test.ExpectPanic(t)
	err = dbInstance.Init(models.DbConnection{
		RedisPrefix: "test_",
		RedisUrl:    "invalid:11",
		Type:        1, // dbabstraction.TypeRedis
	})
	test.IsNotNil(t, err)
}

func TestDatabaseProvider_GetType(t *testing.T) {
	test.IsEqualInt(t, dbInstance.GetType(), 1)
}

func TestDatabaseProvider_Upgrade(t *testing.T) {
	dbInstance = New()
	err := dbInstance.Init(config)
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
	newConfig.RedisUsername = "123"
	newConfig.RedisPassword = "456"
	newConfig.RedisUseSsl = true
	result = getDialOptions(newConfig)
	test.IsEqualInt(t, len(result), 4)
}

func TestGetKey(t *testing.T) {
	key, ok := getKeyString("test1")
	test.IsEqualString(t, key, "")
	test.IsEqualBool(t, ok, false)
	setKey("test1", "content")
	key, ok = getKeyString("test1")
	test.IsEqualString(t, key, "content")
	test.IsEqualBool(t, ok, true)
	deleteKey("test1")
	key, ok = getKeyString("test1")
	test.IsEqualString(t, key, "")

	keyInt, ok := getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 0)
	test.IsEqualBool(t, ok, false)
	setKey("test2", 2)
	keyInt, ok = getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 2)
	test.IsEqualBool(t, ok, true)
	setKey("test2", 0)
	keyInt, ok = getKeyInt("test2")
	test.IsEqualInt(t, keyInt, 0)
	test.IsEqualBool(t, ok, true)

	bytes, ok := getKeyBytes("test3")
	test.IsEqualInt(t, len(bytes), 0)
	test.IsEqualBool(t, ok, false)
	setKey("test3", []byte("test"))
	bytes, ok = getKeyBytes("test3")
	test.IsEqualString(t, string(bytes), "test")
	test.IsEqualBool(t, ok, true)
}

func TestExpiration(t *testing.T) {
	setKey("expTest", "test")
	setKey("expTest2", "test2")
	_, ok := getKeyString("expTest")
	test.IsEqualBool(t, ok, true)
	_, ok = getKeyString("expTest2")
	test.IsEqualBool(t, ok, true)
	setExpiryInSeconds("expTest", 1)
	setExpiryAt("expTest2", time.Now().Add(1*time.Second).Unix())
	_, ok = getKeyString("expTest")
	test.IsEqualBool(t, ok, true)
	_, ok = getKeyString("expTest2")
	test.IsEqualBool(t, ok, true)
	mRedis.FastForward(2 * time.Second)
	_, ok = getKeyString("expTest")
	test.IsEqualBool(t, ok, false)
	_, ok = getKeyString("expTest2")
	test.IsEqualBool(t, ok, false)
}

func TestDeleteAll(t *testing.T) {
	setKey("delTest", "test")
	setKey("delTest2", "test2")
	setKey("delTest3", "test2")

	keys := getAllKeysWithPrefix("delTest")
	test.IsEqualInt(t, len(keys), 3)
	deleteAllWithPrefix("delTest")
	keys = getAllKeysWithPrefix("delTest")
	test.IsEqualInt(t, len(keys), 0)
}

func TestGetAllValuesWithPrefix(t *testing.T) {
	content := make(map[string]string)
	content["alTest"] = "test"
	content["alTest2"] = "test2"
	content["alTest3"] = "test3"
	content["alTest4"] = "test4"
	for k, v := range content {
		setKey(k, v)
	}
	keys := getAllValuesWithPrefix("alTest")
	test.IsEqualInt(t, len(keys), 4)
	for k, v := range keys {
		result, err := redigo.String(v, nil)
		test.IsNil(t, err)
		test.IsEqualString(t, result, content[k])
	}
}

func TestGetHashmap(t *testing.T) {
	hmap, ok := getHashMap("newmap")
	test.IsEqualBool(t, hmap == nil, true)
	test.IsEqualBool(t, ok, false)

	content := make(map[string]string)
	content["alTest"] = "test"
	content["alTest2"] = "test2"
	content["alTest3"] = "test3"
	content["alTest4"] = "test4"
	setHashMap(buildArgs("newmap").AddFlat(content))
	hmap, ok = getHashMap("newmap")
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
	setHashMap(buildArgs("newmap2").AddFlat(content2))

	maps := getAllHashesWithPrefix("newmap")
	test.IsEqualInt(t, len(maps), 2)

}
