package vt

import (
	"bytes"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestPairBytewiseClustersAndBounds(t *testing.T) {
	for _, s := range []string{"é", "👩‍💻", "1️⃣", "❤️", "🇺🇸", "🇺🇸🇨🇦"} {
		e := NewEmulator(12, 2)
		for i := range []byte(s) {
			e.Write([]byte{s[i]})
		}
		if !strings.HasPrefix(e.String(), s) {
			t.Errorf("bytewise %q got %q", s, e.String())
		}
		e.Close()
	}
	e := NewEmulator(12, 2)
	e.WriteString("a" + strings.Repeat("\u0301", 100000) + "Z")
	if len(e.CellAt(0, 0).Content) > 256 || e.CellAt(1, 0).Content != "Z" {
		t.Fatal("cluster bound/recovery")
	}
	e.Close()
}
func TestPairHistoryBounds(t *testing.T) {
	l := DefaultLimits()
	l.HistoryCells = 12
	l.HistoryBytes = 2048
	l.HistoryLines = 4
	e, _ := NewEmulatorWithLimits(8, 2, l)
	defer e.Close()
	for i := 0; i < 100; i++ {
		e.WriteString("12345678\r\n")
	}
	u := e.Usage()
	if u.HistoryCells > 12 || u.HistoryBytes > 2048 || e.ScrollbackLen() > 4 {
		t.Fatalf("history %+v", u)
	}
	e.ClearScrollback()
	if u = e.Usage(); u.HistoryCells != 0 || u.HistoryBytes != 0 {
		t.Fatalf("clear %+v", u)
	}
}
func TestPairStringOverflowEveryBoundary(t *testing.T) {
	for _, size := range []int{31, 32, 33, 1000} {
		for split := 0; split <= 4; split++ {
			l := DefaultLimits()
			l.StringBytes = 32
			l.MetadataBytes = 2000
			e, _ := NewEmulatorWithLimits(4, 2, l)
			var titles []string
			e.SetCallbacks(Callbacks{Title: func(s string) { titles = append(titles, s) }})
			input := "\x1b]2;" + strings.Repeat("x", size-2) + "\x1b\\"
			cut := len(input) - split
			if cut < 0 {
				cut = 0
			}
			e.WriteString(input[:cut])
			e.WriteString(input[cut:])
			e.WriteString("\x1b]2;ok\a")
			want := 1
			if size <= 32 {
				want = 2
			}
			if len(titles) != want || titles[len(titles)-1] != "ok" {
				t.Fatalf("size%d split%d %q", size, split, titles)
			}
			e.Close()
		}
	}
}
func TestPairKeyboardEventsAndAltState(t *testing.T) {
	for _, tc := range []struct {
		flags int
		key   uv.KeyEvent
		want  string
	}{
		{1, uv.KeyPressEvent{Code: uv.KeyUp, Mod: uv.ModAlt}, "\x1b[1;3A"},
		{3, uv.KeyPressEvent{Code: 'a', Mod: uv.ModCtrl, IsRepeat: true}, "\x1b[97;5:2u"},
		{1, uv.KeyReleaseEvent{Code: 'a', Mod: uv.ModCtrl}, ""},
		{8, uv.KeyPressEvent{Code: uv.KeyEnter}, "\x1b[13u"},
		{1, uv.KeyPressEvent{Code: uv.KeyKpEnter}, "\x1b[57414u"},
		{1, uv.KeyPressEvent{Code: uv.KeyMediaPlay}, "\x1b[57428u"},
		{1, uv.KeyPressEvent{Code: uv.KeyEscape}, "\x1b[27u"},
		{1, uv.KeyPressEvent{Code: 'a', Mod: uv.ModShift, Text: "A"}, "A"},
	} {
		e := NewEmulator(4, 2)
		var b bytes.Buffer
		e.SetReplyWriter(&b)
		e.WriteString(fmt.Sprintf("\x1b[>%du", tc.flags))
		e.SendKey(tc.key)
		if b.String() != tc.want {
			t.Errorf("flags %d key %+v: %q want %q", tc.flags, tc.key, b.String(), tc.want)
		}
		e.Close()
	}
	e := NewEmulator(4, 2)
	defer e.Close()
	e.SetReplyWriter(&bytes.Buffer{})
	e.WriteString("\x1b[>1u\x1b[?1049h\x1b[>3u\x1b[?1049l")
	if e.KeyboardFlags() != 1 {
		t.Fatal("alt keyboard leaked")
	}
	for i := 0; i < 100; i++ {
		e.WriteString("\x1b[>1u")
	}
	if len(e.keyboard[0].stack) > 16 {
		t.Fatal("unbounded stack")
	}
	e.WriteString("\x1b[<999999999u")
	if e.KeyboardFlags() != 0 {
		t.Fatal("stack underflow")
	}
}
func TestPairAlternateRepeatedSet(t *testing.T) {
	e := NewEmulator(8, 4)
	defer e.Close()
	e.WriteString("\x1b[2;3H\x1b[?1049h\x1b[4;7H\x1b[?1049h\x1b[?1049lX")
	if e.CellAt(2, 1).Content != "X" {
		t.Fatalf("1049 restore %q", e.String())
	}
	e.WriteString("\x1b[?47hY\x1b[?1047l\x1b[?47h")
	if strings.Contains(e.String(), "Y") {
		t.Fatal("1047 exit must clear")
	}
}
func TestPairMouseReplacementAndHyperlink(t *testing.T) {
	e := NewEmulator(8, 4)
	defer e.Close()
	var b bytes.Buffer
	e.SetReplyWriter(&b)
	e.WriteString("\x1b[?1003h\x1b[?1000h\x1b[?1000l")
	e.SendMouse(uv.MouseClickEvent{Button: uv.MouseLeft})
	if b.Len() != 0 || e.Mode(ansi.ModeMouseAnyEvent).IsSet() {
		t.Fatal("old mouse mode resurrected")
	}
	e.WriteString("\x1b]8;id=x;https://example.com/a;b\aX")
	c := e.CellAt(0, 0)
	if c.Link.URL != "https://example.com/a;b" || c.Link.Params != "id=x" {
		t.Fatalf("link %+v", c.Link)
	}
}

func TestPairNotificationBodyLimit(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	var bodies []string
	e.SetCallbacks(Callbacks{Notification: func(_, body string) { bodies = append(bodies, body) }})
	e.WriteString("\x1b]777;notify;pair;" + strings.Repeat("x", 4096) + "\a")
	if len(bodies) != 1 || len(bodies[0]) != 4096 {
		t.Fatal("canonical max body rejected")
	}
}
func TestPairKeyboardEnhancementFlags(t *testing.T) {
	for _, tc := range []struct {
		flags int
		key   uv.KeyEvent
		want  string
	}{
		{4, uv.KeyPressEvent{Code: 'a', Mod: uv.ModCtrl}, "\x01"},
		{1, uv.KeyPressEvent{Code: uv.KeyEnter, Mod: uv.ModShift}, "\x1b[13;2u"},
		{1, uv.KeyPressEvent{Code: uv.KeyLeftShift}, ""},
		{12, uv.KeyPressEvent{Code: 'a', Mod: uv.ModShift, ShiftedCode: 'A', BaseCode: 'q'}, "\x1b[97:65:113;2u"},
		{24, uv.KeyPressEvent{Code: 'a', Text: "a"}, "\x1b[97;1;97u"},
	} {
		e := NewEmulator(4, 2)
		var b bytes.Buffer
		e.SetReplyWriter(&b)
		e.WriteString(fmt.Sprintf("\x1b[>%du", tc.flags))
		e.SendKey(tc.key)
		if b.String() != tc.want {
			t.Errorf("flags %d key %+v got %q want %q", tc.flags, tc.key, b.String(), tc.want)
		}
		e.Close()
	}
}

func TestPairResizeReleasesCapacity(t *testing.T) {
	e := NewEmulator(100, 100)
	defer e.Close()
	e.WriteString("界")
	if err := e.ResizeChecked(1, 1); err != nil {
		t.Fatal(err)
	}
	for i := range e.scrs {
		b := e.scrs[i].buf
		if cap(b.Lines) != 1 || cap(b.Lines[0]) != 1 {
			t.Fatal("shrink retained old cells")
		}
		if b.CellAt(0, 0).Width > 1 {
			t.Fatal("shrink left clipped wide cell")
		}
	}
}

func TestPairReplyShortWriteAndEffectValidation(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	e.SetReplyWriter(shortWriter{})
	e.SendKey(uv.KeyPressEvent{Code: uv.KeyUp})
	if e.TakeReplyError() == nil {
		t.Fatal("short writer error lost")
	}
	var calls int
	e.SetCallbacks(Callbacks{ClipboardWrite: func(string, []byte) { calls++ }})
	e.WriteString("\x1b]52;bad-target;eA==\a")
	if calls != 0 {
		t.Fatal("invalid clipboard selection reached policy")
	}
}

type shortWriter struct{}

func (shortWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }

func TestPairLargeCursorCommands(t *testing.T) {
	e := NewEmulator(8, 2)
	defer e.Close()
	e.WriteString("a\x1b[100000000I\x1b[100000000Z\x1b[100000000b")
	if e.ScrollbackLen() != 0 {
		t.Fatal("excessive REP executed")
	}
	if e.CursorPosition().X != 0 {
		t.Fatal("tab saturation")
	}
}

func TestPairSmallClusterLimit(t *testing.T) {
	l := DefaultLimits()
	l.GraphemeBytes = 1
	e, err := NewEmulatorWithLimits(4, 2, l)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.WriteString("éA")
	if e.CellAt(0, 0).Content != "A" {
		t.Fatal("first rune exceeded cluster limit")
	}
}

func TestPairOversizedMarginsAreIgnored(t *testing.T) {
	for _, seq := range []string{"\x1b[1;100000000r", "\x1b[?69h\x1b[1;100000000s"} {
		e := NewEmulator(4, 2)
		e.WriteString(seq)
		if e.scr.ScrollRegion() != e.scr.Bounds() {
			t.Fatalf("invalid margins retained: %v", e.scr.ScrollRegion())
		}
		e.WriteString("\x1b[L\x1b[M\x1b[@\x1b[P")
		e.Close()
	}
}
func TestPairCursorStyleRejectsInvalid(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	e.WriteString("\x1b[100000000 q")
	if e.scr.cur.Style != CursorBlock {
		t.Fatal("unknown cursor shape accepted")
	}
}
func TestPairUsageIncludesColorAllocationAllowance(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	before := e.Usage().RetainedBytes
	e.WriteString("\x1b[38;2;1;2;3mA")
	if delta := e.Usage().RetainedBytes - before; delta < 16 {
		t.Fatalf("color payload allowance missing: %d", delta)
	}
}
