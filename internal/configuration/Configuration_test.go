package configuration

import (
	testconfiguration "Gokapi/internal/test"
	"Gokapi/pkg/test"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestLoad(t *testing.T) {
	Load()
	test.IsEqualString(t, Environment.ConfigDir, "test")
	test.IsEqualString(t, ServerSettings.Port, "127.0.0.1:53843")
	test.IsEqualString(t, ServerSettings.AdminName, "test")
	test.IsEqualString(t, ServerSettings.ServerUrl, "http://127.0.0.1:53843/")
	test.IsEqualString(t, ServerSettings.AdminPassword, "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualString(t, HashPassword("testtest", false), "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualInt(t,ServerSettings.LengthId,20)
}

func TestCreateNewConfig(t *testing.T) {
	os.Remove("test/config.json")
	os.Setenv("GOKAPI_USERNAME", "test2")
	os.Setenv("GOKAPI_PASSWORD", "testtest2")
	os.Setenv("GOKAPI_PORT", "1234")
	os.Setenv("GOKAPI_EXTERNAL_URL", "http://test.com")
	os.Setenv("GOKAPI_REDIRECT_URL", "http://test2.com")
	os.Setenv("GOKAPI_SALT_ADMIN", "salt123")
	os.Setenv("GOKAPI_LOCALHOST", "false")
	Load()
	test.IsEqualString(t, Environment.ConfigDir, "test")
	test.IsEqualString(t, ServerSettings.Port, ":1234")
	test.IsEqualString(t, ServerSettings.AdminName, "test2")
	test.IsEqualString(t, ServerSettings.ServerUrl, "http://test.com/")
	test.IsEqualString(t, ServerSettings.RedirectUrl, "http://test2.com")
	test.IsEqualString(t, ServerSettings.AdminPassword, "5bbf5684437a4c658d2e0890d784694afb63f715")
	test.IsEqualString(t, HashPassword("testtest2", false), "5bbf5684437a4c658d2e0890d784694afb63f715")
	test.IsEqualInt(t,ServerSettings.LengthId,15)
}
