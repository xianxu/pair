package assets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // the decoder registers itself
	"image/jpeg"
	"image/png"
	"math"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // the decoder registers itself
)

// bombPixels is the largest canvas an image header may claim before it is
// refused unread: decoding allocates the whole canvas (4 bytes a pixel), so
// a small body claiming a huge one is a memory bomb from the recorded page.
const bombPixels = 64 << 20

// jpegQuality is the resampled photo's quality: no visible loss at the
// viewer's scaled-down size.
const jpegQuality = 85

// Resample fits a raster image into maxPixels (w×h), keeping its aspect
// ratio: an opaque one is stored as JPEG, one with transparency as PNG, and
// orig describes what it was. A body it doesn't change comes back as it
// was, with a nil orig: an image within the budget, one that isn't a raster
// it can decode (SVG, AVIF), and a GIF. An image claiming more than bombPixels is refused.
func Resample(body []byte, mime string, maxPixels int) (out []byte, outMIME string, orig *recording.Orig, err error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return body, mime, nil, nil // not a raster this store reads: exact
	}
	px := cfg.Width * cfg.Height
	if px > bombPixels {
		return nil, "", nil, fmt.Errorf("assets: image claims %dx%d, over the %d px ceiling", cfg.Width, cfg.Height, bombPixels)
	}
	if px <= maxPixels || maxPixels <= 0 {
		return body, mime, nil, nil
	}
	if format == "gif" {
		// Mostly animations, which one frame would lose; and telling needs
		// every frame decoded, past the bomb ceiling's reach. Kept exact.
		return body, mime, nil, nil
	}
	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return body, mime, nil, nil // a header that reads but a body that doesn't: exact
	}
	f := math.Sqrt(float64(maxPixels) / float64(px))
	w, h := max(1, int(float64(cfg.Width)*f)), max(1, int(float64(cfg.Height)*f))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var b bytes.Buffer
	if dst.Opaque() {
		err, outMIME = jpeg.Encode(&b, dst, &jpeg.Options{Quality: jpegQuality}), "image/jpeg"
	} else {
		err, outMIME = png.Encode(&b, dst), "image/png"
	}
	if err != nil {
		return nil, "", nil, fmt.Errorf("assets: encode resampled image: %w", err)
	}
	return b.Bytes(), outMIME, &recording.Orig{Size: int64(len(body)), W: cfg.Width, H: cfg.Height}, nil
}
