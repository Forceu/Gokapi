package configupgrade

import (
	"github.com/forceu/gokapi/internal/configuration/database"
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

var oldConfigFile = models.Configuration{
	Authentication: models.AuthenticationConfig{},
	Port:           "127.0.0.1:53844",
	ServerUrl:      "https://gokapi.url/",
	RedirectUrl:    "https://github.com/Forceu/Gokapi/",
}

func TestUpgradeDb(t *testing.T) {
	exitCode := 0
	osExit = func(code int) {
		exitCode = code
	}
	oldConfigFile.ConfigVersion = 10
	upgradeDone := DoUpgrade(&oldConfigFile, nil)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualInt(t, exitCode, 1)

	database.Init("./test/filestorage.db")
	exitCode = 0
	oldConfigFile.ConfigVersion = 11
	upgradeDone = DoUpgrade(&oldConfigFile, nil)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualInt(t, exitCode, 0)

	exitCode = 0
	oldConfigFile.ConfigVersion = CurrentConfigVersion
	upgradeDone = DoUpgrade(&oldConfigFile, nil)
	test.IsEqualBool(t, upgradeDone, false)
	test.IsEqualInt(t, exitCode, 0)

}
