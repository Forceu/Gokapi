package redis

import (
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
)

// TODO use pools instead

var redisConnection redigo.Conn
var dbPrefix string

type DatabaseProvider struct {
}

// New returns an instance
func New() DatabaseProvider {
	return DatabaseProvider{}
}

// Init connects to the database and creates the table structure, if necessary
func (p DatabaseProvider) Init(config models.DbConnection) error {
	var err error
	redisConnection, err = redigo.Dial("tcp", config.RedisUrl)
	dbPrefix = config.RedisPrefix
	return err
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentVersion int) {
	// Currently no upgrade necessary
	return
}

// Close the database connection
func (p DatabaseProvider) Close() {
	err := redisConnection.Close()
	if err != nil {
		fmt.Println(err)
	}
}

// RunGarbageCollection runs the databases GC
func (p DatabaseProvider) RunGarbageCollection() {
	// No cleanup required
}

// Function to get all hashmaps with a given prefix
func getAllValuesWithPrefix(prefix string) map[string]any {
	result := make(map[string]any)
	fullPrefix := dbPrefix + prefix
	cursor := 0
	for {
		// Use SCAN to get keys matching the prefix
		values, err := redigo.Values(redisConnection.Do("SCAN", cursor, "MATCH", fullPrefix+"*", "COUNT", 100))
		helper.Check(err)

		// Get the new cursor and the keys from the response
		cursor, _ = redigo.Int(values[0], nil)
		keys, _ := redigo.Strings(values[1], nil)

		for _, key := range keys {
			content, err := redisConnection.Do("GET", key)
			helper.Check(err)
			cleanKey := strings.Replace(key, fullPrefix, "", 1)
			result[cleanKey] = content
		}

		// If cursor is 0, the iteration is complete
		if cursor == 0 {
			break
		}
	}
	return result
}

type redisHash struct {
	Key    string
	Values []any
}

// Function to get all hashmaps with a given prefix
func getAllHashesWithPrefix(prefix string) []redisHash {
	var result []redisHash
	fullPrefix := dbPrefix + prefix
	cursor := 0
	for {
		// Use SCAN to get keys matching the prefix
		values, err := redigo.Values(redisConnection.Do("SCAN", cursor, "MATCH", fullPrefix+"*", "COUNT", 100))
		helper.Check(err)

		// Get the new cursor and the keys from the response
		cursor, _ = redigo.Int(values[0], nil)
		keys, _ := redigo.Strings(values[1], nil)

		for _, key := range keys {
			hashValues, err := redigo.Values(redisConnection.Do("HGETALL", key))
			helper.Check(err)
			result = append(result, redisHash{
				Key:    strings.Replace(key, fullPrefix, "", 1),
				Values: hashValues,
			})
		}

		// If cursor is 0, the iteration is complete
		if cursor == 0 {
			break
		}
	}
	return result
}

func getAllKeynamesWithPrefix(prefix string) []string {
	var keys []string
	cursor := 0
	for {
		reply, err := redigo.Values(redisConnection.Do("SCAN", cursor, "MATCH", dbPrefix+prefix+"*", "COUNT", 100))
		helper.Check(err)

		cursor, _ = redigo.Int(reply[0], nil)
		k, _ := redigo.Strings(reply[1], nil)
		keys = append(keys, k...)

		if cursor == 0 {
			break
		}
	}
	return keys
}

func setKey(id string, content any) {
	_, err := redisConnection.Do("SET", dbPrefix+id, content)
	helper.Check(err)
}

func getKeyString(id string) (string, bool) {
	result, err := redigo.String(redisConnection.Do("GET", dbPrefix+id))
	helper.Check(err)
	if result == "" {
		return "", false
	}
	return result, true
}

func getKeyInt(id string) (int, bool) {
	result, err := redisConnection.Do("GET", dbPrefix+id)
	if result == nil {
		return 0, false
	}
	resultInt, err2 := redigo.Int(result, err)
	helper.Check(err2)
	return resultInt, true
}
func getKeyBytes(id string) ([]byte, bool) {
	result, err := redisConnection.Do("GET", dbPrefix+id)
	if result == nil {
		return nil, false
	}
	resultInt, err2 := redigo.Bytes(result, err)
	helper.Check(err2)
	return resultInt, true
}

func getHashMap(id string) ([]any, bool) {
	result, err := redigo.Values(redisConnection.Do("HGETALL", dbPrefix+id))
	helper.Check(err)
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func buildArgs(id string) redigo.Args {
	return redigo.Args{}.Add(dbPrefix + id)
}

func setHashMapArgs(content redigo.Args) {
	_, err := redisConnection.Do("HMSET", content...)
	helper.Check(err)
}

func setExpiryAt(id string, expiry int64) {
	_, err := redisConnection.Do("EXPIREAT", dbPrefix+id, strconv.FormatInt(expiry, 10))
	helper.Check(err)
}
func setExpiryInSeconds(id string, expiry int64) {
	_, err := redisConnection.Do("EXPIRE", dbPrefix+id, strconv.FormatInt(expiry, 10))
	helper.Check(err)
}

func deleteKey(id string) {
	_, err := redisConnection.Do("DEL", dbPrefix+id)
	helper.Check(err)
}

func runEval(cmd string) {
	_, err := redisConnection.Do("EVAL", cmd, "0")
	helper.Check(err)
}

func deleteAllWithPrefix(prefix string) {
	runEval("for _,k in ipairs(redis.call('keys','" + dbPrefix + prefix + "*')) do redis.call('del',k) end")
}
