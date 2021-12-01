// +build test

package configuration

import (
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
	test.IsEqualString(t, serverSettings.AdminName, "test")
	test.IsEqualString(t, serverSettings.ServerUrl, "http://127.0.0.1:53843/")
	test.IsEqualString(t, serverSettings.AdminPassword, "10340aece68aa4fb14507ae45b05506026f276cf")
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

func TestCreateNewConfig(t *testing.T) {
	os.Remove("test/config.json")
	os.Setenv("GOKAPI_USERNAME", "test2")
	os.Setenv("GOKAPI_PASSWORD", "testtest2")
	os.Setenv("GOKAPI_PORT", "1234")
	os.Setenv("GOKAPI_EXTERNAL_URL", "http://test.com")
	os.Setenv("GOKAPI_REDIRECT_URL", "http://test2.com")
	os.Setenv("GOKAPI_SALT_ADMIN", "salt123")
	os.Setenv("GOKAPI_LOCALHOST", "false")
	os.Setenv("GOKAPI_USE_SSL", "false")
	Load()
	test.IsEqualString(t, Environment.ConfigDir, "test")
	test.IsEqualString(t, serverSettings.Port, ":1234")
	test.IsEqualString(t, serverSettings.AdminName, "test2")
	test.IsEqualString(t, serverSettings.ServerUrl, "http://test.com/")
	test.IsEqualString(t, serverSettings.RedirectUrl, "http://test2.com")
	test.IsEqualString(t, serverSettings.AdminPassword, "5bbf5684437a4c658d2e0890d784694afb63f715")
	test.IsEqualString(t, HashPassword("testtest2", false), "5bbf5684437a4c658d2e0890d784694afb63f715")
	test.IsEqualInt(t, serverSettings.LengthId, 15)
	test.IsEqualBool(t, serverSettings.UseSsl, false)
	os.Remove("test/config.json")
	os.Unsetenv("GOKAPI_SALT_ADMIN")
	Load()
	test.IsEqualInt(t, len(serverSettings.SaltAdmin), 30)
	test.IsEqualInt(t, serverSettings.MaxMemory, 20)
	test.IsNotEqualString(t, serverSettings.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	test.IsEqualInt(t, serverSettings.MaxFileSizeMB, 102400)
	test.IsEqualBool(t, serverSettings.DisableLogin, false)
	os.Unsetenv("GOKAPI_USERNAME")
	os.Unsetenv("GOKAPI_PASSWORD")
	os.Unsetenv("GOKAPI_PORT")
	os.Unsetenv("GOKAPI_EXTERNAL_URL")
	os.Unsetenv("GOKAPI_REDIRECT_URL")
	os.Unsetenv("GOKAPI_LOCALHOST")
	os.Unsetenv("GOKAPI_USE_SSL")
}

func TestUpgradeDb(t *testing.T) {
	testconfiguration.WriteUpgradeConfigFile()
	os.Setenv("GOKAPI_USE_SSL", "true")
	os.Setenv("GOKAPI_DISABLE_LOGIN", "true")
	os.Setenv("GOKAPI_MAX_FILESIZE", "5")
	Load()
	test.IsEqualString(t, serverSettings.SaltAdmin, "eefwkjqweduiotbrkl##$2342brerlk2321")
	test.IsEqualString(t, serverSettings.SaltFiles, "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu")
	test.IsEqualString(t, serverSettings.DataDir, Environment.DataDir)
	test.IsEqualInt(t, serverSettings.LengthId, 15)
	test.IsEqualBool(t, serverSettings.Hotlinks == nil, false)
	test.IsEqualBool(t, serverSettings.Sessions == nil, false)
	test.IsEqualBool(t, serverSettings.DownloadStatus == nil, false)
	test.IsEqualString(t, serverSettings.Files["MgXJLe4XLfpXcL12ec4i"].ContentType, "application/octet-stream")
	test.IsEqualInt(t, serverSettings.ConfigVersion, currentConfigVersion)
	test.IsEqualBool(t, serverSettings.UseSsl, true)
	test.IsEqualInt(t, serverSettings.MaxFileSizeMB, 5)
	test.IsEqualBool(t, serverSettings.DisableLogin, true)
	os.Unsetenv("GOKAPI_USE_SSL")
	os.Unsetenv("GOKAPI_MAX_FILESIZE")
	os.Unsetenv("GOKAPI_DISABLE_LOGIN")
	testconfiguration.Create(false)
	Load()
}

func TestAskForUsername(t *testing.T) {
	original := test.StartMockInputStdin("admin")
	output := askForUsername(1)
	test.StopMockInputStdin(original)
	test.IsEqualString(t, output, "admin")
	osExit = test.ExitCode(t, 1)
	askForUsername(6)
	osExit = os.Exit
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
	test.IsEqualString(t, HashPassword("123", false), "423b63a68c68bd7e07b14590927c1e9a473fe035")
	test.IsEqualString(t, HashPassword("", false), "")
	test.IsEqualString(t, HashPassword("123", true), "7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7")
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
	original := test.StartMockInputStdin("")
	url := askForRedirect()
	test.StopMockInputStdin(original)
	test.IsEqualString(t, url, "https://github.com/Forceu/Gokapi/")
	original = test.StartMockInputStdin("https://test.com")
	url = askForRedirect()
	test.StopMockInputStdin(original)
	test.IsEqualString(t, url, "https://test.com")
}

func TestAskForLocalOnly(t *testing.T) {
	environment.IsDocker = "true"
	test.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	environment.IsDocker = "false"
	original := test.StartMockInputStdin("")
	test.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	test.StopMockInputStdin(original)
	original = test.StartMockInputStdin("no")
	test.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	test.StopMockInputStdin(original)
	original = test.StartMockInputStdin("yes")
	test.IsEqualString(t, askForLocalOnly(), environment.IsTrue)
	test.StopMockInputStdin(original)
	original = test.StartMockInputStdin("n")
	test.IsEqualString(t, askForLocalOnly(), environment.IsFalse)
	test.StopMockInputStdin(original)
}

func TestAskForPort(t *testing.T) {
	original := test.StartMockInputStdin("8000")
	test.IsEqualString(t, askForPort(), "8000")
	test.StopMockInputStdin(original)
	original = test.StartMockInputStdin("")
	test.IsEqualString(t, askForPort(), defaultPort)
	test.StopMockInputStdin(original)
}

func TestAskForUrl(t *testing.T) {
	original := test.StartMockInputStdin("https://test.com")
	test.IsEqualString(t, askForUrl("1234"), "https://test.com/")
	test.StopMockInputStdin(original)
	original = test.StartMockInputStdin("")
	test.IsEqualString(t, askForUrl("1234"), "http://127.0.0.1:1234/")
	test.StopMockInputStdin(original)
}

func TestAskForPassword(t *testing.T) {
	os.Setenv("GOKAPI_PASSWORD", "not_short")
	Load()
	test.IsEqualString(t, askForPassword(), "not_short")
}

func TestAskForSsl(t *testing.T) {
	original := test.StartMockInputStdin("y")
	test.IsEqualBool(t, askForSsl(), true)
	test.StartMockInputStdin("n")
	test.IsEqualBool(t, askForSsl(), false)
	test.StartMockInputStdin("")
	test.IsEqualBool(t, askForSsl(), false)
	test.StopMockInputStdin(original)
}

func TestExitValues(t *testing.T) {
	os.Setenv("GOKAPI_PASSWORD", "short")
	os.Setenv("GOKAPI_EXTERNAL_URL", "invalid")
	os.Setenv("GOKAPI_REDIRECT_URL", "invalid")
	Environment = environment.New()
	osExit = test.ExitCode(t, 1)
	askForPassword()
	osExit = test.ExitCode(t, 1)
	askForUrl("123")
	osExit = test.ExitCode(t, 1)
	askForRedirect()
	osExit = os.Exit
}
