package wrapcmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func fmtGeometry(size pty.Winsize) string { return fmt.Sprintf("%dx%d", size.Cols, size.Rows) }

const peerTallReturnSentinel = "RETURN_NEWLINE_PROBE"

type peerTallReturnPhase uint8

const (
	peerTallAwaitPaste peerTallReturnPhase = iota
	peerTallReturnReady
	peerTallAwaitNewline
	peerTallConfirmed
)

// This probe never advances the automatic delivery after its initial paste.
// It owns one native Return, then requires a rendered newline before finishing.
type peerTallReturnProbe struct {
	envelope              string
	phase                 peerTallReturnPhase
	beforeRows, afterRows int
	returnInput           []byte
}

func (r *peerTallReturnProbe) observe(p *proxy) (bool, error) {
	s := p.terminal.Snapshot()
	text, known := peerComposerText("codex", s)
	if !known || !s.Complete {
		return false, nil
	}
	switch r.phase {
	case peerTallAwaitPaste:
		if text != r.envelope {
			return false, nil
		}
		top, footer, ok := codexComposerBounds(s)
		if !ok || footer < 0 || s.Cursor.Y-top < 25 {
			return false, fmt.Errorf("native input did not exercise 26 visible qualified composer rows")
		}
		got := p.emitPlainCR(nil)
		if !bytes.Equal(got, []byte{'\n'}) {
			return false, fmt.Errorf("native tall Return=%q, want LF", got)
		}
		r.returnInput = append(got, []byte(peerTallReturnSentinel)...)
		r.beforeRows = s.Cursor.Y - top + 1
		r.phase = peerTallReturnReady
	case peerTallAwaitNewline:
		if text == r.envelope+"\n"+peerTallReturnSentinel {
			top, footer, ok := codexComposerBounds(s)
			if !ok || footer < 0 || s.Cursor.Y-top+1 != r.beforeRows+1 {
				return false, fmt.Errorf("native Return did not add exactly one composer row")
			}
			r.afterRows = s.Cursor.Y - top + 1
			r.phase = peerTallConfirmed
			return true, nil
		}
	case peerTallConfirmed:
		return true, nil
	}
	return false, nil
}

func (r *peerTallReturnProbe) input() []byte {
	if r.phase != peerTallReturnReady {
		return nil
	}
	r.phase = peerTallAwaitNewline
	input := r.returnInput
	r.returnInput = nil
	return input
}

func TestPeerLiveTallReturnProbeRequiresRenderedNewline(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.resize(100, 60)
	body := codexTallBatchText()
	probe := peerTallReturnProbe{envelope: body}
	f.output(codexTallPaint(body, codexTallTestFooter))
	if done, err := probe.observe(f.proxy); err != nil || done {
		t.Fatalf("paste observation done=%v err=%v", done, err)
	}
	if got := probe.input(); string(got) != "\n"+peerTallReturnSentinel {
		t.Fatalf("native Return probe=%q, want LF plus sentinel", got)
	}
	if got := probe.input(); got != nil {
		t.Fatalf("Return probe replayed input %q", got)
	}
	if done, err := probe.observe(f.proxy); err != nil || done {
		t.Fatalf("unchanged screen proved a newline: done=%v err=%v", done, err)
	}
	f.output(codexTallPaint(body+"\n"+peerTallReturnSentinel, codexTallTestFooter))
	if done, err := probe.observe(f.proxy); err != nil || !done {
		t.Fatalf("native-shaped newline not proved: done=%v err=%v", done, err)
	}
}

func TestPeerLiveTallReturnProbeDeclinesCollapsedOrUnknown(t *testing.T) {
	for _, tc := range []struct{ name, text, footer string }{
		{"collapsed", "[Pasted text: 1500 chars]", codexTallTestFooter},
		{"changed footer", codexTallBatchText(), "New Codex footer format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHarnessSessionFake(t, "codex", true)
			t.Cleanup(f.close)
			f.resize(100, 60)
			probe := peerTallReturnProbe{envelope: codexTallBatchText()}
			f.output(codexTallPaint(tc.text, tc.footer))
			if done, _ := probe.observe(f.proxy); done || len(probe.input()) != 0 {
				t.Fatal("unqualified native tall paste authorized Return")
			}
		})
	}
}

func TestHarnessTTYCaptureCustomGeometry(t *testing.T) {
	for _, size := range []pty.Winsize{{Rows: 38, Cols: 120}, {Rows: 60, Cols: 100}} {
		out, err := captureHarnessTTY(harnessTTYCaptureRequest{
			Executable: os.Args[0], Args: []string{"-test.run=^TestHarnessTTYControlledChild$", "--", "geometry"},
			Env: append(os.Environ(), "PAIR_HARNESS_TTY_CHILD=1"), Size: &size,
			Startup: func(out []byte) bool { return bytes.Contains(out, []byte("READY")) },
		})
		if err != nil {
			t.Fatalf("geometry capture: %v", err)
		}
		want := fmtGeometry(size)
		if !strings.Contains(string(out), want) {
			t.Fatalf("owned PTY geometry=%q, want %q", out, want)
		}
		c := newHarnessTTYLiveClassifierSize(t, "codex", int(size.Cols), int(size.Rows))
		defer c.Close()
		if s := c.proxy.terminal.Snapshot(); s.Width != int(size.Cols) || s.Height != int(size.Rows) {
			t.Fatalf("classifier geometry=%dx%d, PTY=%s", s.Width, s.Height, want)
		}
	}
}
