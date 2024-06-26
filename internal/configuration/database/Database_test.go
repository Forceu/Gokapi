package database

import (
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var configSqlite = models.DbConnection{
	SqliteDataDir:  "./test/",
	SqliteFileName: "gokapi.sqlite",
	Type:           0, // dbabstraction.TypeSqlite
}

var configRedis = models.DbConnection{
	RedisPrefix: "test_",
	RedisUrl:    "127.0.0.1:26379",
	Type:        1, // dbabstraction.TypeRedis
}

var mRedis *miniredis.Miniredis

var availableDatabases []dbabstraction.Database

func TestMain(m *testing.M) {

	mRedis = miniredis.NewMiniRedis()
	err := mRedis.StartAddr("127.0.0.1:26379")
	if err != nil {
		log.Fatal("Could not start miniredis")
	}
	exitVal := m.Run()
	mRedis.Close()
	os.RemoveAll("./test/")
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	availableDatabases = make([]dbabstraction.Database, 0)
	Init(configRedis)
	availableDatabases = append(availableDatabases, db)
	Init(configSqlite)
	availableDatabases = append(availableDatabases, db)
	defer test.ExpectPanic(t)
	Init(models.DbConnection{Type: 2})
}

func TestApiKeys(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return GetAllApiKeys() }, map[string]models.ApiKey{})
	newApiKey := models.ApiKey{
		Id:           "test",
		FriendlyName: "testKey",
		LastUsed:     1000,
		Permissions:  10,
	}
	runAllTypesNoOutput(t, func() { SaveApiKey(newApiKey) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetApiKey("test")
	}, newApiKey, true)
	newApiKey.LastUsed = 2000
	runAllTypesNoOutput(t, func() { UpdateTimeApiKey(newApiKey) })
	runAllTypesCompareOutput(t, func() any { return GetAllApiKeys() }, map[string]models.ApiKey{"test": newApiKey})
	runAllTypesNoOutput(t, func() { DeleteApiKey("test") })
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetApiKey("test")
	}, models.ApiKey{}, false)
}

func TestE2E(t *testing.T) {
	input := models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("test"),
		Content:        []byte("test2"),
		AvailableFiles: []string{"should", "not", "be", "saved"},
	}
	runAllTypesNoOutput(t, func() { SaveEnd2EndInfo(input) })
	input.AvailableFiles = []string{}
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo() }, input)
	runAllTypesNoOutput(t, func() { DeleteEnd2EndInfo() })
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo() }, models.E2EInfoEncrypted{AvailableFiles: []string{}})
}

func TestSessions(t *testing.T) {
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
	input := models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(20 * time.Second).Unix(),
	}
	runAllTypesNoOutput(t, func() { SaveSession("newsession", input) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, input, true)
	runAllTypesNoOutput(t, func() { DeleteSession("newsession") })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
	runAllTypesNoOutput(t, func() { SaveSession("newsession", input) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, input, true)
	runAllTypesNoOutput(t, func() { DeleteAllSessions() })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
}

func TestUploadDefaults(t *testing.T) {
	defaultValues := models.LastUploadValues{
		Downloads:         1,
		TimeExpiry:        14,
		Password:          "",
		UnlimitedDownload: false,
		UnlimitedTime:     false,
	}
	runAllTypesCompareOutput(t, func() any { return GetUploadDefaults() }, defaultValues)
	newValues := models.LastUploadValues{
		Downloads:         5,
		TimeExpiry:        20,
		Password:          "123",
		UnlimitedDownload: true,
		UnlimitedTime:     true,
	}
	runAllTypesNoOutput(t, func() { SaveUploadDefaults(newValues) })
	runAllTypesCompareOutput(t, func() any { return GetUploadDefaults() }, newValues)
}

func TestUploadStatus(t *testing.T) {
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetUploadStatus("newstatus") }, models.UploadStatus{}, false)
	runAllTypesCompareOutput(t, func() any { return GetAllUploadStatus() }, []models.UploadStatus{})
	newStatus := models.UploadStatus{
		ChunkId:       "newstatus",
		CurrentStatus: 1,
	}
	runAllTypesNoOutput(t, func() { SaveUploadStatus(newStatus) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetUploadStatus("newstatus") }, newStatus, true)
	runAllTypesCompareOutput(t, func() any { return GetAllUploadStatus() }, []models.UploadStatus{newStatus})
}

func TestHotlinks(t *testing.T) {
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetHotlink("newhotlink") }, "", false)
	newFile := models.File{Id: "testfile",
		HotlinkId: "newhotlink"}
	runAllTypesNoOutput(t, func() { SaveHotlink(newFile) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetHotlink("newhotlink") }, "testfile", true)
	runAllTypesCompareOutput(t, func() any { return GetAllHotlinks() }, []string{"newhotlink"})
	runAllTypesNoOutput(t, func() { DeleteHotlink("newhotlink") })
	runAllTypesCompareOutput(t, func() any { return GetAllHotlinks() }, []string{})
}

func TestMetaData(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{})
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, models.File{}, false)
	file := models.File{
		Id:                 "testid",
		Name:               "Testname",
		Size:               "3Kb",
		SHA1:               "12345556",
		PasswordHash:       "sfffwefwe",
		HotlinkId:          "hotlink",
		ContentType:        "none",
		AwsBucket:          "aws1",
		ExpireAtString:     "In 10 seconds",
		ExpireAt:           time.Now().Add(10 * time.Second).Unix(),
		SizeBytes:          3 * 1024,
		DownloadsRemaining: 2,
		DownloadCount:      5,
		Encryption: models.EncryptionInfo{
			IsEncrypted:         true,
			IsEndToEndEncrypted: true,
			DecryptionKey:       []byte("dekey"),
			Nonce:               []byte("nonce"),
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	runAllTypesNoOutput(t, func() { SaveMetaData(file) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, file, true)
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{"testid"})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{"testid": file})
	runAllTypesNoOutput(t, func() { DeleteMetaData("testid") })
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{})
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, models.File{}, false)
}

func TestUpgrade(t *testing.T) {
	runAllTypesNoOutput(t, func() { Upgrade(19) })
}

func TestRunGarbageCollection(t *testing.T) {
	runAllTypesNoOutput(t, func() { RunGarbageCollection() })
}

func TestClose(t *testing.T) {
	runAllTypesNoOutput(t, func() { Close() })
}

func runAllTypesNoOutput(t *testing.T, functionToRun func()) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		functionToRun()
	}
}

func runAllTypesCompareOutput(t *testing.T, functionToRun func() any, expectedOutput any) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		output := functionToRun()
		isEqual(t, output, expectedOutput)
	}
}
func runAllTypesCompareTwoOutputs(t *testing.T, functionToRun func() (any, any), expectedOutput1, expectedOutput2 any) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		output1, output2 := functionToRun()
		isEqual(t, output1, expectedOutput1)
		isEqual(t, output2, expectedOutput2)
	}
}

func isEqual(t *testing.T, v1, v2 any) {
	t.Helper()
	if !reflect.DeepEqual(v1, v2) {
		fmt.Println("Values are not as expected: ")
		fmt.Printf("%+v\n", v1)
		fmt.Printf("%+v\n", v2)
		t.Fatal("Unexpected value")
	}
}
