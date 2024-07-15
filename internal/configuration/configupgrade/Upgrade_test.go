package configupgrade

import (
	"github.com/forceu/gokapi/internal/environment"
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
	env := environment.New()
	oldConfigFile.ConfigVersion = 15
	upgradeDone := DoUpgrade(&oldConfigFile, &env)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualInt(t, exitCode, 1)

	exitCode = 0
	oldConfigFile.ConfigVersion = 17
	oldConfigFile.Authentication.OAuthUsers = []string{"test"}
	oldConfigFile.MaxMemory = 40
	upgradeDone = DoUpgrade(&oldConfigFile, &env)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualString(t, oldConfigFile.Authentication.OAuthUserScope, "email")
	test.IsEqualInt(t, oldConfigFile.MaxMemory, 50)
	test.IsEqualInt(t, exitCode, 0)

	exitCode = 0
	oldConfigFile.ConfigVersion = CurrentConfigVersion
	upgradeDone = DoUpgrade(&oldConfigFile, &env)
	test.IsEqualBool(t, upgradeDone, false)
	test.IsEqualInt(t, exitCode, 0)

}
