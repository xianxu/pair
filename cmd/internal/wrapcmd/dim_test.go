package wrapcmd

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/ansi"
)

func feedAll(d *sgrDimmer, on bool, chunks ...string) string {
	var out bytes.Buffer
	for _, c := range chunks {
		out.Write(d.Feed([]byte(c), on))
	}
	return out.String()
}

func TestSGRDimmer(t *testing.T) {
	const faint = "\x1b[2m"
	for _, tc := range []struct {
		name   string
		on     bool
		chunks []string
		want   string
	}{
		{"off is identity", false, []string{"a\x1b[31mb\x1b[0m"}, "a\x1b[31mb\x1b[0m"},
		{"on prefixes faint", true, []string{"abc"}, faint + "abc"},
		{"color keeps faint", true, []string{"\x1b[31mx"}, faint + "\x1b[31m" + faint + "x"},
		{"reset re-asserts faint", true, []string{"\x1b[0mx\x1b[my"}, faint + "\x1b[0m" + faint + "x\x1b[m" + faint + "y"},
		{"truecolor subparams", true, []string{"\x1b[38:2::1:2:3m"}, faint + "\x1b[38:2::1:2:3m" + faint},
		{"private-marker m untouched", true, []string{"\x1b[>4;2m"}, faint + "\x1b[>4;2m"},
		{"non-SGR CSI untouched", true, []string{"\x1b[?25h\x1b[2J\x1b[3;4H"}, faint + "\x1b[?25h\x1b[2J\x1b[3;4H"},
		{"OSC untouched", true, []string{"\x1b]0;title\x07x"}, faint + "\x1b]0;title\x07x"},
		{"ST-terminated OSC split at ESC", true, []string{"\x1b]52;c;aGk=\x1b", "\\\x1b[1mx"}, faint + "\x1b]52;c;aGk=\x1b\\\x1b[1m" + faint + "x"},
		{"SGR split across feeds", true, []string{"a\x1b[3", "1mb"}, faint + "a\x1b[31m" + faint + "b"},
		{"lone ESC split", true, []string{"a\x1b", "[1mb"}, faint + "a\x1b[1m" + faint + "b"},
		{"utf8 passes", true, []string{"héllo ─"}, faint + "héllo ─"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var d sgrDimmer
			if got := feedAll(&d, tc.on, tc.chunks...); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSGRDimmerTransitions(t *testing.T) {
	// A transition never lands inside the agent's own split CSI: the sequence
	// finishes under the mode it started in, and the transition follows it.
	// No byte is ever held back.
	var d sgrDimmer
	if got := string(d.Feed([]byte("a\x1b[3"), true)); got != "\x1b[2ma\x1b[3" {
		t.Fatalf("split tail must be emitted at once: %q", got)
	}
	if got, want := string(d.Feed([]byte("1mb"), false)), "1m\x1b[2m\x1b[22mb"; got != want {
		t.Fatalf("on→off mid-CSI: got %q want %q", got, want)
	}
	d = sgrDimmer{}
	got := feedAll(&d, false, "a\x1b[3")
	got += string(d.Feed([]byte("1mb"), true))
	if want := "a\x1b[31m\x1b[2mb"; got != want {
		t.Fatalf("off→on mid-CSI: got %q want %q", got, want)
	}
	if got := string(d.Feed([]byte("c"), true)); got != "c" {
		t.Fatalf("steady on: %q", got)
	}
	// A transition with no output still takes effect at the boundary.
	if got := string(d.Feed(nil, false)); got != "\x1b[22m" {
		t.Fatalf("empty off feed: %q", got)
	}
	// An over-long unterminated CSI is emitted and no longer tracked.
	d = sgrDimmer{}
	long := "\x1b[" + strings.Repeat("1;", dimPendingMax/2+1)
	if got := string(d.Feed([]byte(long), true)); got != "\x1b[2m"+long {
		t.Fatalf("over-long tail: %d bytes out", len(got))
	}
	if len(d.tail) != 0 {
		t.Fatal("over-long tail still tracked")
	}
}

// The class behind the transition bug: output must not depend on where the pty
// read boundaries fall. For every split point and both modes, two feeds equal
// one; and a mode flip at any split point lands at the first sequence boundary
// at or after it, never inside a sequence.
func TestSGRDimmerChunkInvariance(t *testing.T) {
	stream := "x\x1b[31mred\x1b[0m\x1b]0;t\x07\x1b[?25h\x1b]52;c;aGk=\x1b\\\x1b[38:2::1:2:3mz\x1b[>4;2m\x1b[m end"
	for _, on := range []bool{false, true} {
		var whole sgrDimmer
		want := string(whole.Feed([]byte(stream), on))
		for i := 0; i <= len(stream); i++ {
			var d sgrDimmer
			if got := feedAll(&d, on, stream[:i], stream[i:]); got != want {
				t.Fatalf("on=%v split=%d: got %q want %q", on, i, got, want)
			}
		}
	}
	for i := 0; i <= len(stream); i++ {
		var d sgrDimmer
		got := string(d.Feed([]byte(stream[:i]), true))
		// The oracle: the first token boundary of the WHOLE stream at or
		// after the split — framed once, the way the stream really reads.
		boundary := len(stream)
		for _, edge := range tokenEdges(stream) {
			if edge >= i {
				boundary = edge
				break
			}
		}
		got += string(d.Feed([]byte(stream[i:]), false))
		var on sgrDimmer
		want := string(on.Feed([]byte(stream[:boundary]), true)) + "\x1b[22m" + stream[boundary:]
		if got != want {
			t.Fatalf("flip at %d (boundary %d): got %q want %q", i, boundary, got, want)
		}
	}
}

func TestHandleWinchOrdersDimBeforeResizeAndSignalsOnFlip(t *testing.T) {
	var events []string
	focus := false
	p := &proxy{
		stdinFile:    os.Stdin,
		observeFocus: func() (bool, error) { events = append(events, "observe"); return focus, nil },
		getWinsize: func(*os.File) (*pty.Winsize, error) {
			events = append(events, "getsize")
			return &pty.Winsize{Rows: 24, Cols: 80}, nil
		},
		setPTYWinsize: func(*os.File, *pty.Winsize) error { events = append(events, "setsize"); return nil },
		signalChild:   func(sig syscall.Signal) { events = append(events, "signal "+sig.String()) },
	}
	p.handleWinch() // steady off: resize only
	if want := []string{"observe", "getsize", "setsize"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("steady: %v", events)
	}
	events, focus = nil, true
	p.handleWinch() // flip on: dim set before the resize, then an explicit redraw
	if want := []string{"observe", "getsize", "setsize", "signal " + syscall.SIGWINCH.String()}; !reflect.DeepEqual(events, want) || !p.dim.Load() {
		t.Fatalf("flip: %v dim=%v", events, p.dim.Load())
	}
	events = nil
	p.handleWinch() // steady on: no extra redraw
	if len(events) != 3 {
		t.Fatalf("steady on: %v", events)
	}
}

func TestRefreshDimSignalsChildOnlyOnChange(t *testing.T) {
	var observed bool
	var obsErr error
	p := &proxy{observeFocus: func() (bool, error) { return observed, obsErr }}
	if p.refreshDim() {
		t.Fatal("unchanged off reported a change")
	}
	observed = true
	if !p.refreshDim() || !p.dim.Load() {
		t.Fatal("focus on not applied")
	}
	if p.refreshDim() {
		t.Fatal("steady on reported a change")
	}
	obsErr = errors.New("zellij gone")
	observed = false
	if p.refreshDim() || !p.dim.Load() {
		t.Fatal("observation error must keep the last known state")
	}
	obsErr = nil
	if !p.refreshDim() || p.dim.Load() {
		t.Fatal("focus off not applied")
	}
	if (&proxy{}).refreshDim() {
		t.Fatal("no observer outside zellij must never dim")
	}
}

// tokenEdges frames the whole stream and returns every offset that starts a
// token (escape sequence or byte of text), plus the end.
func tokenEdges(stream string) []int {
	var edges []int
	for i := 0; i < len(stream); {
		edges = append(edges, i)
		if n, status := dimFrame([]byte(stream[i:])); status == ansi.Complete {
			i += n
		} else {
			i++
		}
	}
	return append(edges, len(stream))
}
