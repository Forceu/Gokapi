package configuration

import (
	"Gokapi/internal/environment"
	testconfiguration "Gokapi/internal/test"
	testconfiguration2 "Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration2.Create(false)
	exitVal := m.Run()
	testconfiguration2.Delete()
	os.Exit(exitVal)
}

func TestLoad(t *testing.T) {
	Load()
	testconfiguration.IsEqualString(t, Environment.ConfigDir, "test")
	testconfiguration.IsEqualString(t, ServerSettings.Port, "127.0.0.1:53843")
	testconfiguration.IsEqualString(t, ServerSettings.AdminName, "test")
	testconfiguration.IsEqualString(t, ServerSettings.ServerUrl, "http://127.0.0.1:53843/")
	testconfiguration.IsEqualString(t, ServerSettings.AdminPassword, "10340aece68aa4fb14507ae45b05506026f276cf")
	testconfiguration.IsEqualString(t, HashPassword("testtest", false), "10340aece68aa4fb14507ae45b05506026f276cf")
	testconfiguration.IsEqualInt(t, ServerSettings.LengthId, 20)
}

func TestMutex(t *testing.T) {
	finished := make(chan bool)
	oldValue := ServerSettings.ConfigVersion
	go func() {
		time.Sleep(100 * time.Millisecond)
		LockSessions()
		testconfiguration.IsEqualInt(t, ServerSettings.ConfigVersion, -9)
		ServerSettings.ConfigVersion = oldValue
		UnlockSessionsAndSave()
		testconfiguration.IsEqualInt(t, ServerSettings.ConfigVersion, oldValue)
		finished <- true
	}()
	LockSessions()
	ServerSettings.ConfigVersion = -9
	time.Sleep(150 * time.Millisecond)
	testconfiguration.IsEqualInt(t, ServerSettings.ConfigVersion, -9)
	UnlockSessionsAndSave()
	<-finished
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
	testconfiguration.IsEqualString(t, Environment.ConfigDir, "test")
	testconfiguration.IsEqualString(t, ServerSettings.Port, ":1234")
	testconfiguration.IsEqualString(t, ServerSettings.AdminName, "test2")
	testconfiguration.IsEqualString(t, ServerSettings.ServerUrl, "http://test.com/")
	testconfiguration.IsEqualString(t, ServerSettings.RedirectUrl, "http://test2.com")
	testconfiguration.IsEqualString(t, ServerSettings.AdminPassword, "5bbf5684437a4c658d2e0890d784694afb63f715")
	testconfiguration.IsEqualString(t, HashPassword("testtest2", false), "5bbf5684437a4c658d2e0890d784694afb63f715")
	testconfiguration.IsEqualInt(t, ServerSettings.LengthId, 15)
	os.Remove("test/config.json")
	os.Unsetenv("GOKAPI_SALT_ADMIN")
	Load()
	testconfiguration.IsEqualInt(t, len(ServerSettings.SaltAdmin), 30)
	testconfiguration.IsNotEqualString(t, ServerSettings.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	os.Unsetenv("GOKAPI_USERNAME")
	os.Unsetenv("GOKAPI_PASSWORD")
	os.Unsetenv("GOKAPI_PORT")
	os.Unsetenv("GOKAPI_EXTERNAL_URL")
	os.Unsetenv("GOKAPI_REDIRECT_URL")
	os.Unsetenv("GOKAPI_LOCALHOST")
}

func TestUpgradeDb(t *testing.T) {
	testconfiguration2.WriteUpgradeConfigFile()
	Load()
	testconfiguration.IsEqualString(t, ServerSettings.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	testconfiguration.IsEqualString(t, ServerSettings.SaltFiles, "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu")
	testconfiguration.IsEqualString(t, ServerSettings.DataDir, Environment.DataDir)
	testconfiguration.IsEqualInt(t, ServerSettings.LengthId, 15)
	testconfiguration.IsEqualBool(t, ServerSettings.Hotlinks == nil, false)
	testconfiguration.IsEqualBool(t, ServerSettings.DownloadStatus == nil, false)
	testconfiguration.IsEqualString(t, ServerSettings.Files["MgXJLe4XLfpXcL12ec4i"].ContentType, "application/octet-stream")
	testconfiguration.IsEqualInt(t, ServerSettings.ConfigVersion, currentConfigVersion)
	testconfiguration2.Create(false)
	Load()
}

func TestAskForUsername(t *testing.T) {
	original := testconfiguration2.StartMockInputStdin("admin")
	output := askForUsername()
	testconfiguration2.StopMockInputStdin(original)
	testconfiguration.IsEqualString(t, output, "admin")
}

func TestIsValidPortNumber(t *testing.T) {
	testconfiguration.IsEqualBool(t, isValidPortNumber("invalid"), false)
	testconfiguration.IsEqualBool(t, isValidPortNumber("-1"), false)
	testconfiguration.IsEqualBool(t, isValidPortNumber("0"), true)
	testconfiguration.IsEqualBool(t, isValidPortNumber("100"), true)
	testconfiguration.IsEqualBool(t, isValidPortNumber("65353"), true)
	testconfiguration.IsEqualBool(t, isValidPortNumber("65354"), false)
}

func TestHashPassword(t *testing.T) {
	testconfiguration.IsEqualString(t, HashPassword("123", false), "423b63a68c68bd7e07b14590927c1e9a473fe035")
	testconfiguration.IsEqualString(t, HashPassword("", false), "")
	testconfiguration.IsEqualString(t, HashPassword("123", true), "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7")
}

func TestIsValidUrl(t *testing.T) {
	testconfiguration.IsEqualBool(t, isValidUrl("http://"), false)
	testconfiguration.IsEqualBool(t, isValidUrl("https://"), false)
	testconfiguration.IsEqualBool(t, isValidUrl("invalid"), false)
	testconfiguration.IsEqualBool(t, isValidUrl("http://abc"), true)
	testconfiguration.IsEqualBool(t, isValidUrl("https://abc"), true)
}

func TestAddTrailingSlash(t *testing.T) {
	testconfiguration.IsEqualString(t, addTrailingSlash("abc"), "abc/")
	testconfiguration.IsEqualString(t, addTrailingSlash("abc/"), "abc/")
	testconfiguration.IsEqualString(t, addTrailingSlash("/"), "/")
	testconfiguration.IsEqualString(t, addTrailingSlash(""), "/")
}

func TestAskForRedirect(t *testing.T) {
	original := testconfiguration2.StartMockInputStdin("")
	url := askForRedirect()
	testconfiguration2.StopMockInputStdin(original)
	testconfiguration.IsEqualString(t, url, "https://github.com/Forceu/Gokapi/")
	original = testconfiguration2.StartMockInputStdin("https://test.com")
	url = askForRedirect()
	testconfiguration2.StopMockInputStdin(original)
	testconfiguration.IsEqualString(t, url, "https://test.com")
}

func TestAskForLocalOnly(t *testing.T) {
	environment.IsDocker = "true"
	testconfiguration.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	environment.IsDocker = "false"
	original := testconfiguration2.StartMockInputStdin("")
	testconfiguration.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	testconfiguration2.StopMockInputStdin(original)
	original = testconfiguration2.StartMockInputStdin("no")
	testconfiguration.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	testconfiguration2.StopMockInputStdin(original)
	original = testconfiguration2.StartMockInputStdin("yes")
	testconfiguration.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	testconfiguration2.StopMockInputStdin(original)
	original = testconfiguration2.StartMockInputStdin("n")
	testconfiguration.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	testconfiguration2.StopMockInputStdin(original)
}

func TestAskForPort(t *testing.T) {
	original := testconfiguration2.StartMockInputStdin("8000")
	testconfiguration.IsEqualString(t, askForPort(), "8000")
	testconfiguration2.StopMockInputStdin(original)
	original = testconfiguration2.StartMockInputStdin("")
	testconfiguration.IsEqualString(t, askForPort(), defaultPort)
	testconfiguration2.StopMockInputStdin(original)
}
func TestAskForUrl(t *testing.T) {
	original := testconfiguration2.StartMockInputStdin("https://test.com")
	testconfiguration.IsEqualString(t, askForUrl("1234"), "https://test.com/")
	testconfiguration2.StopMockInputStdin(original)
	original = testconfiguration2.StartMockInputStdin("")
	testconfiguration.IsEqualString(t, askForUrl("1234"), "http://127.0.0.1:1234/")
	testconfiguration2.StopMockInputStdin(original)
}
