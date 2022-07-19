package configuration

import (
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/models"
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

func TestLoad(t *testing.T) {
	test.IsEqualBool(t, Exists(), true)
	Load()
	test.IsEqualString(t, Environment.ConfigDir, "test")
	test.IsEqualString(t, serverSettings.Port, "127.0.0.1:53843")
	test.IsEqualString(t, serverSettings.Authentication.Username, "test")
	test.IsEqualString(t, serverSettings.ServerUrl, "http://127.0.0.1:53843/")
	test.IsEqualString(t, serverSettings.Authentication.Password, "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualString(t, HashPassword("testtest", false), "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualBool(t, serverSettings.UseSsl, false)
	test.IsEqualInt(t, serverSettings.LengthId, 20)
	test.IsEqualInt(t, Get().LengthId, 20)
	os.Setenv("GOKAPI_MAX_MEMORY_UPLOAD", "80")
	Load()
	test.IsEqualInt(t, serverSettings.MaxMemory, 80)
	os.Unsetenv("GOKAPI_MAX_MEMORY_UPLOAD")
	Load()
	test.IsEqualInt(t, serverSettings.ConfigVersion, configupgrade.CurrentConfigVersion)
	testconfiguration.Create(false)
	Load()
}

func TestHashPassword(t *testing.T) {
	test.IsEqualString(t, HashPassword("123", false), "423b63a68c68bd7e07b14590927c1e9a473fe035")
	test.IsEqualString(t, HashPassword("", false), "")
	test.IsEqualString(t, HashPassword("123", true), "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7")
}

func TestHashPasswordCustomSalt(t *testing.T) {
	test.IsEmpty(t, HashPasswordCustomSalt("", "123"))
	test.IsEqualString(t, HashPasswordCustomSalt("test", "salt"), "f438229716cab43569496f3a3630b3727524b81b")
	defer test.ExpectPanic(t)
	HashPasswordCustomSalt("1234", "")
}

func TestLoadFromSetup(t *testing.T) {
	newConfig := models.Configuration{
		Authentication: models.AuthenticationConfig{},
		Port:           "localhost:123",
		ServerUrl:      "serverurl",
		RedirectUrl:    "redirect",
		ConfigVersion:  configupgrade.CurrentConfigVersion,
		LengthId:       10,
		DataDir:        "test",
		MaxMemory:      10,
		UseSsl:         true,
		MaxFileSizeMB:  199,
	}
	newCloudConfig := cloudconfig.CloudConfig{Aws: models.AwsConfig{
		Bucket:    "bucket",
		Region:    "region",
		KeyId:     "keyid",
		KeySecret: "secret",
		Endpoint:  "",
	}}

	testconfiguration.WriteCloudConfigFile(true)
	LoadFromSetup(newConfig, nil, false)
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	test.IsEqualString(t, serverSettings.RedirectUrl, "redirect")

	LoadFromSetup(newConfig, &newCloudConfig, false)
	test.FileExists(t, "test/cloudconfig.yml")
	config, ok := cloudconfig.Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, config.Aws.KeyId, "keyid")
	test.IsEqualString(t, serverSettings.ServerUrl, "serverurl")
}

func TestUsesHttps(t *testing.T) {
	usesHttps = false
	test.IsEqualBool(t, UsesHttps(), false)
	usesHttps = true
	test.IsEqualBool(t, UsesHttps(), true)
}
