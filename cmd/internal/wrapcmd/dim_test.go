package wrapcmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
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
	var d sgrDimmer
	got := feedAll(&d, true, "a\x1b[3")
	// Turning off flushes the held tail raw, then cancels faint before new output.
	got += string(d.Feed([]byte("1mb"), false))
	if want := "\x1b[2ma\x1b[3\x1b[22m1mb"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := string(d.Feed([]byte("c"), false)); got != "c" {
		t.Fatalf("steady off: %q", got)
	}
	// An over-long unterminated CSI is not held forever: it flushes raw.
	d = sgrDimmer{}
	long := "\x1b[" + strings.Repeat("1;", dimPendingMax)
	if got := string(d.Feed([]byte(long), true)); got != "\x1b[2m"+long {
		t.Fatalf("over-long tail held: %d bytes out", len(got))
	}
	if len(d.pending) != 0 {
		t.Fatal("pending retained after flush")
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
