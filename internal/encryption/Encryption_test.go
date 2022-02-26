package encryption

import (
	"bytes"
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestGetRandomCipher(t *testing.T) {
	cipher1, err := GetRandomCipher()
	test.IsNil(t, err)
	test.IsEqualInt(t, len(cipher1), 32)
	cipher2, err := GetRandomCipher()
	test.IsNil(t, err)
	isEqual := bytes.Compare(cipher1, cipher2)
	test.IsEqualBool(t, isEqual != 0, true)
}

func TestPasswordChecksum(t *testing.T) {
	checksum := PasswordChecksum("testpw", "testsalt")
	test.IsEqualString(t, checksum, "30161cdf03347d6d3f99743532b8523e03e79d4d91ddd3a623be414519ee9ca9")
	checksum = PasswordChecksum("testpw", "test")
	test.IsEqualString(t, checksum, "41d1781205837071affbf2268588b3f2e755f0365cfe16aff6136155c1013029")
	checksum = PasswordChecksum("test", "test")
	test.IsEqualString(t, checksum, "a3325e881a99e897aab8ba1de274803cddd4f035409c98e976fec9b8005694e6")
	checksum = PasswordChecksum("test", "testsalt")
	test.IsEqualString(t, checksum, "2dbcdfd0989dd2e1be0eea54f176c102e891fd4cb8182544fa4c9dba45307846")
}
