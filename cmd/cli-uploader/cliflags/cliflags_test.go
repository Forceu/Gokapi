//go:build test

package cliflags

import (
	"os"
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

func TestGetUploadParametersExpiryDays(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"gokapi-cli", "upload", "-f", "file"}
	result := GetUploadParameters(ModeUpload)
	test.IsEqualInt(t, result.ExpiryDays, 0)
	test.IsEqualBool(t, result.IsExpiryDaysSet, false)

	os.Args = []string{"gokapi-cli", "upload", "-f", "file", "--expiry-days", "0"}
	result = GetUploadParameters(ModeUpload)
	test.IsEqualInt(t, result.ExpiryDays, 0)
	test.IsEqualBool(t, result.IsExpiryDaysSet, true)

	os.Args = []string{"gokapi-cli", "upload", "-f", "file", "-e", "7"}
	result = GetUploadParameters(ModeUpload)
	test.IsEqualInt(t, result.ExpiryDays, 7)
	test.IsEqualBool(t, result.IsExpiryDaysSet, true)
}
