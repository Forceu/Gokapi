package redis

import (
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strconv"
)

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

func getAllStringWithPrefix(prefix string) map[string]string {
	var result = make(map[string]string)
	cursor := 0

	for {
		values, err := redigo.Values(redisConnection.Do("SCAN", cursor, "MATCH", dbPrefix+prefix+"*", "COUNT", 100))
		if err != nil {
			helper.Check(err)
			return nil // Return nil or handle error appropriately
		}

		// Get the new cursor and the keys from the response
		cursor, _ = redigo.Int(values[0], nil)
		keys, _ := redigo.Strings(values[1], nil)

		// Retrieve the value for each key and store in the result map
		for _, key := range keys {
			value, err := redigo.String(redisConnection.Do("GET", key))
			if err != nil {
				helper.Check(err)
				continue // Skip keys that cannot be retrieved
			}
			result[key] = value
		}

		// If cursor is 0, the iteration is complete
		if cursor == 0 {
			break
		}
	}
	return result
}

type redisHash struct {
	Key  string
	Hash map[string]string
}

// Function to get all hashmaps with a given prefix
func getAllHashesWithPrefix(prefix string) []redisHash {
	var result []redisHash
	cursor := 0
	for {
		// Use SCAN to get keys matching the prefix
		values, err := redigo.Values(redisConnection.Do("SCAN", cursor, "MATCH", dbPrefix+prefix+"*", "COUNT", 100))
		helper.Check(err)

		// Get the new cursor and the keys from the response
		cursor, _ = redigo.Int(values[0], nil)
		keys, _ := redigo.Strings(values[1], nil)

		// Iterate through the keys and get the hashmaps
		for _, key := range keys {
			// Use HGETALL to get the hashmap stored at the key
			hashValues, err := redigo.Values(redisConnection.Do("HGETALL", key))
			helper.Check(err)

			// Convert the returned values to a map[string]string
			hashMap := make(map[string]string)
			for i := 0; i < len(hashValues); i += 2 {
				field := string(hashValues[i].([]byte))
				value := string(hashValues[i+1].([]byte))
				hashMap[field] = value
			}
			result = append(result, redisHash{
				Key:  key,
				Hash: hashMap,
			})
		}

		// If cursor is 0, the iteration is complete
		if cursor == 0 {
			break
		}
	}
	return result
}

func getKeyString(id string) (string, bool) {
	result, err := redigo.String(redisConnection.Do("GET", dbPrefix+id))
	helper.Check(err)
	if result == "" {
		return "", false
	}
	return result, true
}

func setKeyString(id, content string) {
	_, err := redisConnection.Do("SET", dbPrefix+id, content)
	helper.Check(err)
}

func getKeyInt(id string) (int, bool) {
	result, err := redisConnection.Do("GET", dbPrefix+id)
	if errors.Is(err, redigo.ErrNil) {
		return 0, false
	}
	resultInt, err2 := redigo.Int(result, err)
	helper.Check(err2)
	return resultInt, true
}

func setKeyInt(id string, content int) {
	_, err := redisConnection.Do("SET", dbPrefix+id, content)
	helper.Check(err)
}

func getHashMap(id string) (map[string]string, bool) {
	result, err := redigo.StringMap(redisConnection.Do("HGETALL", dbPrefix+id))
	helper.Check(err)
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func setHashMap(id string, content map[string]string) {
	args := redigo.Args{}.Add(dbPrefix + id)
	for k, v := range content {
		args = args.Add(k, v)
	}

	_, err := redisConnection.Do("HMSET", args...)
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
