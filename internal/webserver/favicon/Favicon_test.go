package favicon

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"
	"testing/fstest"

	"github.com/forceu/gokapi/internal/test"
)

// generateTestImage creates a valid 512x512 PNG in memory for testing
func generateTestImage(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInitAndGetFavicon(t *testing.T) {
	imageData := generateTestImage(t)

	// 1. Setup Mock FS for default icon
	mockFS := fstest.MapFS{
		"defaultFavicon.png": &fstest.MapFile{Data: imageData},
	}

	// 2. Setup a temporary file for the "custom" icon
	tmpFile, err := os.CreateTemp("", "custom_icon*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.Write(imageData)
	tmpFile.Close()

	t.Run("Initialize with default icon", func(t *testing.T) {
		// Pass a non-existent path to force use of fsDefault
		Init("non_existent_path.png", mockFS)

		// Verify various sizes
		icoRes := GetFavicon("/favicon.ico")
		test.IsEqualBool(t, len(icoRes) > 0, true)

		png16 := GetFavicon("/favicon-16x16.png")
		test.IsEqualBool(t, len(png16) > 0, true)

		png512 := GetFavicon("/favicon-android-chrome-512x512.png")
		test.IsEqualInt(t, len(png512), len(imageData))
	})

	t.Run("Initialize with custom icon", func(t *testing.T) {
		Init(tmpFile.Name(), mockFS)

		// Verify apple touch icon (180x180)
		appleIcon := GetFavicon("/favicon-apple-touch-icon.png")
		test.IsEqualBool(t, len(appleIcon) > 0, true)

		// Verify fallback to ICO
		fallback := GetFavicon("/unknown-path")
		test.IsEqualInt(t, len(fallback), len(faviconIco))
	})
}

func TestScaleImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 512, 512))

	t.Run("Scale to PNG", func(t *testing.T) {
		data := scaleImage(src, 32, true)
		img, err := png.Decode(bytes.NewReader(data))
		test.IsNil(t, err)
		test.IsEqualInt(t, img.Bounds().Dx(), 32)
		test.IsEqualInt(t, img.Bounds().Dy(), 32)
	})

	t.Run("Scale to ICO", func(t *testing.T) {
		data := scaleImage(src, 48, false)
		// Basic check for ICO header (00 00 01 00)
		test.IsEqualBool(t, len(data) > 4, true)
		test.IsEqualInt(t, int(data[2]), 1)
	})
}
