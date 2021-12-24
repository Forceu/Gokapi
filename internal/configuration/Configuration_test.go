//go:build test
// +build test

package configuration

import (
	"Gokapi/internal/configuration/configUpgrade"
	"Gokapi/internal/environment"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
	"time"
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
	test.IsEqualString(t, serverSettings.Port, "127.0.0.1:53843")
	test.IsEqualString(t, serverSettings.Authentication.Username, "test")
	test.IsEqualString(t, serverSettings.ServerUrl, "http://127.0.0.1:53843/")
	test.IsEqualString(t, serverSettings.Authentication.Password, "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualString(t, HashPassword("testtest", false), "10340aece68aa4fb14507ae45b05506026f276cf")
	test.IsEqualBool(t, serverSettings.UseSsl, false)
	test.IsEqualInt(t, GetLengthId(), 20)
	settings := GetServerSettings()
	Release()
	test.IsEqualInt(t, settings.LengthId, 20)
}

func TestMutexSession(t *testing.T) {
	finished := make(chan bool)
	oldValue := serverSettings.ConfigVersion
	go func() {
		time.Sleep(100 * time.Millisecond)
		Lock()
		test.IsEqualInt(t, serverSettings.ConfigVersion, -9)
		serverSettings.ConfigVersion = oldValue
		ReleaseAndSave()
		test.IsEqualInt(t, serverSettings.ConfigVersion, oldValue)
		finished <- true
	}()
	Lock()
	serverSettings.ConfigVersion = -9
	time.Sleep(150 * time.Millisecond)
	test.IsEqualInt(t, serverSettings.ConfigVersion, -9)
	Release()
	<-finished
}

func TestUpgradeDb(t *testing.T) {
	testconfiguration.WriteUpgradeConfigFileV0()
	os.Setenv("GOKAPI_USE_SSL", "true")
	os.Setenv("GOKAPI_MAX_FILESIZE", "5")
	Load()
	test.IsEqualString(t, serverSettings.Authentication.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	test.IsEqualString(t, serverSettings.Authentication.SaltFiles, "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu")
	test.IsEqualString(t, serverSettings.DataDir, Environment.DataDir)
	test.IsEqualInt(t, serverSettings.LengthId, 15)
	test.IsEqualBool(t, serverSettings.Hotlinks == nil, false)
	test.IsEqualBool(t, serverSettings.Sessions == nil, false)
	test.IsEqualBool(t, serverSettings.DownloadStatus == nil, false)
	test.IsEqualString(t, serverSettings.Files["MgXJLe4XLfpXcL12ec4i"].ContentType, "application/octet-stream")
	test.IsEqualInt(t, serverSettings.ConfigVersion, configUpgrade.CurrentConfigVersion)
	test.IsEqualBool(t, serverSettings.UseSsl, true)
	test.IsEqualInt(t, serverSettings.MaxFileSizeMB, 5)
	os.Unsetenv("GOKAPI_USE_SSL")
	os.Unsetenv("GOKAPI_MAX_FILESIZE")
	testconfiguration.Create(false)
	Load()
}
func TestHashPassword(t *testing.T) {
	test.IsEqualString(t, HashPassword("123", false), "423b63a68c68bd7e07b14590927c1e9a473fe035")
	test.IsEqualString(t, HashPassword("", false), "")
	test.IsEqualString(t, HashPassword("123", true), "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7")
}

func TestAskForPassword(t *testing.T) {
	os.Setenv("GOKAPI_PASSWORD", "not_short")
	Load()
	test.IsEqualString(t, askForPassword(), "not_short")
}

func TestExitValues(t *testing.T) {
	os.Setenv("GOKAPI_PASSWORD", "short")
	os.Setenv("GOKAPI_EXTERNAL_URL", "invalid")
	os.Setenv("GOKAPI_REDIRECT_URL", "invalid")
	Environment = environment.New()
	osExit = test.ExitCode(t, 1)
	askForPassword()
	osExit = test.ExitCode(t, 1)
}
