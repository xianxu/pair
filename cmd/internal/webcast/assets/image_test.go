package assets

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math/rand"
	"testing"
)

// noisy is a w×h image of random pixels: incompressible, so a size bound
// on it is not flattered by a compressor.
func noisy(w, h int, opaque bool) image.Image {
	r := rand.New(rand.NewSource(1))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	r.Read(img.Pix)
	if opaque {
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xff
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestResampleBoundsSize is the size bound: a photo far over the budget is
// stored within it, aspect kept, with its original described.
func TestResampleBoundsSize(t *testing.T) {
	src := encodePNG(t, noisy(4000, 3000, true))
	budget := 1920 * 1080
	out, mime, orig, err := Resample(src, "image/png", budget)
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width*cfg.Height > budget {
		t.Errorf("stored %dx%d is over the %d px budget", cfg.Width, cfg.Height, budget)
	}
	if want := cfg.Width * 3 / 4; cfg.Height < want-1 || cfg.Height > want+1 {
		t.Errorf("aspect lost: %dx%d", cfg.Width, cfg.Height)
	}
	if mime != "image/jpeg" || format != "jpeg" {
		t.Errorf("an opaque image is stored as JPEG, got %s (%s)", mime, format)
	}
	if orig == nil || orig.W != 4000 || orig.H != 3000 || orig.Size != int64(len(src)) {
		t.Errorf("orig %+v", orig)
	}
	// JPEG of an incompressible image at the budget: about 3 bytes a pixel
	// at worst. The bound is what keeps a recording's size bounded.
	if max := budget * 3; len(out) > max {
		t.Errorf("stored %d bytes, over %d", len(out), max)
	}
}

func TestResampleKeepsAlpha(t *testing.T) {
	out, mime, orig, err := Resample(encodePNG(t, noisy(300, 200, false)), "image/png", 100*100)
	if err != nil || orig == nil {
		t.Fatalf("%v %+v", err, orig)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil || mime != "image/png" {
		t.Fatalf("an image with alpha is stored as PNG: %s %v", mime, err)
	}
	if b := img.Bounds(); b.Dx()*b.Dy() > 100*100 {
		t.Errorf("over budget: %v", b)
	}
}

func TestResampleExact(t *testing.T) {
	small := encodePNG(t, noisy(40, 30, true))
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="9000" height="9000"/>`)
	var anim bytes.Buffer
	pal := color.Palette{color.Black, color.White}
	frame := image.NewPaletted(image.Rect(0, 0, 3000, 3000), pal)
	gif.EncodeAll(&anim, &gif.GIF{Image: []*image.Paletted{frame, frame}, Delay: []int{10, 10}})
	for _, c := range []struct {
		name, mime string
		body       []byte
	}{
		{"within budget", "image/png", small},
		{"svg", "image/svg+xml", svg},
		{"gif", "image/gif", anim.Bytes()},
		{"undecodable", "image/avif", []byte("not an image I can read")},
	} {
		out, mime, orig, err := Resample(c.body, c.mime, 1000*1000)
		if err != nil || orig != nil || mime != c.mime || !bytes.Equal(out, c.body) {
			t.Errorf("%s must be kept exact: %v %+v %s", c.name, err, orig, mime)
		}
	}
}

// TestResampleRefusesBomb: a header claiming a huge canvas is refused before
// any decode allocates it.
func TestResampleRefusesBomb(t *testing.T) {
	if _, _, _, err := Resample(pngHeader(20000, 20000), "image/png", 1000); err == nil {
		t.Errorf("a 400 MP image must be refused")
	}
}

// pngHeader is a PNG signature and IHDR only: enough for DecodeConfig.
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6
	chunk := append([]byte("IHDR"), ihdr...)
	binary.Write(&b, binary.BigEndian, uint32(13))
	b.Write(chunk)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

// TestResampleWebP: the blog's image pipeline serves WebP.
func TestResampleWebP(t *testing.T) {
	out, mime, orig, err := Resample(webp8x6, "image/webp", 12)
	if err != nil || orig == nil || orig.W != 8 || orig.H != 6 {
		t.Fatalf("webp: %v %+v", err, orig)
	}
	if _, _, err := image.Decode(bytes.NewReader(out)); err != nil || mime == "image/webp" {
		t.Errorf("resampled webp must decode as its new type %s: %v", mime, err)
	}
}

// webp8x6 is an 8×6 lossless WebP (cwebp -lossless of a gradient).
var webp8x6 = []byte{
	0x52, 0x49, 0x46, 0x46, 0x2a, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x4c, 0x1e, 0x00, 0x00, 0x00, 0x2f, 0x07, 0x40, 0x01,
	0x00, 0xb9, 0x8c, 0xe8, 0x7f, 0xec, 0x22, 0xa2, 0xff, 0x01, 0x21, 0x01,
	0x61, 0x8e, 0xff, 0x1b, 0x25, 0x0b, 0xc6, 0x63, 0xb4, 0x62, 0x02, 0x90,
	0x5a, 0x36,
}
