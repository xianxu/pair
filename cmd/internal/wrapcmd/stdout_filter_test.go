package wrapcmd

import (
	"bytes"
	"reflect"
	"testing"
)

// Negotiation belongs to the receiving Endpoint. No wrapper diagnostic flag
// may remove state transitions, including when a marker spans PTY reads.
func TestOutputPreservesTerminalNegotiationEverySplit(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		for _, marker := range []string{"\x1b[?2026h", "\x1b[?2026l", "\x1b[?1004h", "\x1b[?1004l", "\x1b[>4;0m", "\x1b[>7u", "\x1b[?u"} {
			for split := 0; split <= len(marker); split++ {
				var out bytes.Buffer
				p := &proxy{agentBasename: agent, stdout: &out}
				var rolling []byte
				p.handleChunk([]byte("a"+marker[:split]), &rolling)
				p.handleChunk([]byte(marker[split:]+"b"), &rolling)
				p.flushStdout("test")
				if got, want := out.String(), "a"+marker+"b"; got != want {
					t.Fatalf("%s %q split%d got%q want%q", agent, marker, split, got, want)
				}
			}
		}
	}
}

func TestNormalizedOutputObserverMatchesQueuedVisualStream(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	var out bytes.Buffer
	f.proxy.stdout = &out
	f.proxy.notifyModeActive = "native"
	raw := []byte("before\x1b]9;finished\x07\x1b[?1004h\x1b[?2026hafter\x1b[?2026l")
	for _, b := range raw {
		f.proxy.handleChunk([]byte{b}, &f.rolling)
	}
	f.proxy.flushStdout("test")
	want := newTerminalModelForTest(t, 80, 38)
	if err := want.Feed(out.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.proxy.terminal.Snapshot(), want.Snapshot()) {
		t.Fatal("observer diverged from normalized queued stream")
	}
	if bytes.Contains(out.Bytes(), []byte("\x1b]9;finished")) {
		t.Fatalf("native notification was not normalized: %q", out.Bytes())
	}
	if !bytes.Contains(out.Bytes(), []byte("\x1b[?1004h")) {
		t.Fatal("normalization swallowed unrelated negotiation")
	}
}
