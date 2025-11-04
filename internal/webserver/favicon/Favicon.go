package favicon

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"strings"

	"github.com/Kodeworks/golang-image-ico"
	"github.com/forceu/gokapi/internal/helper"
	"golang.org/x/image/draw"
)

var faviconIco []byte

var faviconPng16x16 []byte

var faviconPng32x32 []byte
var faviconPng180x180 []byte

var faviconPng192x192 []byte

var faviconPng512x512 []byte

func Init(pathCustomIcon string, fsDefault fs.FS) {
	var imageContent []byte
	exists, err := helper.FileExists(pathCustomIcon)
	helper.Check(err)
	if exists {
		content, err := os.ReadFile(pathCustomIcon)
		helper.Check(err)
		imageContent = content
	} else {
		content, err := fsDefault.Open("defaultFavicon.png")
		helper.Check(err)
		defer content.Close()
		imageContent, err = io.ReadAll(content)
		helper.Check(err)
	}
	img, _, err := image.Decode(bytes.NewReader(imageContent))
	if err != nil {
		fmt.Println(err)
		fmt.Println("Could not decode favicon, please make sure to supply a 512x512 png image.")
		os.Exit(1)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 512 || bounds.Dy() != 512 {
		fmt.Println("Could not decode favicon, please make sure to supply a 512x512 png image.")
		os.Exit(1)
	}

	faviconIco = scaleImage(img, 48, false)

	faviconPng16x16 = scaleImage(img, 16, true)
	faviconPng32x32 = scaleImage(img, 32, true)
	faviconPng180x180 = scaleImage(img, 180, true)
	faviconPng192x192 = scaleImage(img, 192, true)
	faviconPng512x512 = imageContent
}

func GetFavicon(url string) []byte {
	if strings.HasPrefix(url, "/favicon.ico") {
		return faviconIco
	}
	if strings.HasPrefix(url, "/favicon-16x16.png") {
		return faviconPng16x16
	}
	if strings.HasPrefix(url, "/favicon-32x32.png") {
		return faviconPng32x32
	}
	if strings.HasPrefix(url, "/favicon-android-chrome-192x192.png") {
		return faviconPng192x192
	}
	if strings.HasPrefix(url, "/favicon-android-chrome-512x512.png") {
		return faviconPng512x512
	}
	if strings.HasPrefix(url, "/favicon-apple-touch-icon.png") {
		return faviconPng180x180
	}
	return faviconIco
}

func scaleImage(src image.Image, size int, isPng bool) []byte {
	buf := new(bytes.Buffer)
	ratio := (float64)(src.Bounds().Max.Y) / (float64)(src.Bounds().Max.X)
	height := int(math.Round(float64(size) * ratio))
	dst := image.NewRGBA(image.Rect(0, 0, size, height))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	if isPng {
		err := png.Encode(buf, dst)
		helper.Check(err)
		return buf.Bytes()
	}
	err := ico.Encode(buf, dst)
	helper.Check(err)
	return buf.Bytes()
}
