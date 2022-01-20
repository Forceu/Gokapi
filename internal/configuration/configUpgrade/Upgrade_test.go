package configUpgrade

import (
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
	upgradeDone := DoUpgrade(&bufferConfig, &env)
	test.IsEqualBool(t, upgradeDone, true)
	upgradeDone = DoUpgrade(&bufferConfig, &env)
	test.IsEqualBool(t, upgradeDone, false)
	firstUpgrade := oldConfigFile
	upgradeDone = DoUpgrade(&firstUpgrade, &env)
	test.IsEqualBool(t, upgradeDone, true)

	test.IsEqualString(t, firstUpgrade.Authentication.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	test.IsEqualString(t, firstUpgrade.Authentication.SaltFiles, "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu")
	test.IsEqualString(t, firstUpgrade.DataDir, env.DataDir)
	test.IsEqualInt(t, firstUpgrade.LengthId, 15)
	test.IsEqualInt(t, firstUpgrade.ConfigVersion, CurrentConfigVersion)
	test.IsEqualInt(t, firstUpgrade.MaxFileSizeMB, 5)
	test.IsEqualInt(t, firstUpgrade.Authentication.Method, 0)
	test.IsEqualBool(t, firstUpgrade.Authentication.HeaderUsers == nil, false)
	test.IsEqualBool(t, firstUpgrade.Authentication.OauthUsers == nil, false)
	test.IsEqualString(t, firstUpgrade.Authentication.Username, "admin")
	test.IsEqualString(t, firstUpgrade.Authentication.Password, "7450c2403ab85f0e8d5436818b66b99fdd287ac6")

	oldConfigFile.ConfigVersion = 8
	testconfiguration.WriteUpgradeConfigFileV8()
	upgradeDone = DoUpgrade(&oldConfigFile, &env)
	test.IsEqualBool(t, upgradeDone, true)
	test.IsEqualString(t, oldConfigFile.Authentication.SaltAdmin, "LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C")
	test.IsEqualString(t, oldConfigFile.Authentication.SaltFiles, "lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE")

	os.Unsetenv("GOKAPI_MAX_FILESIZE")

}
