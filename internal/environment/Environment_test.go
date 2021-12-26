//go:build test
// +build test

package environment

import (
	"Gokapi/internal/test"
	"os"
	"testing"
)

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


func TestToBool(t *testing.T) {
	test.IsEqualBool(t, ToBool(IsTrue), true)
	test.IsEqualBool(t, ToBool(IsFalse), false)
	test.IsEqualBool(t, ToBool("invalid"), false)
}
