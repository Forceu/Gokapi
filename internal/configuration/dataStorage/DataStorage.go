package dataStorage

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"git.mills.io/prologic/bitcask"
	"log"
	"strings"
	"time"
)

const prefixFile = "file:id:"
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
		var bufInt uint32
		err = binary.Read(bytes.NewBuffer(bufByte), binary.LittleEndian, &bufInt)
		helper.Check(err)
		downloads = int(bufInt)
	}
	if database.Has([]byte(idDefaultExpiry)) {
		bufByte, err := database.Get([]byte(idDefaultExpiry))
		helper.Check(err)
		var bufInt uint32
		err = binary.Read(bytes.NewBuffer(bufByte), binary.LittleEndian, &bufInt)
		helper.Check(err)
		expiry = int(bufInt)
	}
	if database.Has([]byte(idDefaultPassword)) {
		buf, err := database.Get([]byte(idDefaultPassword))
		helper.Check(err)
		password = string(buf)
	}
	return downloads, expiry, password
}

func SaveUploadDefaults(downloads, expiry int, password string) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(downloads))
	err := database.Put([]byte(idDefaultDownloads), buf)
	helper.Check(err)

	binary.LittleEndian.PutUint32(buf, uint32(expiry))
	err = database.Put([]byte(idDefaultExpiry), buf)
	helper.Check(err)

	err = database.Put([]byte(idDefaultPassword), []byte(password))
	helper.Check(err)
}

func Close() {
	if database != nil {
		database.Close()
	}
}
