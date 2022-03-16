package database

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"git.mills.io/prologic/bitcask"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"log"
	"strings"
	"time"
)

const prefixApiKey = "apikey:id:"
const prefixFile = "file:id:"
const prefixHotlink = "hotlink:id:"
const prefixSessions = "session:id:"
const idLastUploadConfig = "default:lastupload"

const maxKeySize = 96

var bitcaskDb *bitcask.Bitcask

// Init creates the database files and connects to it
func Init(dbPath string) {
	if bitcaskDb == nil {
		db, err := bitcask.Open(dbPath, bitcask.WithMaxKeySize(maxKeySize))
		if err != nil {
			log.Fatal(err)
		}
		bitcaskDb = db
	}
}

// GetLengthAvailable returns the maximum length for a key name
func GetLengthAvailable() int {
	maxLength := 0
	for _, key := range []string{prefixApiKey, prefixFile, prefixHotlink, prefixSessions} {
		length := len(key)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxKeySize - maxLength
}

// Close syncs the database to the filesystem and closes it
func Close() {
	if bitcaskDb != nil {
		err := bitcaskDb.Sync()
		if err != nil {
			fmt.Println(err)
		}
		err = bitcaskDb.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	bitcaskDb = nil
}

// ## File Metadata ##

// GetAllMetadata returns a map of all available files
func GetAllMetadata() map[string]models.File {
	if bitcaskDb == nil {
		panic("Database not loaded!")
	}
	result := make(map[string]models.File)
	var keys []string
	err := bitcaskDb.Scan([]byte(prefixFile), func(key []byte) error {
		fileId := strings.Replace(string(key), prefixFile, "", 1)
		keys = append(keys, fileId)
		return nil
	})

	helper.Check(err)

	for _, key := range keys {
		file, ok := GetMetaDataById(key)
		if ok {
			result[file.Id] = file
		}
	}

	return result
}

// GetMetaDataById returns a models.File,true from the ID passed or false if the id is not valid
func GetMetaDataById(id string) (models.File, bool) {
	result := models.File{}
	value, ok := getValue(prefixFile + id)
	if !ok {
		return result, false
	}
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&result)
	helper.Check(err)
	return result, true
}

// SaveMetaData stores the metadata of a file to the disk
func SaveMetaData(file models.File) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(file)
	helper.Check(err)
	err = bitcaskDb.Put([]byte(prefixFile+file.Id), buf.Bytes())
	helper.Check(err)
	err = bitcaskDb.Sync()
	helper.Check(err)
}

// DeleteMetaData deletes information about a file
func DeleteMetaData(id string) {
	deleteKey(prefixFile + id)
}

// ## Hotlinks ##

// GetHotlink returns the id of the file associated or false if not found
func GetHotlink(id string) (string, bool) {
	value, ok := getValue(prefixHotlink + id)
	if !ok {
		return "", false
	}
	return string(value), true
}

// SaveHotlink stores the hotlink associated with the file in the bitcaskDb
func SaveHotlink(file models.File) {
	err := bitcaskDb.PutWithTTL([]byte(prefixHotlink+file.HotlinkId), []byte(file.Id), expiryToDuration(file))
	helper.Check(err)
	err = bitcaskDb.Sync()
	helper.Check(err)
}

// DeleteHotlink deletes a hotlink with the given ID
func DeleteHotlink(id string) {
	deleteKey(prefixHotlink + id)
}

// ## API Keys ##

// GetAllApiKeys returns a map with all API keys
func GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)
	var keys []string
	err := bitcaskDb.Scan([]byte(prefixApiKey), func(key []byte) error {
		apikeyID := strings.Replace(string(key), prefixApiKey, "", 1)
		keys = append(keys, apikeyID)
		return nil
	})
	helper.Check(err)

	for _, key := range keys {
		apiKey, ok := GetApiKey(key)
		if ok {
			result[apiKey.Id] = apiKey
		}
	}
	return result
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func GetApiKey(id string) (models.ApiKey, bool) {
	result := models.ApiKey{}
	value, ok := getValue(prefixApiKey + id)
	if !ok {
		return result, false
	}
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&result)
	helper.Check(err)
	return result, true
}

