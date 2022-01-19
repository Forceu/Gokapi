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

const prefixFile = "file:id:"
const prefixSessions = "session:id:"
const idDefaultDownloads = "default:downloads"
const idDefaultExpiry = "default:expiry"
const idDefaultPassword = "default:password"

var database *bitcask.Bitcask

func Init(dbPath string) {
	db, err := bitcask.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	database = db
}

func GetSession(id string) (models.Session, bool) {
	result := models.Session{}
	if !database.Has([]byte(prefixSessions + id)) {
		return result, false
	}
	value, err := database.Get([]byte(prefixSessions + id))
	helper.Check(err)
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&result)
	helper.Check(err)
	return result, true
}

func DeleteSession(id string) {
	if !database.Has([]byte(prefixSessions + id)) {
		return
	}
	err := database.Delete([]byte(prefixSessions + id))
	helper.Check(err)
}

func SaveSession(id string, session models.Session, expiry time.Duration) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(session)
	helper.Check(err)
	err = database.PutWithTTL([]byte(prefixSessions+id), buf.Bytes(), expiry)
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
	if database.Has([]byte(prefixFile + id)) {
		return result, false
	}
	value, err := database.Get([]byte(prefixFile + id))
	helper.Check(err)
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&result)
	helper.Check(err)
	return result, true
}

func SaveMetaData(file models.File, expiry time.Duration) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(file)
	helper.Check(err)
	err = database.PutWithTTL([]byte(prefixFile+file.Id), buf.Bytes(), expiry)
	helper.Check(err)
}

func DeleteMetaData(file models.File) {
	if !database.Has([]byte(prefixFile + file.Id)) {
		return
	}
	err := database.Delete([]byte(prefixFile + file.Id))
	helper.Check(err)
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
