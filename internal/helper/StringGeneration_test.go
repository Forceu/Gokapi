package helper

import (
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestByteCountSI(t *testing.T) {
	test.IsEqualString(t, ByteCountSI(5), "5 B")
	test.IsEqualString(t, ByteCountSI(5000), "4.9 kB")
	test.IsEqualString(t, ByteCountSI(5000000), "4.8 MB")
	test.IsEqualString(t, ByteCountSI(5000000000), "4.7 GB")
	test.IsEqualString(t, ByteCountSI(5000000000000), "4.5 TB")
}

func TestCleanString(t *testing.T) {
	test.IsEqualString(t, cleanRandomString("abc-123%%___!"), "abc123")
}

func TestGenerateRandomString(t *testing.T) {
	test.IsEqualBool(t, len(GenerateRandomString(100)) == 100, true)
}

func TestGenerateUnsafeId(t *testing.T) {
	test.IsEqualBool(t, len(generateUnsafeId(100)) == 100, true)
}
