package database

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"log"
	"os"
	"testing"
	"time"
)

var configSqlite = models.DbConnection{
	HostUrl: "./test/gokapi.sqlite",
	Type:    0, // dbabstraction.TypeSqlite
}

var configRedis = models.DbConnection{
	RedisPrefix: "test_",
	HostUrl:     "127.0.0.1:26379",
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
	Connect(configRedis)
	availableDatabases = append(availableDatabases, db)
	Connect(configSqlite)
	availableDatabases = append(availableDatabases, db)
	defer test.ExpectPanic(t)
	Connect(models.DbConnection{Type: 2})
}

func TestApiKeys(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return GetAllApiKeys() }, map[string]models.ApiKey{})
	newApiKey := models.ApiKey{
		Id:           "test",
		FriendlyName: "testKey",
		PublicId:     "wfwefewwfefwe",
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

	runAllTypesNoOutput(t, func() {
		SaveApiKey(models.ApiKey{
			Id:       "publicTest",
			PublicId: "publicId",
		})
	})
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetApiKey("publicTest")
		return ok
	}, true)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetApiKeyByPublicKey("publicTest")
		return ok
	}, false)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetApiKey("publicId")
		return ok
	}, false)
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetApiKeyByPublicKey("publicId")
	}, "publicTest", true)

	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSystemKey(6)
		return ok
	}, false)

	runAllTypesNoOutput(t, func() {
		SaveApiKey(models.ApiKey{
			Id:          "sysKey1",
			PublicId:    "sysKey1",
			IsSystemKey: true,
			Expiry:      time.Now().Add(1 * time.Hour).Unix(),
			UserId:      6,
		})
		SaveApiKey(models.ApiKey{
			Id:          "sysKey2",
			PublicId:    "sysKey2",
			IsSystemKey: true,
			Expiry:      time.Now().Add(2 * time.Hour).Unix(),
			UserId:      6,
		})
	})
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		key, ok := GetSystemKey(6)
		return key.Id, ok
	}, "sysKey2", true)
}

func TestE2E(t *testing.T) {
	input := models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("test"),
		Content:        []byte("test2"),
		AvailableFiles: []string{"should", "not", "be", "saved"},
	}
	runAllTypesNoOutput(t, func() { SaveEnd2EndInfo(input, 3) })
	input.AvailableFiles = []string{}
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo(3) }, input)
	runAllTypesNoOutput(t, func() { DeleteEnd2EndInfo(3) })
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo(3) }, models.E2EInfoEncrypted{AvailableFiles: []string{}})
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

	runAllTypesNoOutput(t, func() {
		SaveSession("session1", models.Session{
			RenewAt:    2147483645,
			ValidUntil: 2147483645,
			UserId:     20,
		})
		SaveSession("session2", models.Session{
			RenewAt:    2147483645,
			ValidUntil: 2147483645,
			UserId:     20,
		})
		SaveSession("session3", models.Session{
			RenewAt:    2147483645,
			ValidUntil: 2147483645,
			UserId:     40,
		})
	})

	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session1")
		return ok
	}, true)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session2")
		return ok
	}, true)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session3")
		return ok
	}, true)
	runAllTypesNoOutput(t, func() {
		DeleteAllSessionsByUser(20)
	})
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session1")
		return ok
	}, false)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session2")
		return ok
	}, false)
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSession("session3")
		return ok
	}, true)
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

	increasedDownload := file
	increasedDownload.DownloadCount = increasedDownload.DownloadCount + 1

	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		SaveMetaData(file)
		IncreaseDownloadCount(file.Id, false)
		return GetMetaDataById(file.Id)
	}, increasedDownload, true)

	increasedDownload.DownloadCount = increasedDownload.DownloadCount + 1
	increasedDownload.DownloadsRemaining = increasedDownload.DownloadsRemaining - 1

	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		IncreaseDownloadCount(file.Id, true)
		return GetMetaDataById(file.Id)
	}, increasedDownload, true)
	runAllTypesNoOutput(t, func() { DeleteMetaData(file.Id) })
}

