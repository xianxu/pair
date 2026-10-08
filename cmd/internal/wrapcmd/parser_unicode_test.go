package wrapcmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestTerminalControlObserverUnicodeStringContainment(t *testing.T) {
	for _, family := range []struct{ name, start, end string }{
		{"OSC", "\x1b]9;", "\a"},
		{"DCS", "\x1bPq", "\x1b\\"},
		{"SOS", "\x1bX", "\x1b\\"},
		{"PM", "\x1b^", "\x1b\\"},
		{"APC", "\x1b_", "\x1b\\"},
	} {
		for _, padding := range []int{0, terminalControlDataMax + 1} {
			for _, mode := range []string{"?25l", "?1049h"} {
				t.Run(fmt.Sprintf("%s/padding%d/%s", family.name, padding, mode), func(t *testing.T) {
					// ✛ is E2 9C 9B in UTF-8. A premature exit at its 0x9c
					// continuation exposes the final 0x9b as a C1 CSI.
					wire := family.start + strings.Repeat("x", padding) + "✛" + mode + family.end
					for cut := 0; cut <= len(wire); cut++ {
						var observer terminalControlObserver
						observer.Feed([]byte("\x1b[?25h"))
						observer.Feed([]byte(wire[:cut]))
						observer.Feed([]byte(wire[cut:]))
						if !observer.visible {
							t.Fatalf("split %d: string payload changed cursor visibility", cut)
						}
						observer.Feed([]byte("\x1b[?25l"))
						if observer.visible {
							t.Fatalf("split %d: real CSI after string was swallowed", cut)
						}
					}
					var bytewise terminalControlObserver
					bytewise.Feed([]byte("\x1b[?25h"))
					for i := range len(wire) {
						bytewise.Feed([]byte{wire[i]})
						if !bytewise.visible {
							t.Fatalf("byte %d: string payload changed cursor visibility", i)
						}
					}
				})
			}
		}
	}
}

func TestOutputBoundaryUnicodeStringContainment(t *testing.T) {
	for _, family := range []struct{ name, start, end string }{
		{"OSC", "\x1b]9;", "\a"},
		{"DCS", "\x1bPq", "\x1b\\"},
		{"SOS", "\x1bX", "\x1b\\"},
		{"PM", "\x1b^", "\x1b\\"},
		{"APC", "\x1b_", "\x1b\\"},
	} {
		t.Run(family.name, func(t *testing.T) {
			var boundary outputBoundary
			wire := family.start + "✻ Crunched ✛?25l" + family.end
			for i := range len(wire) {
				boundary.advance(wire[i])
				if got, want := boundary.safe(), i == len(wire)-1; got != want {
					t.Fatalf("byte %d: safe = %v, want %v", i, got, want)
				}
			}
			if boundary.disabled {
				t.Fatal("Unicode string permanently disabled notification insertion")
			}
		})
	}
}
