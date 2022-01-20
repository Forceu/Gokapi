package dataStorage

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"git.mills.io/prologic/bitcask"
	"log"
	"strings"
	"time"
)

const prefixApiKey = "apikey:id:"
const prefixDownloadStatus = "download:id:"
const prefixFile = "file:id:"
const prefixHotlink = "hotlink:id:"
const prefixSessions = "session:id:"
const idDefaultDownloads = "default:downloads"
const idDefaultExpiry = "default:expiry"
const idDefaultPassword = "default:password"

var database *bitcask.Bitcask

func Init(dbPath string) {
	db, err := bitcask.Open(dbPath, bitcask.WithMaxKeySize(256))
	if err != nil {
		log.Fatal(err)
	}
	database = db
}

func GetDownloadStatus(id string) (string, bool) {
	value, ok := getValue(prefixDownloadStatus + id)
	if !ok {
		return "", false
	}
	return string(value), true
}

func SaveDownloadStatus(id, fileId string) {
	err := database.PutWithTTL([]byte(prefixDownloadStatus+id), []byte(fileId), 24*time.Hour)
	helper.Check(err)
}

func DeleteDownloadStatus(id string) {
	deleteKey(prefixDownloadStatus + id)
}

func GetAllDownloadStatus() map[string]string {
	result := make(map[string]string)
	err := database.Scan([]byte(prefixDownloadStatus), func(key []byte) error {
		downloadId := strings.Replace(string(key), prefixDownloadStatus, "", 1)
		fileId, ok := GetDownloadStatus(downloadId)
		if !ok {
			return errors.New("getall: key does not exist")
		}
		result[downloadId] = fileId
		return nil
	})
	helper.Check(err)
	return result
}

func GetHotlink(id string) (string, bool) {
	value, ok := getValue(prefixHotlink + id)
	if !ok {
		return "", false
	}
	return string(value), true
}

func SaveHotlink(id string, file models.File) {
	err := database.PutWithTTL([]byte(prefixHotlink+id), []byte(file.Id), expiryToDuration(file))
	helper.Check(err)
}

func DeleteHotlink(id string) {
	deleteKey(prefixHotlink + id)
}

func GetAllApiKeys() map[string]models.ApiKey {
	result := make(map[string]models.ApiKey)
	err := database.Scan([]byte(prefixApiKey), func(key []byte) error {
		apikeyID := strings.Replace(string(key), prefixApiKey, "", 1)
		apiKey, ok := GetApiKey(apikeyID)
		if !ok {
			return errors.New("getall: key does not exist")
		}
		result[apiKey.Id] = apiKey
		return nil
	})
	helper.Check(err)
	return result
}

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

func SaveApiKey(apikey models.ApiKey, updateTimeOnly bool) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(apikey)
	helper.Check(err)
	err = database.Put([]byte(prefixApiKey+apikey.Id), buf.Bytes())
	helper.Check(err)
	if !updateTimeOnly {
		err = database.Sync()
		helper.Check(err)
	}
}

func DeleteApiKey(id string) {
	deleteKey(prefixApiKey + id)
}

func expiryToDuration(file models.File) time.Duration {
	return time.Until(time.Unix(file.ExpireAt, 0))
}

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

func DeleteSession(id string) {
	deleteKey(prefixSessions + id)
}

func SaveSession(id string, session models.Session, expiry time.Duration) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(session)
	helper.Check(err)
	err = database.PutWithTTL([]byte(prefixSessions+id), buf.Bytes(), expiry)
	helper.Check(err)
	err = database.Sync()
	helper.Check(err)
}

func GetAllFiles() map[string]models.File {
	result := make(map[string]models.File)
	err := database.Scan([]byte(prefixFile), func(key []byte) error {
		fileId := strings.Replace(string(key), prefixFile, "", 1)
		file, ok := GetMetaDataById(fileId)
		if !ok {
			return errors.New("getall: key does not exist")
		}
		result[file.Id] = file
		return nil
	})
	helper.Check(err)
	return result
}

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

func SaveMetaData(file models.File) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(file)
	helper.Check(err)
	err = database.Put([]byte(prefixFile+file.Id), buf.Bytes())
	helper.Check(err)
	err = database.Sync()
	helper.Check(err)
}

func DeleteMetaData(id string) {
	deleteKey(prefixFile + id)
}

func GetUploadDefaults() (int, int, string) {
	downloads := 1
	expiry := 14
	password := ""
	if database.Has([]byte(idDefaultDownloads)) {
		bufByte, err := database.Get([]byte(idDefaultDownloads))
		helper.Check(err)
		downloads = byteToInt(bufByte)
	}
	if database.Has([]byte(idDefaultExpiry)) {
		bufByte, err := database.Get([]byte(idDefaultExpiry))
		helper.Check(err)
		expiry = byteToInt(bufByte)
	}
	if database.Has([]byte(idDefaultPassword)) {
		buf, err := database.Get([]byte(idDefaultPassword))
		helper.Check(err)
		password = string(buf)
	}
	return downloads, expiry, password
}

func SaveUploadDefaults(downloads, expiry int, password string) {
	err := database.Put([]byte(idDefaultDownloads), intToByte(downloads))
	helper.Check(err)
	err = database.Put([]byte(idDefaultExpiry), intToByte(expiry))
	helper.Check(err)
	err = database.Put([]byte(idDefaultPassword), []byte(password))
	helper.Check(err)
	err = database.Sync()
	helper.Check(err)
}

func Close() {
	if database != nil {
		err := database.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func intToByte(integer int) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, uint32(integer))
	return result
}

func byteToInt(intByte []byte) int {
	var result uint32
	err := binary.Read(bytes.NewBuffer(intByte), binary.LittleEndian, &result)
	helper.Check(err)
	return int(result)
}

func deleteKey(id string) {
	if !database.Has([]byte(id)) {
		return
	}
	err := database.Delete([]byte(id))
	helper.Check(err)
	err = database.Sync()
	helper.Check(err)
}

func getValue(id string) ([]byte, bool) {
	value, err := database.Get([]byte(id))
	if err == nil {
		return value, true
	}
	if err == bitcask.ErrEmptyKey || err == bitcask.ErrKeyExpired || err == bitcask.ErrKeyNotFound {
		return nil, false
	}
	panic(err)
}