func TestUsers(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return len(GetAllUsers()) }, 0)
	user := models.User{
		Id:            1000,
		Name:          "test2",
		Permissions:   models.UserPermissionNone,
		UserLevel:     models.UserLevelAdmin,
		LastOnline:    1338,
		Password:      "1234568",
		ResetPassword: true,
	}
	runAllTypesNoOutput(t, func() { SaveUser(user, true) })
	user.Id = 1
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetUser(1)
	}, user, true)
	runAllTypesCompareOutput(t, func() any { return len(GetAllUsers()) }, 1)
	user.Name = "test3"
	runAllTypesNoOutput(t, func() { SaveUser(user, false) })
	runAllTypesCompareOutput(t, func() any { return len(GetAllUsers()) }, 1)
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetUserByName("test3")
	}, user, true)
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetUserByName("TEST3")
	}, user, true)
	user.Name = "test4"
	runAllTypesNoOutput(t, func() { SaveUser(user, true) })
	var allUsersSqlite []models.User
	var allUsersRedis []models.User
	runAllTypesCompareOutput(t, func() any {
		allUsers := GetAllUsers()
		switch db.GetType() {
		case dbabstraction.TypeSqlite:
			allUsersSqlite = allUsers
		case dbabstraction.TypeRedis:
			allUsersRedis = allUsers
		default:
			t.Fatal("Unrecognized database type")
		}
		return len(GetAllUsers())
	}, 2)
	test.IsEqual(t, allUsersSqlite, allUsersRedis)
	runAllTypesNoOutput(t, func() { UpdateUserLastOnline(1) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		retrievedUser, ok := GetUser(1)
		isUpdated := time.Now().Unix()-retrievedUser.LastOnline < 5 && time.Now().Unix()-retrievedUser.LastOnline > -1
		return isUpdated, ok
	}, true, true)
	runAllTypesNoOutput(t, func() { DeleteUser(1) })
	runAllTypesCompareOutput(t, func() any {
		_, ok := GetUser(1)
		return ok
	}, false)

	user.Id = 10
	user.Name = "TEST5"
	runAllTypesNoOutput(t, func() { SaveUser(user, false) })
	runAllTypesCompareOutput(t, func() any {
		retrievedUser, _ := GetUser(10)
		return retrievedUser.Name
	}, "test5")

	runAllTypesCompareOutput(t, func() any {
		_, ok := GetSuperAdmin()
		return ok
	}, false)

	runAllTypesCompareOutput(t, func() any {
		err := EditSuperAdmin("user", "password")
		return err == nil
	}, false)

	runAllTypesNoOutput(t, func() {
		users := GetAllUsers()
		for _, rUser := range users {
			DeleteUser(rUser.Id)
		}
	})
	runAllTypesCompareOutput(t, func() any { return len(GetAllUsers()) }, 0)

	runAllTypesCompareOutput(t, func() any {
		return EditSuperAdmin("username", "pwhash")
	}, nil)
	runAllTypesCompareOutput(t, func() any {
		admin, ok := GetSuperAdmin()
		test.IsEqualInt(t, int(admin.Permissions), int(models.UserPermissionAll))
		test.IsEqualInt(t, int(admin.UserLevel), int(models.UserLevelSuperAdmin))
		test.IsEqualString(t, admin.Name, "username")
		test.IsEqualString(t, admin.Password, "pwhash")
		return ok
	}, true)

	runAllTypesNoOutput(t, func() {
		err := EditSuperAdmin("username2", "")
		test.IsNil(t, err)
		admin, ok := GetSuperAdmin()
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, admin.Name, "username2")
		test.IsEqualString(t, admin.Password, "pwhash")
	})
	runAllTypesNoOutput(t, func() {
		err := EditSuperAdmin("", "pwhash2")
		test.IsNil(t, err)
		admin, ok := GetSuperAdmin()
		test.IsEqualBool(t, ok, true)
		test.IsEqualString(t, admin.Name, "username2")
		test.IsEqualString(t, admin.Password, "pwhash2")
	})

	user.Name = ""
	defer test.ExpectPanic(t)
	SaveUser(user, true)
}