// SaveApiKey saves the API key to the database. If updateTimeOnly is true, the database might not be synced afterwards
func SaveApiKey(apikey models.ApiKey, updateTimeOnly bool) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(apikey)
	helper.Check(err)
	err = bitcaskDb.Put([]byte(prefixApiKey+apikey.Id), buf.Bytes())
	helper.Check(err)
	if !updateTimeOnly {
		err = bitcaskDb.Sync()
		helper.Check(err)
	}
}

// DeleteApiKey deletes an API key with the given ID
func DeleteApiKey(id string) {
	deleteKey(prefixApiKey + id)
}

// ## Sessions ##

// GetSession returns the session with the given ID or false if not a valid ID
func GetSession(id string) (models.Session, bool) {
	result := models.Session{}
	value, ok := getValue(prefixSessions + id)
	if !ok {
		return result, false
	}
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&result)
	helper.Check(err)
	return result, true
}

// DeleteSession deletes a session with the given ID
func DeleteSession(id string) {
	deleteKey(prefixSessions + id)
}

// DeleteAllSessions logs all users out
func DeleteAllSessions() {
	err := bitcaskDb.SiftScan([]byte(prefixSessions), func(key []byte) (bool, error) {
		return true, nil
	})
	helper.Check(err)
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func SaveSession(id string, session models.Session, expiry time.Duration) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(session)
	helper.Check(err)
	err = bitcaskDb.PutWithTTL([]byte(prefixSessions+id), buf.Bytes(), expiry)
	helper.Check(err)
	err = bitcaskDb.Sync()
	helper.Check(err)
}

// ## Upload Defaults ##

// GetUploadDefaults returns the last used setting for amount of downloads allowed, last expiry in days and
// a password for the file
func GetUploadDefaults() models.LastUploadValues {
	defaultValues := models.LastUploadValues{
		Downloads:         1,
		TimeExpiry:        14,
		Password:          "",
		UnlimitedDownload: false,
		UnlimitedTime:     false,
	}
	result := models.LastUploadValues{}
	if bitcaskDb.Has([]byte(idLastUploadConfig)) {
		value, err := bitcaskDb.Get([]byte(idLastUploadConfig))
		helper.Check(err)
		buf := bytes.NewBuffer(value)
		dec := gob.NewDecoder(buf)
		err = dec.Decode(&result)
		helper.Check(err)
		return result
	}
	return defaultValues
}

// SaveUploadDefaults saves the last used setting for an upload
func SaveUploadDefaults(values models.LastUploadValues) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(values)
	helper.Check(err)
	err = bitcaskDb.Put([]byte(idLastUploadConfig), buf.Bytes())
	helper.Check(err)
}

// RunGarbageCollection runs the databases GC
func RunGarbageCollection() {
	err := bitcaskDb.RunGC()
	helper.Check(err)
}

func intToByte(integer int) []byte {
	buf := make([]byte, binary.MaxVarintLen32)
	n := binary.PutVarint(buf, int64(integer))
	return buf[:n]
}

func byteToInt(intByte []byte) int {
	integer, _ := binary.Varint(intByte)
	return int(integer)
}

func deleteKey(id string) {
	if !bitcaskDb.Has([]byte(id)) {
		return
	}
	err := bitcaskDb.Delete([]byte(id))
	helper.Check(err)
	err = bitcaskDb.Sync()
	helper.Check(err)
}

func getValue(id string) ([]byte, bool) {
	value, err := bitcaskDb.Get([]byte(id))
	if err == nil {
		return value, true
	}
	if err == bitcask.ErrEmptyKey || err == bitcask.ErrKeyExpired || err == bitcask.ErrKeyNotFound {
		return nil, false
	}
	panic(err)
}

func expiryToDuration(file models.File) time.Duration {
	return time.Until(time.Unix(file.ExpireAt, 0))
}
