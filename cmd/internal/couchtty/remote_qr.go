package couchtty

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"
)

// remoteLinkQR renders a local-only QR beside the console's text link when it
// fits. Each terminal cell contains two square modules, with explicit black on
// white colors and the encoder's four-module quiet zone. The caller owns the
// trusted-console boundary; these rows contain the same capability as the link.
// Nil asks the caller to retain its text-only presentation.
func remoteLinkQR(link string, width, height int) []string {
	// Even a version-one code needs 21 modules plus its quiet zone. Bound the
	// input before parsing or encoding, including when terminal sizes are huge.
	if len(link) == 0 || len(link) > 1024 || width < 29 || height < 15 {
		return nil
	}
	if !utf8.ValidString(link) || strings.IndexFunc(link, unicode.IsControl) >= 0 {
		return nil
	}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil
	}
	code, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		return nil
	}
	bitmap := code.Bitmap()
	size := len(bitmap)
	// QR version 40 is at most 177 modules, plus eight quiet-zone modules.
	if size == 0 || size > 185 || size > width || (size+1)/2 > height {
		return nil
	}
	rows := make([]string, (size+1)/2)
	for y := 0; y < size; y += 2 {
		var row strings.Builder
		row.Grow(size*3 + len("\x1b[38;2;0;0;0;48;2;255;255;255m\x1b[0m"))
		row.WriteString("\x1b[38;2;0;0;0;48;2;255;255;255m")
		for x := 0; x < size; x++ {
			top := bitmap[y][x]
			bottom := y+1 < size && bitmap[y+1][x]
			switch {
			case top && bottom:
				row.WriteRune('█')
			case top:
				row.WriteRune('▀')
			case bottom:
				row.WriteRune('▄')
			default:
				row.WriteByte(' ')
			}
		}
		row.WriteString("\x1b[0m")
		rows[y/2] = row.String()
	}
	return rows
}
