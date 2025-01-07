package migration

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestGetType(t *testing.T) {
	test.IsEqualString(t, getType(dbabstraction.TypeSqlite), "SQLite")
	test.IsEqualString(t, getType(dbabstraction.TypeRedis), "Redis")
	test.IsEqualString(t, getType(2), "Invalid")
}

var exitCode int

func TestMigration(t *testing.T) {
	osExit = func(code int) { exitCode = code }
	Do(flagparser.MigrateFlags{
		Source:      "",
		Destination: "sqlite://ignore",
	})
	test.IsEqualInt(t, exitCode, 1)
	exitCode = 0

	Do(flagparser.MigrateFlags{
		Source:      "sqlite://./tempfile",
		Destination: "",
	})
	test.IsEqualInt(t, exitCode, 1)
	exitCode = 0

	err := os.WriteFile("tempfile", []byte("ignore"), 777)
	test.IsNil(t, err)
	Do(flagparser.MigrateFlags{
		Source:      "sqlite://./tempfile",
		Destination: "",
	})
	test.IsEqualInt(t, exitCode, 2)
	exitCode = 0

	err = os.Remove("tempfile")
	test.IsNil(t, err)

	dbUrl := testconfiguration.SqliteUrl
	dbUrlNew := dbUrl + "2"
	Do(flagparser.MigrateFlags{
		Source:      dbUrl,
		Destination: dbUrlNew,
	})
	err = os.Setenv("GOKAPI_DATABASE_URL", dbUrlNew)
	test.IsNil(t, err)
	configuration.Load()
	configuration.ConnectDatabase()
	_, ok := database.GetHotlink("PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg")
	test.IsEqualBool(t, ok, true)
	_, ok = database.GetApiKey("validkey")
	test.IsEqualBool(t, ok, true)
	_, ok = database.GetMetaDataById("Wzol7LyY2QVczXynJtVo")
	test.IsEqualBool(t, ok, true)
}
