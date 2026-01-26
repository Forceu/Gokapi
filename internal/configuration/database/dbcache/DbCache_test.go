package dbcache

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/forceu/gokapi/internal/test"
)

func TestLastOnlineRequiresSave(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		test.IsEqualBool(t, LastOnlineRequiresSave(100), true)
		test.IsEqualBool(t, LastOnlineRequiresSave(100), false)
		time.Sleep(61 * time.Second)
		test.IsEqualBool(t, LastOnlineRequiresSave(100), true)
	})
}

func TestResetAll(t *testing.T) {
	LastOnlineRequiresSave(100)
	test.IsEqualInt(t, len(lastOnlineTimeUpdate), 1)
	ResetAll()
	test.IsEqualInt(t, len(lastOnlineTimeUpdate), 0)
}
