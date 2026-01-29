package dbcache

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/forceu/gokapi/internal/test"
)

func TestLastOnlineRequiresSave(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		Init()
		test.IsEqualBool(t, RequireSaveUserOnline(100), true)
		test.IsEqualBool(t, RequireSaveUserOnline(100), false)
		time.Sleep(31 * time.Second)
		test.IsEqualBool(t, RequireSaveUserOnline(100), true)

		test.IsEqualBool(t, RequireSaveApiKeyUsage("100"), true)
		test.IsEqualBool(t, RequireSaveApiKeyUsage("100"), false)
		time.Sleep(31 * time.Second)
		test.IsEqualBool(t, RequireSaveApiKeyUsage("100"), true)
	})
}

func TestResetAll(t *testing.T) {
	Init()
	test.IsEqualBool(t, RequireSaveUserOnline(200), true)
	test.IsEqualBool(t, RequireSaveApiKeyUsage("200"), true)
	test.IsEqualBool(t, RequireSaveUserOnline(200), false)
	test.IsEqualBool(t, RequireSaveApiKeyUsage("200"), false)
	Init()
	test.IsEqualBool(t, RequireSaveUserOnline(200), true)
	test.IsEqualBool(t, RequireSaveApiKeyUsage("200"), true)
	test.IsEqualBool(t, RequireSaveApiKeyUsage("200"), false)
}
