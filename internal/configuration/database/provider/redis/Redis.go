package redis

import (
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	redigo "github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"time"
)

var pool *redigo.Pool
var dbPrefix string

type DatabaseProvider struct {
}

// New returns an instance
func New() DatabaseProvider {
	return DatabaseProvider{}
}

// GetType returns 1, for being a Redis interface
func (p DatabaseProvider) GetType() int {
	return 1 // dbabstraction.Redis
}

// Init connects to the database and creates the table structure, if necessary
func (p DatabaseProvider) Init(config models.DbConnection) error {
	dbPrefix = config.RedisPrefix
	pool = newPool(config)
	conn := pool.Get()
	defer conn.Close()
	_, err := redigo.String(conn.Do("PING"))
	return err
}

func getDialOptions(config models.DbConnection) []redigo.DialOption {
	dialOptions := []redigo.DialOption{redigo.DialClientName("gokapi")}
	if config.RedisUsername != "" {
		dialOptions = append(dialOptions, redigo.DialUsername(config.RedisUsername))
	}
	if config.RedisPassword != "" {
		dialOptions = append(dialOptions, redigo.DialPassword(config.RedisPassword))
	}
	if config.RedisUseSsl {
		dialOptions = append(dialOptions, redigo.DialUseTLS(true))
	}
	return dialOptions
}

func newPool(config models.DbConnection) *redigo.Pool {

	newRedisPool := &redigo.Pool{
		MaxIdle:     10,
		IdleTimeout: 2 * time.Minute,

		Dial: func() (redigo.Conn, error) {
			c, err := redigo.Dial("tcp", config.RedisUrl, getDialOptions(config)...)
			if err != nil {
				fmt.Println("Error connecting to redis")
			}
			helper.Check(err)
			return c, err
		},

		TestOnBorrow: func(c redigo.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return newRedisPool
}

// Upgrade migrates the DB to a new Gokapi version, if required
func (p DatabaseProvider) Upgrade(currentVersion int) {
	// Currently no upgrade necessary
	return
}

// Close the database connection
func (p DatabaseProvider) Close() {
	err := pool.Close()
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
	allKeys := getAllKeysWithPrefix(prefix)
	for _, key := range allKeys {
		value, err := getKeyRaw(key)
		if errors.Is(err, redigo.ErrNil) {
			continue
		}
		helper.Check(err)
		result[key] = value
	}
	return result
}

// Function to get all hashmaps with a given prefix
func getAllHashesWithPrefix(prefix string) map[string][]any {
	result := make(map[string][]any)
	allKeys := getAllKeysWithPrefix(prefix)
	for _, key := range allKeys {
		hashMap, ok := getHashMap(key)
		if !ok {
			continue
		}
		result[key] = hashMap
	}
	return result
}

func getAllKeysWithPrefix(prefix string) []string {
	var result []string
	conn := pool.Get()
	defer conn.Close()
	fullPrefix := dbPrefix + prefix
	cursor := 0
	for {
		reply, err := redigo.Values(conn.Do("SCAN", cursor, "MATCH", fullPrefix+"*", "COUNT", 100))
		helper.Check(err)

		cursor, _ = redigo.Int(reply[0], nil)
		keys, _ := redigo.Strings(reply[1], nil)
		for _, key := range keys {
			result = append(result, strings.Replace(key, dbPrefix, "", 1))
		}

		if cursor == 0 {
			break
		}
	}
	return result
}

func setKey(id string, content any) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", dbPrefix+id, content)
	helper.Check(err)
}

func getKeyRaw(id string) (any, error) {
	conn := pool.Get()
	defer conn.Close()
	return conn.Do("GET", dbPrefix+id)
}

func getKeyString(id string) (string, bool) {
	result, err := redigo.String(getKeyRaw(id))
	if result == "" {
		return "", false
	}
	helper.Check(err)
	return result, true
}

func getKeyInt(id string) (int, bool) {
	result, err := getKeyRaw(id)
	if result == nil {
		return 0, false
	}
	resultInt, err2 := redigo.Int(result, err)
	helper.Check(err2)
	return resultInt, true
}
func getKeyBytes(id string) ([]byte, bool) {
	result, err := getKeyRaw(id)
	if result == nil {
		return nil, false
	}
	resultInt, err2 := redigo.Bytes(result, err)
	helper.Check(err2)
	return resultInt, true
}

func getHashMap(id string) ([]any, bool) {
	conn := pool.Get()
	defer conn.Close()
	result, err := redigo.Values(conn.Do("HGETALL", dbPrefix+id))
	helper.Check(err)
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func buildArgs(id string) redigo.Args {
	return redigo.Args{}.Add(dbPrefix + id)
}

func setHashMap(content redigo.Args) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", content...)
	helper.Check(err)
}

func setExpiryAt(id string, expiry int64) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("EXPIREAT", dbPrefix+id, strconv.FormatInt(expiry, 10))
	helper.Check(err)
}
func setExpiryInSeconds(id string, expiry int64) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("EXPIRE", dbPrefix+id, strconv.FormatInt(expiry, 10))
	helper.Check(err)
}

func deleteKey(id string) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", dbPrefix+id)
	helper.Check(err)
}

func runEval(cmd string) {
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("EVAL", cmd, "0")
	helper.Check(err)
}

func deleteAllWithPrefix(prefix string) {
	runEval("for _,k in ipairs(redis.call('keys','" + dbPrefix + prefix + "*')) do redis.call('del',k) end")
}
