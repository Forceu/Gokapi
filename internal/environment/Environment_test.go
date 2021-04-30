package environment

import (
	"Gokapi/internal/test"
	"os"
	"testing"
)

func TestEnvLoad(t *testing.T) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_CONFIG_FILE", "test2")
	os.Setenv("GOKAPI_LOCALHOST", "yes")
	os.Setenv("GOKAPI_LENGTH_ID", "7")
	env := New()
	test.IsEqualString(t, env.ConfigPath, "test/test2")
	test.IsEqualString(t, env.WebserverLocalhost, IsTrue)
	test.IsEqualInt(t, env.LengthId, 7)
	os.Setenv("GOKAPI_LENGTH_ID", "3")
	os.Setenv("GOKAPI_LOCALHOST", "false")
	env = New()
	test.IsEqualInt(t, env.LengthId, 5)
	test.IsEqualString(t, env.WebserverLocalhost, IsFalse)
	os.Unsetenv("GOKAPI_LOCALHOST")
	os.Unsetenv("GOKAPI_LENGTH_ID")
	env = New()
	test.IsEqualString(t, env.WebserverLocalhost, "")
	os.Setenv("GOKAPI_LENGTH_ID", "15")
	os.Setenv("GOKAPI_LOCALHOST", "invalid")
	os.Setenv("GOKAPI_LENGTH_ID", "invalid")
	env = New()
	test.IsEqualString(t, env.WebserverLocalhost, "")
	test.IsEqualInt(t, env.LengthId, -1)
}
