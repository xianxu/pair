package vt

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Frozen pre-optimization transition: keep independent of the shortcut so
// arbitrary parser chunking and state sequences have a slow semantic oracle.
func slowPairHandlePrint(e *Emulator, r rune) {
	next := string(r)
	if len(next) > e.limits.GraphemeBytes {
		e.flushGrapheme()
		return
	}
	if e.cluster != "" {
		joined := e.cluster + next
		first, width := ansi.FirstGraphemeCluster(joined, ansi.GraphemeWidth)
		if len(first) == len(joined) {
			if len(joined) > e.limits.GraphemeBytes {
				e.clusterDropping = true
				return
			}
			if e.clusterDropping {
				return
			}
			e.cluster = joined
			if width == e.clusterWidth {
				c := *e.scr.CellAt(e.clusterX, e.clusterY)
				c.Content = joined
				e.scr.SetCell(e.clusterX, e.clusterY, &c)
			} else {
				// Undo the provisional cell before a width-changing extension. Wrapping
				// happens once the extended cluster's width is known, including scrolling.
				e.scr.SetCell(e.clusterX, e.clusterY, &e.clusterOld[0])
				if e.clusterX+1 < e.scr.Width() {
					e.scr.SetCell(e.clusterX+1, e.clusterY, &e.clusterOld[1])
				}
				e.scr.setCursor(e.clusterX, e.clusterY, false)
				e.atPhantom = false
				e.handleGrapheme(joined, width)
			}
			return
		}
	}
	_, width := ansi.FirstGraphemeCluster(next, ansi.GraphemeWidth)
	if width == 0 {
		// Controls seal the provisional cluster. Without an open printable
		// predecessor, consume non-attaching zero-width input without creating
		// a nonempty continuation cell or changing cursor/pending-wrap state.
		// This follows the native Zellij orphan policy; valid open clusters
		// still accept combining marks, joiners and selectors above.
		e.flushGrapheme()
		return
	}
	e.clusterDropping = false
	e.cluster = next
	e.handleGrapheme(next, width)
}

func pairSlowPrint(e *Emulator) {
	pairPrintHandler(e, func(r rune) { slowPairHandlePrint(e, r) })
}
func pairPrintHandler(e *Emulator, printRune func(rune)) {
	e.parser.SetHandler(ansi.Handler{
		Print:   printRune,
		Execute: e.handleControl, HandleCsi: e.handleCsi, HandleEsc: e.handleEsc,
		HandleDcs: e.handleDcs, HandleOsc: e.handleOsc, HandleApc: e.handleApc,
		HandlePm: e.handlePm, HandleSos: e.handleSos,
	})
}

func TestPairPrintableASCIIBoundaryOracle(t *testing.T) {
	for a := byte(0x20); a <= 0x7e; a++ {
		for b := byte(0x20); b <= 0x7e; b++ {
			first, width := ansi.FirstGraphemeCluster(string([]byte{a, b}), ansi.GraphemeWidth)
			if first != string(a) || width != 1 {
				t.Fatalf("%x,%x: %q width%d", a, b, first, width)
			}
		}
	}
}

type pairASCIIObservation struct {
	replies bytes.Buffer
	effects []string
}

