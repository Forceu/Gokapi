package environment

import (
	"github.com/forceu/gokapi/internal/test"
	"os"
	"testing"
)

var returnCode = 0

func TestMain(m *testing.M) {

	osExit = func(code int) {
		returnCode = code
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestTempDir(t *testing.T) {
	test.IsEqualString(t, os.Getenv("TMPDIR"), "")
	New()
	test.IsEqualString(t, os.Getenv("TMPDIR"), "")
	IsDocker = "true"
	New()
	test.IsEqualString(t, os.Getenv("TMPDIR"), "data")
	os.Setenv("TMPDIR", "test")
	New()
	test.IsEqualString(t, os.Getenv("TMPDIR"), "test")
	os.Unsetenv("TMPDIR")
	IsDocker = "false"
}

func TestEnvLoad(t *testing.T) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_CONFIG_FILE", "test2")
	os.Setenv("GOKAPI_LENGTH_ID", "7")
	env := New()
	test.IsEqualString(t, env.ConfigPath, "test/test2")
	test.IsEqualInt(t, env.LengthId, 7)
	os.Setenv("GOKAPI_LENGTH_ID", "3")
	env = New()
	test.IsEqualInt(t, env.LengthId, 5)
	os.Setenv("GOKAPI_LENGTH_ID", "86")
	env = New()
	test.IsEqualInt(t, env.LengthId, 85)
	os.Unsetenv("GOKAPI_LENGTH_ID")
	env = New()
	os.Setenv("GOKAPI_LENGTH_ID", "15")
	os.Setenv("GOKAPI_MAX_MEMORY_UPLOAD", "0")
	os.Setenv("GOKAPI_MAX_FILESIZE", "0")
	env = New()
	test.IsEqualInt(t, env.LengthId, 15)
	test.IsEqualInt(t, env.MaxFileSize, 5)
	test.IsEqualInt(t, env.MaxMemory, 5)
	os.Setenv("GOKAPI_MAX_FILESIZE", "invalid")
	returnCode = 0
	New()
	test.IsEqualInt(t, returnCode, 1)
	os.Unsetenv("GOKAPI_MAX_FILESIZE")
}

func TestIsAwsProvided(t *testing.T) {
	os.Unsetenv("GOKAPI_AWS_BUCKET")
	os.Unsetenv("GOKAPI_AWS_REGION")
	os.Unsetenv("GOKAPI_AWS_KEY")
	os.Unsetenv("GOKAPI_AWS_KEY_SECRET")
	env := New()
	test.IsEqualBool(t, env.IsAwsProvided(), false)
	os.Setenv("GOKAPI_AWS_BUCKET", "test")
	os.Setenv("GOKAPI_AWS_REGION", "test")
	os.Setenv("GOKAPI_AWS_KEY", "test")
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "test")
	env = New()
	test.IsEqualBool(t, env.IsAwsProvided(), true)
}

func TestGetConfigPaths(t *testing.T) {
	configPath, configDir, configFile, awsConfig := GetConfigPaths()
	test.IsEqualString(t, configPath, "test/test2")
	test.IsEqualString(t, configDir, "test")
	test.IsEqualString(t, configFile, "test2")
	test.IsEqualString(t, awsConfig, "test/cloudconfig.yml")
}
