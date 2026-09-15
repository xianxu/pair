package vt

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestPairParserRetentionAndRecovery(t *testing.T) {
	e := NewEmulator(8, 2)
	defer e.Close()
	e.SetReplyWriter(io.Discard)
	var titles []string
	e.SetCallbacks(Callbacks{Title: func(s string) { titles = append(titles, s) }})
	e.WriteString("\x1b]2;")
	chunk := []byte(strings.Repeat("x", 1024))
	for i := 0; i < 2048; i++ {
		e.Write(chunk)
	}
	if got := cap(e.parser.Data()); got != DefaultLimits().StringBytes+1 {
		t.Fatalf("string backing capacity %d", got)
	}
	if len(e.parser.Data()) != DefaultLimits().StringBytes+1 {
		t.Fatal("overflow sentinel not retained")
	}
	e.WriteString("\a\x1b]2;ok\aZ")
	if len(titles) != 1 || titles[0] != "ok" || e.CellAt(0, 0).Content != "Z" {
		t.Fatalf("overflow/recovery %q %q", titles, e.String())
	}
	for _, fragment := range []string{strings.Repeat("9", 1024), strings.Repeat("1;", 512)} {
		e.WriteString("\x1b[")
		for i := 0; i < 1024; i++ {
			e.WriteString(fragment)
		}
		if len(e.parser.Params()) > 32 {
			t.Fatal("unbounded parameters")
		}
		e.WriteString("m")
	}
	before := len(e.modes)
	for i := 0; i < 5000; i++ {
		e.WriteString(fmt.Sprintf("\x1b[?%dh", 100000+i))
	}
	if len(e.modes) != before {
		t.Fatal("unknown modes retained")
	}
}
func TestPairLargeEditingCounts(t *testing.T) {
	for _, final := range []byte{'@', 'P', 'L', 'M', 'S', 'T', 'X'} {
		e := NewEmulator(4, 2)
		e.WriteString("ABCD\x1b[H")
		e.WriteString(fmt.Sprintf("\x1b[100000000%c", final))
		if e.scr.ScrollRegion() != e.scr.Bounds() {
			t.Fatal("editing altered margins")
		}
		e.Close()
	}
}
