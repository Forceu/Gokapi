package cliflags

import (
	"os"
	"testing"
)

func TestGetUploadParametersTmpFolder(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	for _, flag := range []string{"-t", "--tmpfolder", "--tempfolder"} {
		os.Args = []string{"gokapi-cli", "upload-dir", "-D", "/upload", flag, "/custom/tmp"}
		result := GetUploadParameters(ModeArchive)
		if result.TmpFolder != "/custom/tmp" {
			t.Errorf("%s: TmpFolder = %q, want %q", flag, result.TmpFolder, "/custom/tmp")
		}
	}
}
