package wrapcmd

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// Sanitized from ops:0's 2026-10-20 11:08 frame: 189x39, cursor (2,35),
// shell footer, then a duplicate resize after the final show-cursor. Only
// transcript body rows are omitted; the admission geometry/chrome is exact.
func recordedIdleShellFrame(shells string) string {
	rule := strings.Repeat("─", 189)
	return fmt.Sprintf("\x1b[?2004h\x1b[?2026h\x1b[?25l\x1b[2J\x1b[33;1H✻ Brewed for 6s · done 11:08 AM · %s still running\x1b[35;1H%s\x1b[36;1H❯\u00a0\x1b[37;1H%s\x1b[38;1H  ⏵⏵ bypass permissions on · %s · ↓ to manage\x1b[36;3H\x1b[?25h\x1b[?2026l", shells, rule, rule, shells)
}

func TestPeerRecordedIdleShellSurvivesDuplicateResize(t *testing.T) {
	for _, shells := range []string{"1 shell", "2 shells"} {
		t.Run(shells, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			f.resize(189, 39)
			f.output(recordedIdleShellFrame(shells))
			before := f.proxy.terminal.Snapshot()
			if peerComposerState("claude", before) != PeerComposerEmpty {
				t.Fatal("recorded frame not ready before resize")
			}
			if err := f.proxy.terminal.Resize(189, 39); err != nil {
				t.Fatal(err)
			}
			after := f.proxy.terminal.Snapshot()
			if !reflect.DeepEqual(before, after) {
				t.Fatal("same-size resize discarded valid recorded composer evidence")
			}
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if !strings.Contains(out.String(), "hello") || d.receipt().Status != couchmessage.Delivering {
				t.Fatalf("quiet shell frame held delivery: %+v", d.receipt())
			}
		})
	}
}

func TestTerminalDuplicateResizePreservesExistingUncertainty(t *testing.T) {
	for _, traffic := range []string{"\x1b[?25l", "\x1b[?2026h"} {
		t.Run(fmt.Sprintf("%q", traffic), func(t *testing.T) {
			m, err := newTerminalModel(189, 39)
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			if err := m.Feed([]byte(recordedIdleShellFrame("1 shell") + traffic)); err != nil {
				t.Fatal(err)
			}
			before := m.Snapshot()
			if err := m.Resize(189, 39); err != nil {
				t.Fatal(err)
			}
			if after := m.Snapshot(); !reflect.DeepEqual(before, after) {
				t.Fatal("no-op resize changed hidden/incomplete evidence")
			}
		})
	}
}

func TestTerminalSameSizeRetryAfterAbortedResizeCanRepaint(t *testing.T) {
	m, err := newTerminalModel(189, 39)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Feed([]byte(recordedIdleShellFrame("1 shell"))); err != nil {
		t.Fatal(err)
	}
	resize, err := m.PrepareResize(190, 39)
	if err != nil {
		t.Fatal(err)
	}
	resize.Abort() // The emulator moved but native PTY resize failed.
	if err := m.Resize(190, 39); err != nil {
		t.Fatal(err)
	}
	if m.Snapshot().CursorVisible {
		t.Fatal("retry granted stale cursor evidence")
	}
	if err := m.Feed([]byte("\x1b[?25h")); err != nil {
		t.Fatal(err)
	}
	if !m.Snapshot().CursorVisible {
		t.Fatal("successful same-size retry cannot observe a fresh native cursor")
	}
}

func TestPeerComposerWaitNamesMissingEvidence(t *testing.T) {
	for _, tc := range []struct{ name, traffic, want string }{
		{"hidden", "\x1b[?25l", "cursor is hidden"},
		{"incomplete", "\x1b[?2026h\x1b[?25l", "terminal frame is incomplete"},
		{"geometry", "\x1b[2J\x1b[1;3H", "composer geometry or input is unrecognized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			f.output(tc.traffic)
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			r := d.receipt()
			if out.Len() != 0 || r.Status.Terminal() || !strings.Contains(r.Detail, tc.want) {
				t.Fatalf("missing wait evidence %q: %+v, wrote %d", tc.want, r, out.Len())
			}
		})
	}
}

func TestPeerBatchPreparationNamesHiddenCursor(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	f.output("\x1b[?25l")
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	r, ok, ready := d.batchStatus(messages[0].ID)
	if !ok || ready || out.Len() != 0 || !strings.Contains(r.Detail, "cursor is hidden") {
		t.Fatalf("batch wait: %+v ready=%v", r, ready)
	}
}
