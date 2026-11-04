package thumbnail

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"

	"io"
)

const (
	// targetWidth is the output width of the resulting thumbnail image
	targetWidth = 600
	// targetHeight is the output height of the resulting thumbnail image
	targetHeight = 315
)

// ProcessMinify loads an image (PNG or JPEG) and minifies it to a
func ProcessMinify(r io.Reader, w io.Writer) error {
	img, err := decode(r)
	if err != nil {
		return err
	}

	outImg := process(img)
	return jpeg.Encode(w, outImg, &jpeg.Options{Quality: 70})
}

func decode(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

func process(img image.Image) *image.RGBA {
	// flatten onto white
	flattened := flattenToWhite(img)

	// crop center to target aspect ratio
	cropped := cropCenterAspect(flattened, float64(targetWidth)/float64(targetHeight))

	// resize
	resized := resizeNearest(cropped, targetWidth, targetHeight)

	return resized
}

// flattenToWhite fills transparent areas with white
func flattenToWhite(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)

	// fill with white background
	draw.Draw(dst, b, &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	// draw original image on top
	draw.Draw(dst, b, src, b.Min, draw.Over)

	return dst
}

// cropCenterAspect crops to the center in the correct aspect ratio
func cropCenterAspect(img image.Image, aspect float64) image.Image {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	current := float64(w) / float64(h)

	var newW, newH int

	if current > aspect {
		newH = h
		newW = int(float64(h) * aspect)
	} else {
		newW = w
		newH = int(float64(w) / aspect)
	}

	startX := (w - newW) / 2
	startY := (h - newH) / 2

	rect := image.Rect(startX, startY, startX+newW, startY+newH)

	return img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)
}

// resizeNearest nearest-neighbor resize
func resizeNearest(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	sb := src.Bounds()
	srcW := sb.Dx()
	srcH := sb.Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcX := sb.Min.X + x*srcW/w
			srcY := sb.Min.Y + y*srcH/h
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	return dst
}
