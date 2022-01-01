//go:build test
// +build test

package environment

import (
	"Gokapi/internal/test"
	"os"
	"testing"
)


func TestTempDir(t *testing.T) {
	test.IsEqualString(t,os.Getenv("TMPDIR"),"")
	New()
	test.IsEqualString(t,os.Getenv("TMPDIR"),"")
	IsDocker = "true"
	New()
	test.IsEqualString(t,os.Getenv("TMPDIR"),"data")
	os.Setenv("TMPDIR","test")
	New()
	test.IsEqualString(t,os.Getenv("TMPDIR"),"test")
	os.Unsetenv("TMPDIR")
	IsDocker="false"
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
	os.Unsetenv("GOKAPI_LOCALHOST")
	os.Unsetenv("GOKAPI_LENGTH_ID")
	env = New()
	os.Setenv("GOKAPI_LENGTH_ID", "15")
	os.Setenv("GOKAPI_LOCALHOST", "invalid")
	os.Setenv("GOKAPI_LENGTH_ID", "invalid")
	env = New()
	test.IsEqualInt(t, env.LengthId, -1)
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