func TestUpgrade(t *testing.T) {
	runAllTypesNoOutput(t, func() { test.IsEqualBool(t, db.GetDbVersion() != 1, true) })
	runAllTypesNoOutput(t, func() { db.SetDbVersion(1) })
	runAllTypesNoOutput(t, func() { test.IsEqualInt(t, db.GetDbVersion(), 1) })
	// runAllTypesNoOutput(t, func() { Upgrade() })
	// runAllTypesNoOutput(t, func() { test.IsEqualInt(t, db.GetDbVersion(), db.GetSchemaVersion()) })
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
		test.IsEqual(t, output, expectedOutput)
	}
}

func runAllTypesCompareTwoOutputs(t *testing.T, functionToRun func() (any, any), expectedOutput1, expectedOutput2 any) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		output1, output2 := functionToRun()
		test.IsEqual(t, output1, expectedOutput1)
		test.IsEqual(t, output2, expectedOutput2)
	}
}

func TestParseUrl(t *testing.T) {
	expectedOutput := models.DbConnection{}
	output, err := ParseUrl("invalid", false)
	test.IsNotNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	_, err = ParseUrl("", false)
	test.IsNotNil(t, err)
	_, err = ParseUrl("inv\r\nalid", false)
	test.IsNotNil(t, err)
	_, err = ParseUrl("", false)
	test.IsNotNil(t, err)

	expectedOutput = models.DbConnection{
		HostUrl: "./test",
		Type:    dbabstraction.TypeSqlite,
	}
	output, err = ParseUrl("sqlite://./test", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	_, err = ParseUrl("sqlite:///invalid", true)
	test.IsNotNil(t, err)
	output, err = ParseUrl("sqlite:///invalid", false)
	test.IsNil(t, err)
	test.IsEqualString(t, output.HostUrl, "/invalid")

	expectedOutput = models.DbConnection{
		HostUrl:     "127.0.0.1:1234",
		RedisPrefix: "",
		Username:    "",
		Password:    "",
		RedisUseSsl: false,
		Type:        dbabstraction.TypeRedis,
	}
	output, err = ParseUrl("redis://127.0.0.1:1234", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	expectedOutput = models.DbConnection{
		HostUrl:     "127.0.0.1:1234",
		RedisPrefix: "tpref",
		Username:    "tuser",
		Password:    "tpw",
		RedisUseSsl: true,
		Type:        dbabstraction.TypeRedis,
	}
	output, err = ParseUrl("redis://tuser:tpw@127.0.0.1:1234/?ssl=true&prefix=tpref", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)
}

func TestMigration(t *testing.T) {
	configNew := models.DbConnection{
		RedisPrefix: "testmigrate_",
		HostUrl:     "127.0.0.1:26379",
		Type:        1, // dbabstraction.TypeRedis
	}
	dbOld, err := dbabstraction.GetNew(configSqlite)
	test.IsNil(t, err)
	testFile := models.File{Id: "file1234", HotlinkId: "hotlink123"}
	dbOld.SaveMetaData(testFile)
	dbOld.SaveHotlink(testFile)
	dbOld.SaveApiKey(models.ApiKey{Id: "api123"})
	dbOld.SaveHotlink(testFile)
	dbOld.Close()

	Migrate(configSqlite, configNew)

	dbNew, err := dbabstraction.GetNew(configNew)
	test.IsNil(t, err)
	_, ok := dbNew.GetHotlink("hotlink123")
	test.IsEqualBool(t, ok, true)
	_, ok = dbNew.GetApiKey("api123")
	test.IsEqualBool(t, ok, true)
	_, ok = dbNew.GetMetaDataById("file1234")
	test.IsEqualBool(t, ok, true)
}
