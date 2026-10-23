package wrapcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchtty"
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

func nativeTabWrapFixture(t *testing.T) ([]byte, []struct{ Offset, Cols, Rows int }) {
	t.Helper()
	raw, err := os.ReadFile("testdata/peer/claude/2.1.304/tab-wrap.raw")
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile("testdata/peer/claude/2.1.304/tab-wrap.events.json")
	if err != nil {
		t.Fatal(err)
	}
	var es []struct{ Offset, Cols, Rows int }
	if err := json.Unmarshal(events, &es); err != nil {
		t.Fatal(err)
	}
	if len(es) != 3 {
		t.Fatalf("native resize events=%d", len(es))
	}
	return raw, es
}

func TestPeerVisibleTabWrapNativeResize(t *testing.T) {
	raw, es := nativeTabWrapFixture(t)
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint("batch=", batch), func(t *testing.T) {
			var f *harnessSessionFake
			var d *peerDelivery
			if batch {
				f, d, _, _ = peerBatchFixture(t)
			} else {
				f, d = peerIntegrationFixture(t)
				peerIntegrationEnqueue(t, d)
			}
			// The host stays 94x40; showing a second group wraps the chrome.
			one := couchtty.StatusModel{Actors: []couchtty.StatusActor{{Label: strings.Repeat("a", 50), GroupKey: "a", Active: true}}}
			two := one
			two.Actors = append(append([]couchtty.StatusActor(nil), one.Actors...), couchtty.StatusActor{Label: strings.Repeat("b", 50), GroupKey: "b"})
			var input bytes.Buffer
			for i, model := range []couchtty.StatusModel{one, two, one} {
				bar := couchtty.RenderStatusBar(94, 2, model)
				wantRows := 1
				if i == 1 {
					wantRows = 2
				}
				if bar.Height() != wantRows {
					t.Fatalf("chrome rows=%d want%d", bar.Height(), wantRows)
				}
				if es[i].Cols != 94 || es[i].Rows != 40-bar.Height() {
					t.Fatal("fixture not the visible body's chrome-derived geometry")
				}
				f.resize(es[i].Cols, es[i].Rows)
				if i > 0 {
					f.proxy.dispatchPeer(&input)
					if input.Len() != 0 || !strings.Contains(d.receipt().Detail, "cursor is hidden") || batch && d.batchReady() {
						t.Fatalf("unrepainted resize admitted: %+v", d.receipt())
					}
				}
				end := len(raw)
				if i+1 < len(es) {
					end = es[i+1].Offset
				}
				f.output(string(raw[es[i].Offset:end]))
				snapshot := f.proxy.terminal.Snapshot()
				if peerComposerState("claude", snapshot) != PeerComposerEmpty || !snapshot.Complete || !snapshot.CursorVisible {
					t.Fatalf("native repaint did not restore empty composer: step%d", i)
				}
				// Duplicate SIGWINCH without output must preserve the repainted evidence.
				f.resize(es[i].Cols, es[i].Rows)
				if peerComposerState("claude", f.proxy.terminal.Snapshot()) != PeerComposerEmpty {
					t.Fatal("duplicate resize lost repaint")
				}
			}
			f.proxy.dispatchPeer(&input)
			if batch {
				if !d.batchReady() || input.Len() != 0 {
					t.Fatal("batch not ready after native repaint")
				}
			} else if !strings.Contains(input.String(), "hello") {
				t.Fatalf("delivery did not recover: %+v", d.receipt())
			}
		})
	}
}

func TestPeerResizeThenSilencePreservesRefusal(t *testing.T) {
	raw, es := nativeTabWrapFixture(t)
	f, d := peerIntegrationFixture(t)
	f.resize(es[0].Cols, es[0].Rows)
	f.output(string(raw[:es[1].Offset]))
	if peerComposerState("claude", f.proxy.terminal.Snapshot()) != PeerComposerEmpty {
		t.Fatal("not idle before resize")
	}
	now := time.Now()
	d.now = func() time.Time { return now }
	peerIntegrationEnqueue(t, d)
	f.resize(es[1].Cols, es[1].Rows)
	var input bytes.Buffer
	var sink peerCaptureWriter
	f.proxy.wrapEventsFD = &sink
	started := now
	for _, seconds := range []int{0, 15, 30, 300} {
		now = started.Add(time.Duration(seconds) * time.Second)
		f.proxy.dispatchPeer(&input)
		if input.Len() != 0 || !strings.Contains(d.receipt().Detail, "cursor is hidden") {
			t.Fatalf("silence after %ds changed safety: %+v", seconds, d.receipt())
		}
	}
	var record peerWaitRecord
	if err := json.Unmarshal(bytes.TrimSpace(sink.Bytes()), &record); err != nil {
		t.Fatal(err)
	}
	if record.Evidence.LastResize == nil || record.Evidence.LastResize.Rows != 38 || record.CursorVisible {
		t.Fatal("silent resize evidence absent")
	}
	if bytes.Count(sink.Bytes(), []byte(`"label":"PEER-wait"`)) != 1 {
		t.Fatal("silence repeated capture")
	}
	f.output(string(raw[es[1].Offset:es[2].Offset]))
	f.proxy.dispatchPeer(&input)
	if !strings.Contains(input.String(), "hello") {
		t.Fatal("native repaint did not restore safe admission")
	}
}
