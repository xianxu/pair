package broadcast

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// jetBrainsMonoAdvance is JetBrains Mono's advance at its 1000 units per em;
// symbols.py reads it from the packed font when it builds the subset.
const jetBrainsMonoAdvance, jetBrainsMonoUPM = 600, 1000

// symbolList is vendor/fonts/symbols.txt, the code points the symbol font
// supplies (#415).
func symbolList(t *testing.T) []rune {
	t.Helper()
	var list []rune
	b, err := os.ReadFile("web/vendor/fonts/symbols.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "U+") {
			continue
		}
		cp, err := strconv.ParseUint(strings.Fields(line)[0][2:], 16, 32)
		if err != nil {
			t.Fatalf("symbols.txt: %q: %v", line, err)
		}
		list = append(list, rune(cp))
	}
	if len(list) == 0 || len(list) > 100 {
		t.Fatalf("symbols.txt lists %d code points", len(list))
	}
	return list
}

// TestSymbolFontCoversExactlyTheList: the subset maps exactly the listed
// code points, and the stylesheet asks for it for exactly those.
func TestSymbolFontCoversExactlyTheList(t *testing.T) {
	list := symbolList(t)
	font := parseWOFF(t, mustAsset(t, "vendor/fonts/NotoSansSymbols2-Couch.woff"))
	if got := slices.Sorted(func(yield func(rune) bool) {
		for r := range font.cmap {
			if !yield(r) {
				return
			}
		}
	}); !slices.Equal(got, list) {
		t.Fatalf("font maps %U, symbols.txt lists %U", got, list)
	}
	css := string(mustAsset(t, "viewer.css"))
	m := regexp.MustCompile(`(?s)font-family: "Couch Symbols";.*?unicode-range: ([^;]*);`).FindStringSubmatch(css)
	if m == nil {
		t.Fatal("viewer.css has no unicode-range for Couch Symbols")
	}
	var want []string
	for _, r := range list {
		want = append(want, fmt.Sprintf("U+%04X", r))
	}
	if m[1] != strings.Join(want, ", ") {
		t.Fatalf("unicode-range %q, want %q", m[1], strings.Join(want, ", "))
	}
}

// TestViewerPreloadsSymbolFont: the face loads before the first frame, with
// a sample it actually covers. xterm.js caches each character's measured
// width on first draw, so a face fetched lazily on first use leaves symbols
// squeezed to no width (#415 iPad smoke).
func TestViewerPreloadsSymbolFont(t *testing.T) {
	viewer := string(mustAsset(t, "viewer.js"))
	m := regexp.MustCompile(`SYMBOL_SAMPLE = '(.)';`).FindStringSubmatch(viewer)
	if m == nil || !strings.Contains(viewer, "document.fonts.load(`16px ${SYMBOLS}`, SYMBOL_SAMPLE)") {
		t.Fatal("viewer.js must load the symbol face up front with SYMBOL_SAMPLE")
	}
	if r := []rune(m[1])[0]; !slices.Contains(symbolList(t), r) {
		t.Fatalf("SYMBOL_SAMPLE %U is not in symbols.txt, so it loads nothing", r)
	}
}

// TestSymbolFontKeepsCouchWidth: every symbol is one column wide in Couch
// and exactly one JetBrains Mono cell wide in the font, so xterm.js lays a
// row out the same with or without it.
func TestSymbolFontKeepsCouchWidth(t *testing.T) {
	font := parseWOFF(t, mustAsset(t, "vendor/fonts/NotoSansSymbols2-Couch.woff"))
	if font.upm != jetBrainsMonoUPM {
		t.Fatalf("units per em %d, want %d", font.upm, jetBrainsMonoUPM)
	}
	for _, r := range symbolList(t) {
		if w := ansi.GraphemeWidth.StringWidth(string(r)); w != 1 {
			t.Errorf("%U %c: Couch counts %d columns, want 1", r, r, w)
		}
		if adv := font.advance(font.cmap[r]); adv != jetBrainsMonoAdvance {
			t.Errorf("%U %c: advance %d, want %d", r, r, adv, jetBrainsMonoAdvance)
		}
	}
}

type woffFont struct {
	upm      uint16
	cmap     map[rune]uint16
	advances []uint16
}

func (f woffFont) advance(glyph uint16) uint16 {
	if int(glyph) < len(f.advances) {
		return f.advances[glyph]
	}
	return f.advances[len(f.advances)-1]
}

// parseWOFF reads the units per em, the Unicode cmap (format 4 or 12) and
// the advances of a WOFF 1.0 font: just enough to check the subset.
func parseWOFF(t *testing.T, b []byte) woffFont {
	t.Helper()
	be := binary.BigEndian
	if string(b[:4]) != "wOFF" {
		t.Fatalf("not a WOFF 1.0 font: %q", b[:4])
	}
	tables := map[string][]byte{}
	for i := range int(be.Uint16(b[12:])) {
		e := b[44+20*i:]
		off, comp, orig := be.Uint32(e[4:]), be.Uint32(e[8:]), be.Uint32(e[12:])
		data := b[off : off+comp]
		if comp < orig {
			z, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if data, err = io.ReadAll(z); err != nil {
				t.Fatal(err)
			}
		}
		tables[string(e[:4])] = data
	}
	f := woffFont{upm: be.Uint16(tables["head"][18:]), cmap: map[rune]uint16{}}
	cmap := tables["cmap"]
	for i := range int(be.Uint16(cmap[2:])) {
		rec := cmap[4+8*i:]
		if be.Uint16(rec) != 3 && be.Uint16(rec) != 0 { // Windows or Unicode
			continue
		}
		st := cmap[be.Uint32(rec[4:]):]
		switch be.Uint16(st) {
		case 4:
			n := int(be.Uint16(st[6:])) / 2
			ends, starts, deltas, offs := st[14:], st[16+2*n:], st[16+4*n:], st[16+6*n:]
			for s := range n {
				for c := int(be.Uint16(starts[2*s:])); c <= int(be.Uint16(ends[2*s:])) && c != 0xFFFF; c++ {
					g := uint16(c) + be.Uint16(deltas[2*s:])
					if ro := int(be.Uint16(offs[2*s:])); ro != 0 {
						g = be.Uint16(offs[2*s+ro+2*(c-int(be.Uint16(starts[2*s:]))):])
						if g != 0 {
							g += be.Uint16(deltas[2*s:])
						}
					}
					if g != 0 {
						f.cmap[rune(c)] = g
					}
				}
			}
		case 12:
			for g := range int(be.Uint32(st[12:])) {
				grp := st[16+12*g:]
				for c := be.Uint32(grp); c <= be.Uint32(grp[4:]); c++ {
					f.cmap[rune(c)] = uint16(be.Uint32(grp[8:]) + c - be.Uint32(grp))
				}
			}
		}
	}
	hmtx := tables["hmtx"]
	for i := range int(be.Uint16(tables["hhea"][34:])) {
		f.advances = append(f.advances, be.Uint16(hmtx[4*i:]))
	}
	return f
}