func pairObserved(e *Emulator) *pairASCIIObservation {
	o := new(pairASCIIObservation)
	e.SetReplyWriter(&o.replies)
	e.SetCallbacks(Callbacks{
		Bell:           func() { o.effects = append(o.effects, "bell") },
		Title:          func(s string) { o.effects = append(o.effects, "title:"+s) },
		Notification:   func(a, b string) { o.effects = append(o.effects, "notice:"+a+":"+b) },
		ClipboardWrite: func(a string, b []byte) { o.effects = append(o.effects, "clip:"+a+":"+string(b)) },
	})
	return o
}
func pairSameASCIIState(t *testing.T, a, b *Emulator, ao, bo *pairASCIIObservation) {
	t.Helper()
	for i := range a.scrs {
		if !reflect.DeepEqual(a.scrs[i].buf.Lines, b.scrs[i].buf.Lines) ||
			!reflect.DeepEqual(a.scrs[i].rows, b.scrs[i].rows) ||
			!reflect.DeepEqual(a.scrs[i].cur, b.scrs[i].cur) ||
			!reflect.DeepEqual(a.scrs[i].saved, b.scrs[i].saved) {
			t.Fatalf("screen%d differs: got%q want%q", i, a.String(), b.String())
		}
	}
	if !reflect.DeepEqual(a.HistorySnapshot(), b.HistorySnapshot()) ||
		!reflect.DeepEqual(a.modes, b.modes) || !reflect.DeepEqual(a.keyboard, b.keyboard) ||
		a.screenIndex() != b.screenIndex() || a.atPhantom != b.atPhantom || a.lastChar != b.lastChar ||
		a.cluster != b.cluster || a.clusterWidth != b.clusterWidth || a.clusterX != b.clusterX || a.clusterY != b.clusterY ||
		a.clusterDropping != b.clusterDropping || !reflect.DeepEqual(a.clusterOld, b.clusterOld) ||
		a.gl != b.gl || a.gr != b.gr || a.gsingle != b.gsingle || !reflect.DeepEqual(a.charsets, b.charsets) ||
		a.title != b.title || a.cwd != b.cwd || ao.replies.String() != bo.replies.String() || !reflect.DeepEqual(ao.effects, bo.effects) {
		t.Fatalf("state differs: cluster%q/%q, wrap%v/%v replies%q/%q effects%q/%q", a.cluster, b.cluster, a.atPhantom, b.atPhantom, ao.replies.String(), bo.replies.String(), ao.effects, bo.effects)
	}
}
func pairASCIIEmulators(t *testing.T, w, h, limit int) (*Emulator, *Emulator, *pairASCIIObservation, *pairASCIIObservation) {
	t.Helper()
	l := DefaultLimits()
	l.GraphemeBytes = limit
	a, err := NewEmulatorWithLimits(w, h, l)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewEmulatorWithLimits(w, h, l)
	if err != nil {
		t.Fatal(err)
	}
	pairSlowPrint(b)
	return a, b, pairObserved(a), pairObserved(b)
}
func TestPairASCIIPrintDifferentialPartitions(t *testing.T) {
	streams := []string{
		"ABe\u0301Z1\ufe0f\u20e3X", "AB\u0600C\u0301D", "AB👩\u200d💻🇺🇸❤\ufe0fZ",
		"\x1b[2;4HAB1\ufe0f\u20e3X\x1b[6n\x1b[c", "\x1b(0lqqk\x1b(BAB\u0301\x1b[6n",
		"AB" + strings.Repeat("\u0301", 10) + "CD", "AB\x1b[?1049hCD\x1b[?1049lEF\x1b[?7lGH\x1b[?7h",
		"AB\x1b]0;title\aCD\a\x1b]52;c;eA==\aEF\x1b]777;notify;t;b\a",
		"AB\xff\xc0\x80Z\r\nCD\bE\tF", strings.Repeat("ABCD\r\n", 8),
	}
	for si, stream := range streams {
		for _, width := range []int{1, 4, 8} {
			for _, limit := range []int{1, 4, 256} {
				t.Run(fmt.Sprintf("stream%d/w%d/limit%d", si, width, limit), func(t *testing.T) {
					for split := 0; split <= len(stream); split++ {
						a, b, ao, bo := pairASCIIEmulators(t, width, 2, limit)
						a.WriteString(stream[:split])
						b.WriteString(stream[:split])
						pairSameASCIIState(t, a, b, ao, bo)
						a.WriteString(stream[split:])
						b.WriteString(stream[split:])
						pairSameASCIIState(t, a, b, ao, bo)
						a.Resize(3, 3)
						b.Resize(3, 3)
						pairSameASCIIState(t, a, b, ao, bo)
						a.WriteString("AB1\ufe0f\u20e3\x1b[6n")
						b.WriteString("AB1\ufe0f\u20e3\x1b[6n")
						pairSameASCIIState(t, a, b, ao, bo)
						a.Close()
						b.Close()
					}
				})
			}
		}
	}
}
func TestPairASCIIPrintGeneratedSequences(t *testing.T) {
	tokens := []string{"ABCD", "1\ufe0f\u20e3", "e\u0301", "\u0600A", "👩\u200d💻", "\r\n", "\x1b[H", "\x1b[2J", "\x1b[6n", "\x1b[?1049h", "\x1b[?1049l", "\x1b(0q\x1b(B", "\x1b[?7l", "\x1b[?7h", "\x1b]0;x\a", "\a"}
	for seed := int64(0); seed < 32; seed++ {
		r := rand.New(rand.NewSource(seed))
		a, b, ao, bo := pairASCIIEmulators(t, 4, 3, 8)
		for step := 0; step < 40; step++ {
			s := tokens[r.Intn(len(tokens))]
			for len(s) > 0 {
				n := min(len(s), 1+r.Intn(5))
				a.WriteString(s[:n])
				b.WriteString(s[:n])
				pairSameASCIIState(t, a, b, ao, bo)
				s = s[n:]
			}
			if step%11 == 0 {
				w, h := 1+r.Intn(8), 1+r.Intn(4)
				a.Resize(w, h)
				b.Resize(w, h)
				pairSameASCIIState(t, a, b, ao, bo)
			}
		}
		a.Close()
		b.Close()
	}
}

func BenchmarkPairPrintRedraw(b *testing.B) {
	for _, tc := range []struct{ name, payload string }{
		{"ascii", "\x1b[H" + strings.Repeat("ordinary ASCII status 0123456789 ", 60)},
		{"mixed", "\x1b[H" + strings.Repeat("status e\u0301 界 👩\u200d💻 1\ufe0f\u20e3 \x1b[32mOK\x1b[0m ", 30)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			e := NewEmulator(80, 24)
			defer e.Close()
			e.SetReplyWriter(&bytes.Buffer{})
			b.SetBytes(int64(len(tc.payload)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				e.WriteString(tc.payload)
			}
		})
	}
}

// Faulty test-only transitions qualify the oracle against lost amendment and
// a shortcut incorrectly broadened to Unicode Prepend predecessors.
func TestPairASCIIOracleRejectsMutants(t *testing.T) {
	for _, tc := range []struct {
		name, stream string
		mutant       func(*Emulator, rune)
	}{
		{"seals-every-print", "AB\u0301", func(e *Emulator, r rune) { slowPairHandlePrint(e, r); e.flushGrapheme() }},
		{"non-ASCII-predecessor", "AB\u0600C", func(e *Emulator, r rune) {
			if r >= 0x20 && r <= 0x7e {
				e.flushGrapheme()
			}
			slowPairHandlePrint(e, r)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b, _, _ := pairASCIIEmulators(t, 8, 2, 256)
			defer a.Close()
			defer b.Close()
			pairPrintHandler(a, func(r rune) { tc.mutant(a, r) })
			a.WriteString(tc.stream)
			b.WriteString(tc.stream)
			if reflect.DeepEqual(a.scr.buf.Lines, b.scr.buf.Lines) {
				t.Fatal("slow oracle did not distinguish faulty transition")
			}
		})
	}
}
