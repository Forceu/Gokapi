package configUpgrade

import (
	"Gokapi/internal/configuration/dataStorage"
	"Gokapi/internal/environment"
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

var oldConfigFile = models.Configuration{
	Authentication: models.AuthenticationConfig{},
	Port:           "127.0.0.1:53844",
	ServerUrl:      "https://gokapi.url/",
	RedirectUrl:    "https://github.com/Forceu/Gokapi/",
}

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestUpgradeDb(t *testing.T) {
	testconfiguration.WriteUpgradeConfigFileV0()
	os.Setenv("GOKAPI_MAX_FILESIZE", "5")

	env := environment.New()
	bufferConfig := oldConfigFile
	wasExit := false
	osExit = func(code int) {
		wasExit = true
	}
	_ = DoUpgrade(&bufferConfig, &env)
	test.IsEqualBool(t, wasExit, true)

	oldConfigFile.ConfigVersion = 8
	dataStorage.Init("./test/filestorage.db")
	testconfiguration.WriteUpgradeConfigFileV8()
	upgradeDone := DoUpgrade(&oldConfigFile, &env)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualString(t, oldConfigFile.Authentication.SaltAdmin, "LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C")
	test.IsEqualString(t, oldConfigFile.Authentication.SaltFiles, "lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE")
	// TODO write further tests
	os.Unsetenv("GOKAPI_MAX_FILESIZE")

}
