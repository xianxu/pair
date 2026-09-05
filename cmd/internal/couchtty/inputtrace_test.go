package couchtty

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// The probe's whole job is to let an operator compare what the terminal sent
// against what the chord table watches for, so it must render a chord the same
// way the table spells it.
func TestRenderInputBytesSpellsChordsLikeTheTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		chord workbenchshortcut.Chord
		want  string
	}{
		{"alt+n", workbenchshortcut.ChordAltN, `\x1b[110;3u`},
		{"ctrl+alt+n", workbenchshortcut.ChordCtrlAltN, `\x1b[110;7u`},
		{"alt+shift+n", workbenchshortcut.ChordAltShiftN, `\x1b[78;4u`},
		{"alt+x kitty", workbenchshortcut.ChordAltX, `\x1b[120;3u`},
		// alt+x also has a LEGACY encoding, and alt+n does not. That asymmetry
		// is the whole reason this probe exists: with the Kitty protocol off,
		// alt+x still reaches couch and alt+n cannot.
		{"alt+x legacy", workbenchshortcut.ChordAltX, `\x1bx`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rendered []string
			for _, encoding := range workbenchshortcut.ChordEncodings(tc.chord) {
				rendered = append(rendered, renderInputBytes(encoding))
			}
			if !slices.Contains(rendered, tc.want) {
				t.Errorf("renderings %q do not include %q", rendered, tc.want)
			}
		})
	}
}

func TestRenderInputBytesKeepsTextReadableAndEscapesTheRest(t *testing.T) {
	if got, want := renderInputBytes([]byte("hi there")), "hi there"; got != want {
		t.Errorf("printable text: got %q, want %q", got, want)
	}
	// ctrl-space's legacy encoding is NUL, the one byte an operator is most
	// likely to be hunting for and the one a naive renderer drops silently.
	if got, want := renderInputBytes([]byte{0x00}), `\x00`; got != want {
		t.Errorf("NUL: got %q, want %q", got, want)
	}
	if got, want := renderInputBytes([]byte{0x08, 0x7f}), `\x08\x7f`; got != want {
		t.Errorf("control bytes: got %q, want %q", got, want)
	}
}

// A nil tracer is the OFF state, and it is the state every operator runs in.
func TestNilInputTracerRecordsWithoutPanicking(t *testing.T) {
	var tracer *inputTracer
	tracer.record([]byte("\x1b[110;3u"))
}

// An instrument that was asked for and could not start must SAY so. Returning a
// nil tracer for both "off" and "could not open" made an unwritable path produce
// an empty trace indistinguishable from "no bytes arrived" -- the exact
// ambiguity the probe exists to remove.
func TestATraceThatCannotStartSaysSoInsteadOfTracingNothing(t *testing.T) {
	unwritable := filepath.Join(t.TempDir(), "no-such-dir", "keys.log")
	tracer, err := newInputTracer(unwritable)
	if err == nil {
		t.Fatal("an unopenable trace path reported no error")
	}
	if tracer != nil {
		t.Fatal("a failed tracer was returned as usable")
	}

	// And the console surfaces it rather than starting silently blind.
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	if err := con.SetInputTrace(unwritable); err == nil {
		t.Fatal("SetInputTrace hid the failure from its caller")
	}
	var bodies []string
	for _, message := range con.feed.Messages() {
		bodies = append(bodies, message.Body)
	}
	if !slices.ContainsFunc(bodies, func(body string) bool {
		return strings.Contains(body, "COUCH_INPUT_TRACE")
	}) {
		t.Fatalf("console kept a dead probe and said nothing: %q", bodies)
	}
}

// A Console must not open a trace file nobody asked it for. couchtty builds many
// per test run, and reading os.Getenv from the constructor meant an exported
// COUCH_INPUT_TRACE leaked one fd per Console plus fixture bytes into a real
// operator file -- the same shape as the PAIR_SESSION_ID leak this repo already
// hit in `make test`.
func TestAConsoleOpensNoTraceUnlessAskedTo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.log")
	t.Setenv("COUCH_INPUT_TRACE", path)

	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	con.mu.Lock()
	tracer := con.trace
	con.mu.Unlock()
	if tracer != nil {
		t.Fatal("the constructor read ambient env and opened a trace file")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("the constructor created the trace file named by ambient env")
	}
}

// Off is still silent: no path, no tracer, no notice.
func TestTracingOffIsSilent(t *testing.T) {
	tracer, err := newInputTracer("")
	if tracer != nil || err != nil {
		t.Fatalf("tracing off produced tracer %v err %v", tracer, err)
	}
}

// A closed tracer must not panic or write; teardown closes it.
func TestAClosedTracerStopsRecording(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.log")
	tracer, err := newInputTracer(path)
	if err != nil {
		t.Fatal(err)
	}
	tracer.record([]byte("a"))
	if err := tracer.Close(); err != nil {
		t.Fatal(err)
	}
	tracer.record([]byte("b"))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "b") {
		t.Fatalf("a closed tracer kept recording: %q", body)
	}
}
