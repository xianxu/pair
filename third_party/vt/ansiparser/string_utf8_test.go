package ansiparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestControlStringUTF8Containment(t *testing.T) {
	for _, kind := range []string{"]9;", "Pq", "X", "^", "_"} {
		for _, body := range []string{"✻ Crunched for 12s", "é\u009c尾", "Ü🜀✓", "\xe0\xa0\x9c", "\xed\x9c\x80", "\xf0\x90\x9c\x80", "\xf4\x8f\x9c\xbf"} {
			for _, end := range []string{"\x1b\\", "\x9c", "\a"} {
				if end == "\a" && kind != "]9;" {
					continue
				}
				wire := "\x1b" + kind + body + end + "Z"
				for split := -1; split <= len(wire); split++ {
					t.Run(fmt.Sprintf("%s/%x/%x/%d", kind, body, end, split), func(t *testing.T) {
						p := NewParser()
						var screen strings.Builder
						var payloads []string
						capture := func(b []byte) { payloads = append(payloads, string(b)) }
						p.SetHandler(ansi.Handler{Print: func(r rune) { screen.WriteRune(r) }, HandleOsc: func(_ int, b []byte) { capture(b) }, HandleDcs: func(_ ansi.Cmd, _ ansi.Params, b []byte) { capture(b) }, HandleSos: capture, HandlePm: capture, HandleApc: capture})
						feed := func(s string) {
							for _, b := range []byte(s) {
								p.Advance(b)
							}
						}
						if split < 0 {
							for _, b := range []byte(wire) {
								feed(string([]byte{b}))
							}
						} else {
							feed(wire[:split])
							feed(wire[split:])
						}
						want := body
						if kind == "]9;" {
							want = "9;" + body
						}
						if screen.String() != "Z" || len(payloads) != 1 || payloads[0] != want {
							t.Fatalf("screen=%q payload=%q; want Z, %q", screen.String(), payloads, want)
						}
					})
				}
			}
		}
	}
}

func TestControlStringOverflowRecovery(t *testing.T) {
	for _, kind := range []string{"]9;", "Pq", "X", "^", "_"} {
		p := NewParser()
		p.SetDataSize(8)
		var screen strings.Builder
		p.SetHandler(ansi.Handler{Print: func(r rune) { screen.WriteRune(r) }})
		wire := "\x1b" + kind + strings.Repeat("✻é", 1000) + "\x1b\\Z"
		for _, b := range []byte(wire) {
			p.Advance(b)
		}
		if screen.String() != "Z" || cap(p.Data()) > 8 {
			t.Fatalf("%q screen=%q capacity=%d", kind, screen.String(), cap(p.Data()))
		}
	}
}

func TestControlStringInterruptedUTF8(t *testing.T) {
	for _, kind := range []string{"]9;", "Pq", "X", "^", "_"} {
		for _, prefix := range []string{"\xe2", "\xf0\x90", "\xe0", "\xed", "\xf4"} {
			for _, end := range []string{"\x1b\\", "\x18", "\x1a"} {
				p := NewParser()
				var screen strings.Builder
				p.SetHandler(ansi.Handler{Print: func(r rune) { screen.WriteRune(r) }})
				for _, b := range []byte("\x1b" + kind + prefix + end + "Z") {
					p.Advance(b)
				}
				if screen.String() != "Z" {
					t.Fatalf("kind=%q prefix=%x end=%x screen=%q", kind, prefix, end, screen.String())
				}
				for _, b := range []byte("\x1b" + kind + prefix) {
					p.Advance(b)
				}
				p.Reset()
				p.Advance('Q')
				if screen.String() != "ZQ" {
					t.Fatalf("reset retained UTF8 state: %q", screen.String())
				}
			}
		}
	}
}

func TestInvalidUTF8PrefixDoesNotHideStandaloneST(t *testing.T) {
	for _, prefix := range []string{"\xe0", "\xed\xa0", "\xf0\x8f", "\xf4\x90", "\xc0", "\xf5", "\xe2x"} {
		for _, kind := range []string{"]9;", "Pq", "X", "^", "_"} {
			p := NewParser()
			var screen strings.Builder
			p.SetHandler(ansi.Handler{Print: func(r rune) { screen.WriteRune(r) }})
			for _, b := range []byte("\x1b" + kind + prefix + "\x9cZ") {
				p.Advance(b)
			}
			if screen.String() != "Z" {
				t.Fatalf("%q prefix=%x: screen=%q", kind, prefix, screen.String())
			}
		}
	}
}
