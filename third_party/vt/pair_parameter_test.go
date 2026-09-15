package vt

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

func TestPairParameterOverflowAtomic(t *testing.T) {
	wire := "\x1b[?1002;" + strings.Repeat("1006;", 32) + "1004h"
	for split := 0; split <= len(wire); split++ {
		e := NewEmulator(8, 2)
		var replies bytes.Buffer
		e.SetReplyWriter(&replies)
		e.WriteString(wire[:split])
		e.WriteString(wire[split:])
		if e.Mode(ansi.DECMode(1002)).IsSet() || e.Mode(ansi.DECMode(1006)).IsSet() || e.Mode(ansi.DECMode(1004)).IsSet() || replies.Len() != 0 {
			t.Fatalf("split %d: partial mode effects", split)
		}
		e.WriteString("\x1b[?1004hZ")
		if !e.Mode(ansi.DECMode(1004)).IsSet() || e.CellAt(0, 0).Content != "Z" {
			t.Fatalf("split %d: failed recovery", split)
		}
		e.Close()
	}
}

func TestPairParameterBoundaries(t *testing.T) {
	for _, sep := range []string{";", ":"} {
		for _, count := range []int{parser.MaxParamsSize - 1, parser.MaxParamsSize, parser.MaxParamsSize + 1, parser.MaxParamsSize + 9} {
			for _, dcs := range []bool{false, true} {
				params := strings.TrimSuffix(strings.Repeat("1"+sep, count), sep)
				wire := "\x1b[" + params + "q"
				if dcs {
					wire = "\x1bP" + params + "qpayload\x1b\\"
				}
				t.Run(fmt.Sprintf("%s/%d/dcs%v", sep, count, dcs), func(t *testing.T) {
					for split := 0; split <= len(wire); split++ {
						e := NewEmulator(8, 2)
						e.SetReplyWriter(&bytes.Buffer{})
						calls, got := 0, 0
						e.RegisterCsiHandler('q', func(p ansi.Params) bool { calls++; got = len(p); return true })
						e.RegisterDcsHandler('q', func(p ansi.Params, data []byte) bool { calls++; got = len(p); return true })
						e.WriteString(wire[:split])
						e.WriteString(wire[split:])
						want := 0
						if count <= parser.MaxParamsSize {
							want = 1
						}
						if calls != want || want == 1 && got != count {
							t.Fatalf("split %d: calls=%d params=%d want calls=%d params=%d", split, calls, got, want, count)
						}
						e.WriteString("\x1b[1q")
						if calls != want+1 || got != 1 {
							t.Fatalf("split %d: recovery calls=%d params=%d", split, calls, got)
						}
						e.Close()
					}
				})
			}
		}
	}
}

func TestPairNumericParameterOverflow(t *testing.T) {
	for _, number := range []string{"2147483647", "2147483648", "18446744073709551617", strings.Repeat("9", 100)} {
		for _, prefix := range []string{"\x1b[", "\x1bP"} {
			wire := prefix + "1;" + number + "q"
			if prefix == "\x1bP" {
				wire += "data\x1b\\"
			}
			e := NewEmulator(8, 2)
			calls := 0
			e.RegisterCsiHandler('q', func(ansi.Params) bool { calls++; return true })
			e.RegisterDcsHandler('q', func(ansi.Params, []byte) bool { calls++; return true })
			for i := range wire {
				e.WriteString(wire[i : i+1])
			}
			if calls != 0 {
				t.Fatalf("%q invoked truncated command", wire)
			}
			e.WriteString("\x1b[2147483646q")
			if calls != 1 {
				t.Fatal("maximum representable parameter rejected or recovery failed")
			}
			e.Close()
		}
	}
}

func TestPairParameterOverflowTermination(t *testing.T) {
	for _, intro := range []string{"\x1b[", "\x9b", "\x1bP", "\x90"} {
		for _, ending := range []string{"\x18", "\x1a", "\x1b[1q"} {
			e := NewEmulator(8, 2)
			calls := 0
			e.RegisterCsiHandler('q', func(ansi.Params) bool { calls++; return true })
			e.RegisterDcsHandler('q', func(ansi.Params, []byte) bool { calls++; return true })
			wire := intro + strings.Repeat(";", 32) + ending + "\x1b[1q"
			for i := range wire {
				e.WriteString(wire[i : i+1])
			}
			want := 1
			if ending == "\x1b[1q" {
				want = 2
			}
			if calls != want {
				t.Fatalf("%q calls=%d want=%d", wire, calls, want)
			}
			e.Close()
		}
	}
}

func TestPairEmptyParameterBoundary(t *testing.T) {
	for _, sep := range []string{";", ":"} {
		for _, count := range []int{31, 32} {
			e := NewEmulator(8, 2)
			calls, params := 0, 0
			e.RegisterCsiHandler('q', func(p ansi.Params) bool { calls++; params = len(p); return true })
			e.WriteString("\x1b[" + strings.Repeat(sep, count) + "q")
			if count == 31 && (calls != 1 || params != 32) || count == 32 && calls != 0 {
				t.Fatalf("%q count=%d calls=%d params=%d", sep, count, calls, params)
			}
			e.Close()
		}
	}
}
