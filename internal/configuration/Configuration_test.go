package configuration

import (
	"Gokapi/internal/environment"
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
	test.IsEqualInt(t, ServerSettings.LengthId, 20)
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
	test.IsEqualInt(t, ServerSettings.LengthId, 15)
	os.Unsetenv("GOKAPI_USERNAME")
	os.Unsetenv("GOKAPI_PASSWORD")
	os.Unsetenv("GOKAPI_PORT")
	os.Unsetenv("GOKAPI_EXTERNAL_URL")
	os.Unsetenv("GOKAPI_REDIRECT_URL")
	os.Unsetenv("GOKAPI_SALT_ADMIN")
	os.Unsetenv("GOKAPI_LOCALHOST")
}

func TestUpgradeDb(t *testing.T) {
	Load()
	ServerSettings.ConfigVersion = 0
	updateConfig()
	Load()
	test.IsEqualString(t, ServerSettings.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	test.IsEqualInt(t, ServerSettings.LengthId, 15)
	test.IsEqualInt(t, ServerSettings.ConfigVersion, currentConfigVersion)
}

func TestAskForUsername(t *testing.T) {
	original := testconfiguration.StartMockInputStdin(t, "admin")
	output := askForUsername()
	testconfiguration.StopMockInputStdin(original)
	test.IsEqualString(t, output, "admin")
}

func TestIsValidPortNumber(t *testing.T) {
	test.IsEqualBool(t, isValidPortNumber("invalid"), false)
	test.IsEqualBool(t, isValidPortNumber("-1"), false)
	test.IsEqualBool(t, isValidPortNumber("0"), true)
	test.IsEqualBool(t, isValidPortNumber("100"), true)
	test.IsEqualBool(t, isValidPortNumber("65353"), true)
	test.IsEqualBool(t, isValidPortNumber("65354"), false)
}

func TestHashPassword(t *testing.T) {
	test.IsEqualString(t, HashPassword("123", false), "45dcdc57b6a50bd0020fabc958ae254406713559")
	test.IsEqualString(t, HashPassword("", false), "")
	test.IsEqualString(t, HashPassword("123", true), "e143a1801faba4c5c6fdc2e823127c988940f72e")
}

func TestIsValidUrl(t *testing.T) {
	test.IsEqualBool(t, isValidUrl("http://"), false)
	test.IsEqualBool(t, isValidUrl("https://"), false)
	test.IsEqualBool(t, isValidUrl("invalid"), false)
	test.IsEqualBool(t, isValidUrl("http://abc"), true)
	test.IsEqualBool(t, isValidUrl("https://abc"), true)
}

func TestAddTrailingSlash(t *testing.T) {
	test.IsEqualString(t, addTrailingSlash("abc"), "abc/")
	test.IsEqualString(t, addTrailingSlash("abc/"), "abc/")
	test.IsEqualString(t, addTrailingSlash("/"), "/")
	test.IsEqualString(t, addTrailingSlash(""), "/")
}

func TestAskForRedirect(t *testing.T) {
	original := testconfiguration.StartMockInputStdin(t, "")
	url := askForRedirect()
	testconfiguration.StopMockInputStdin(original)
	test.IsEqualString(t, url, "https://github.com/Forceu/Gokapi/")
	original = testconfiguration.StartMockInputStdin(t, "https://test.com")
	url = askForRedirect()
	testconfiguration.StopMockInputStdin(original)
	test.IsEqualString(t, url, "https://test.com")
}

func TestAskForLocalOnly(t *testing.T) {
	environment.IsDocker = "true"
	test.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	environment.IsDocker = "false"
	original := testconfiguration.StartMockInputStdin(t, "")
	test.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	testconfiguration.StopMockInputStdin(original)
	original = testconfiguration.StartMockInputStdin(t, "no")
	test.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	testconfiguration.StopMockInputStdin(original)
	original = testconfiguration.StartMockInputStdin(t, "yes")
	test.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	testconfiguration.StopMockInputStdin(original)
	original = testconfiguration.StartMockInputStdin(t, "n")
	test.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	testconfiguration.StopMockInputStdin(original)
}

func TestAskForPort(t *testing.T) {
	original := testconfiguration.StartMockInputStdin(t, "8000")
	test.IsEqualString(t, askForPort(), "8000")
	testconfiguration.StopMockInputStdin(original)
	original = testconfiguration.StartMockInputStdin(t, "")
	test.IsEqualString(t, askForPort(), defaultPort)
	testconfiguration.StopMockInputStdin(original)
}
func TestAskForUrl(t *testing.T) {
	original := testconfiguration.StartMockInputStdin(t, "https://test.com")
	test.IsEqualString(t, askForUrl("1234"), "https://test.com/")
	testconfiguration.StopMockInputStdin(original)
	original = testconfiguration.StartMockInputStdin(t, "")
	test.IsEqualString(t, askForUrl("1234"), "http://127.0.0.1:1234/")
	testconfiguration.StopMockInputStdin(original)
}
