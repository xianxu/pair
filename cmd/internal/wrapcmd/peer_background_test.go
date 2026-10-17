package wrapcmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestBackgroundWorkLeavesTheFreeComposerDeliverable(t *testing.T) {
	for _, name := range []string{"idle-suggestion", "pr-footer"} {
		t.Run(name, func(t *testing.T) {
			// Paste-mode negotiation belongs to the fake session; the minimal
			// native fixture preserves cells/cursor, not unrelated terminal modes.
			screen := "\x1b[?2004h" + backgroundScreen(t, name)
			f := pickerFake(t)
			now := time.Now()
			d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
			sink, clock := &recordingPeerSink{}, &fakeTimers{}
			d.session, d.afterFunc, d.settleProbe = sink, clock.after, f.proxy.settleState
			f.proxy.peer = d
			peerRender(f, d, screen)
			clock.fireAll()
			d.mu.Lock()
			settled := d.settled
			d.mu.Unlock()
			sink.mu.Lock()
			active := len(sink.backgrounds) > 0 && backgroundName(sink.backgrounds[len(sink.backgrounds)-1]) == "active"
			sink.mu.Unlock()
			if !settled || !active {
				t.Fatalf("native background-only wait: settled=%v background=%v", settled, active)
			}
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if !strings.Contains(out.String(), "hello") {
				t.Fatalf("background work prevented peer paste: %+v", d.receipt())
			}
			// Simulate the native paste repaint without changing the observed footer.
			cursor := f.proxy.terminal.Snapshot().Cursor
			pasted := screen + fmt.Sprintf("\x1b[?2026h\x1b[%d;3H\x1b[0Khello\x1b[?2026l", cursor.Y+1)
			peerRender(f, d, pasted)

			now = now.Add(PeerSubmitDelay)
			f.proxy.dispatchPeer(&out)
			if !bytes.HasSuffix(out.Bytes(), f.proxy.ttyProfile.keymap.altCR) {
				t.Fatalf("background work prevented peer submit: %+v", d.receipt())
			}
			now = now.Add(time.Second)
			peerRender(f, d, screen)
			f.proxy.dispatchPeer(&out)
			if r := d.receipt(); r.Status != couchmessage.Submitted {
				t.Fatalf("native background wait prevented confirmed receipt: %+v", r)
			}
			clock.fireAll()
			if got := backgroundName(f.proxy.settleState().BackgroundWork); got != "active" {
				t.Fatalf("confirmation lost running shell evidence: %s", got)
			}
		})
	}
}

func TestBackgroundWorkPublishesWhileSettledUnchanged(t *testing.T) {
	clock := &fakeTimers{}
	sink := &recordingPeerSink{}
	active, none := true, false
	state := &active
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.settled, d.session, d.afterFunc = true, sink, clock.after
	d.settleProbe = func() wrapperSettleState { return wrapperSettleState{Settled: true, BackgroundWork: state} }
	for _, next := range []*bool{&active, &none, nil} {
		state = next
		d.mu.Lock()
		d.armSettleLocked()
		d.mu.Unlock()
		clock.fireAll()
		sink.mu.Lock()
		if len(sink.backgrounds) == 0 || backgroundName(sink.backgrounds[len(sink.backgrounds)-1]) != backgroundName(next) {
			sink.mu.Unlock()
			t.Fatalf("unchanged Settled suppressed background=%s", backgroundName(next))
		}
		if len(sink.settles) != 0 {
			sink.mu.Unlock()
			t.Fatal("background evidence changed settled admission")
		}
		sink.mu.Unlock()
	}
}

func TestBackgroundWorkStaleSettleProbeCannotPublish(t *testing.T) {
	clock := &fakeTimers{}
	sink := &recordingPeerSink{}
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.settled, d.session, d.afterFunc = true, sink, clock.after
	d.settleProbe = func() wrapperSettleState {
		d.observeOutput([]byte("new paint"))
		d.outputForwarded()
		active := true
		return wrapperSettleState{Settled: true, BackgroundWork: &active}
	}
	d.mu.Lock()
	d.armSettleLocked()
	d.mu.Unlock()
	clock.fireAll()
	sink.mu.Lock()
	defer sink.mu.Unlock()
	for _, v := range sink.backgrounds {
		if v != nil && *v {
			t.Fatal("stale probe published positive work after source changed")
		}
	}
}

func TestBackgroundWorkObservesTerminalPaintAndLifetime(t *testing.T) {
	f := pickerFake(t)
	f.output(peerBusyScreen(t, "idle-suggestion"))
	if got := backgroundName(f.proxy.settleState().BackgroundWork); got != "active" {
		t.Fatalf("complete native footer = %s", got)
	}
	f.output("\x1b[?2026h")
	if got := f.proxy.settleState().BackgroundWork; got != nil {
		t.Fatal("open synchronized frame retained known background work")
	}
	f.output("\x1b[?2026l" + peerBusyScreen(t, "permission-autodenied"))
	if got := backgroundName(f.proxy.settleState().BackgroundWork); got != "none" {
		t.Fatalf("completed native footer = %s", got)
	}
	resize, err := f.proxy.terminal.PrepareResize(95, 39)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.proxy.settleState().BackgroundWork; got != nil {
		t.Fatal("resize retained known background work")
	}
	_ = resize.Commit()
	_ = f.proxy.terminal.Close()
	if got := f.proxy.settleState().BackgroundWork; got != nil {
		t.Fatal("closed terminal retained known background work")
	}
}

func backgroundScreen(t *testing.T, name string) string {
	t.Helper()
	if name != "pr-footer" {
		return peerBusyScreen(t, name)
	}
	raw, err := os.ReadFile("testdata/peer/claude/2.1.301/pr-footer.raw")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestBackgroundWorkRecordedPRFooter(t *testing.T) {
	s := backgroundSnapshot(t, "pr-footer")
	if s.Width != 94 || s.Height != 39 || s.Cursor.X != 2 || s.Cursor.Y != 34 || !s.CursorVisible || !s.Complete {
		t.Fatalf("native geometry/cursor: %+v", s.Cursor)
	}
	top, bottom, ok := ruledBoxComposerBounds(s, claudeComposerSpec())
	if !ok || top != 34 || bottom != 35 {
		t.Fatalf("native bounds = %d,%d,%v", top, bottom, ok)
	}
	if peerComposerState("claude", s) != PeerComposerEmpty {
		t.Fatal("native PR composer not empty")
	}
	for _, y := range []int{33, 35} {
		if got := terminalRowText(s, y); got != strings.Repeat("─", 94) {
			t.Fatalf("native border row %d = %q", y, got)
		}
	}
	const footer = "⏵⏵ bypass permissions on · PR #387 · 2 shells · ← for agents · ↓ to manage"
	if got := strings.TrimSpace(terminalRowText(s, 36)); got != footer {
		t.Fatalf("native footer = %q", got)
	}

	if got := backgroundName(backgroundWorkState("claude", s)); got != "active" {
		t.Fatalf("native PR footer = %s, want active; row=%q", got, terminalRowText(s, 36))
	}
}

func backgroundSnapshot(t *testing.T, name string) terminalSnapshot {
	t.Helper()
	f := pickerFake(t)
	f.output(backgroundScreen(t, name))
	return f.proxy.terminal.Snapshot()
}

func backgroundName(v *bool) string {
	if v == nil {
		return "unknown"
	}
	if *v {
		return "active"
	}
	return "none"
}

func backgroundRow(s *terminalSnapshot, y int, text string) {
	for x := 0; x < s.Width; x++ {
		s.Cells[y*s.Width+x] = uv.Cell{Content: " ", Width: 1}
	}
	for x, r := range []rune(text) {
		if x >= s.Width {
			break
		}
		s.Cells[y*s.Width+x] = uv.Cell{Content: string(r), Width: 1}
	}
}

func TestBackgroundWorkRecordedFooter(t *testing.T) {
	for _, name := range []string{"shell-footer", "idle-suggestion"} {
		t.Run(name, func(t *testing.T) {
			s := backgroundSnapshot(t, name)
			if peerComposerState("claude", s) != PeerComposerEmpty {
				t.Fatal("background work made the recorded composer unavailable")
			}
			if got := backgroundName(backgroundWorkState("claude", s)); got != "active" {
				t.Fatalf("recorded native shell footer = %s, want active", got)
			}
		})
	}
}

func TestBackgroundWorkQualifiedFooterGrammar(t *testing.T) {
	base := backgroundSnapshot(t, "permission-autodenied")
	_, bottom, ok := ruledBoxComposerBounds(base, claudeComposerSpec())
	if !ok {
		t.Fatal("recorded composer missing")
	}
	for _, tc := range []struct{ field, want string }{
		{"1 shell · ↓ to manage", "active"},
		{"2 shells · ↓ to manage", "active"},
		{"0 shells · ↓ to manage", "none"},
		{"PR #387 · 2 shells · ← for agents · ↓ to manage", "active"},
		{"PR #1 · 1 shell · ↓ to manage", "active"},
		{"PR #387 · 0 shells · ↓ to manage", "none"},
		{"PR #387", "unknown"},
		{"PR #", "unknown"},
		{"PR #0 · 2 shells · ↓ to manage", "unknown"},
		{"PR #01 · 2 shells · ↓ to manage", "unknown"},
		{"PR #+1 · 2 shells · ↓ to manage", "unknown"},
		{"PR #-1 · 2 shells · ↓ to manage", "unknown"},
		{"PR #x · 2 shells · ↓ to manage", "unknown"},
		{"PR #99999999999999999999 · 2 shells · ↓ to manage", "unknown"},
		{"PR #387 · 2 she", "unknown"},
		{"PR #387 · PR #388 · 2 shells · ↓ to manage", "unknown"},
		{"2 shells · PR #387 · ↓ to manage", "unknown"},
		{"← for agents · PR #387 · 2 shells · ↓ to manage", "unknown"},
		{"garbage · PR #387 · 2 shells · ↓ to manage", "unknown"},
		{"PR #387 · 2 shells · ↓ to manage · garbage", "unknown"},
		{"1 shells · ↓ to manage", "unknown"},
		{"2 shell · ↓ to manage", "unknown"},
		{"01 shell · ↓ to manage", "unknown"},
		{"-1 shell · ↓ to manage", "unknown"},
		{"999999999999999999999 shells · ↓ to manage", "unknown"},
		{"x shell · ↓ to manage", "unknown"},
		{"1 shell", "unknown"},
		{"1 shell · 2 shells · ↓ to manage", "unknown"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			s := cloneTerminalSnapshot(base)
			backgroundRow(&s, bottom+1, "  ⏵⏵ bypass permissions on · "+tc.field)
			if got := backgroundName(backgroundWorkState("claude", s)); got != tc.want {
				t.Fatalf("footer %q = %s, want %s", tc.field, got, tc.want)
			}
		})
	}
}

func TestBackgroundWorkRecordedNoShellFooter(t *testing.T) {
	for _, name := range []string{"permission-autodenied", "turn-empty", "turn-paste", "turn-queued"} {
		if got := backgroundName(backgroundWorkState("claude", backgroundSnapshot(t, name))); got != "none" {
			t.Fatalf("recorded %s footer = %s, want none", name, got)
		}
	}
}

func TestBackgroundWorkUnqualifiedFooterStaysUnknown(t *testing.T) {
	base := backgroundSnapshot(t, "permission-autodenied")
	_, bottom, _ := ruledBoxComposerBounds(base, claudeComposerSpec())
	for _, footer := range []string{
		"⏵⏵ bypass permissions on (shift+tab to cycle) · PR #387",
		"⏵⏵ bypass permissions on · PR#387 · 2 shells · ↓ to manage",
		"⏵⏵ bypass permissions on · PR #387… · 2 shells · ↓ to manage",
		"⏵⏵ garbage",
		"⏵⏵ garbage · 1 shell · ↓ to manage",
		"⏵⏵ bypass permissions on",
		"⏵⏵ bypass permissions on · 1 she",
		"⏵⏵ bypass permissions on (shift+tab to cycle) · 1 she",
		"⏵⏵ bypass permissions on (shift+tab to cycle) · 1",
		"⏵⏵ bypass permissions on (shift+tab to cycle) · garbage",
		"⏵⏵ bypass permissions on (shift+tab to cycle) · ↓ to manage",
		"⏵⏵ bypass permissions on · 1 shell · ↓ to manage · garbage",
		"⏵⏵ bypass permissions on · 1 shell · ↓ to manage · ↓ to manage",
	} {
		t.Run(footer, func(t *testing.T) {
			s := cloneTerminalSnapshot(base)
			backgroundRow(&s, bottom+1, footer)
			if got := backgroundWorkState("claude", s); got != nil {
				t.Fatalf("unqualified footer %q = %s, want unknown", footer, backgroundName(got))
			}
		})
	}
}

func TestBackgroundWorkNeverUsesBodyDraftOrIncompletePaint(t *testing.T) {
	base := backgroundSnapshot(t, "permission-autodenied")
	top, bottom, _ := ruledBoxComposerBounds(base, claudeComposerSpec())
	for _, y := range []int{0, top - 2, top} {
		s := cloneTerminalSnapshot(base)
		text := "⏵⏵ bypass permissions on · 1 shell · ↓ to manage"
		if y == top {
			text = "❯ " + text
		}
		backgroundRow(&s, y, text)
		if got := backgroundWorkState("claude", s); got != nil && *got {
			t.Fatalf("body/draft row %d counted as work", y)
		}
	}
	s := backgroundSnapshot(t, "idle-suggestion")
	s.Complete = false
	if got := backgroundWorkState("claude", s); got != nil {
		t.Fatalf("unfinished frame = %s, want unknown", backgroundName(got))
	}
	for _, agent := range []string{"codex", "qoder", "unknown"} {
		if got := backgroundWorkState(agent, base); got != nil {
			t.Fatalf("unsupported %s = %s", agent, backgroundName(got))
		}
	}
	s = cloneTerminalSnapshot(base)
	for y := bottom + 1; y < s.Height; y++ {
		backgroundRow(&s, y, "")
	}
	if got := backgroundWorkState("claude", s); got != nil {
		t.Fatalf("missing native footer = %s", backgroundName(got))
	}
	if got := backgroundWorkState("claude", terminalSnapshot{}); got != nil {
		t.Fatal("empty snapshot counted as known")
	}
}

func FuzzBackgroundWorkState(f *testing.F) {
	f.Add("1 shell · ↓ to manage", uint8(37), true)
	f.Add("PR #387 · 2 shells · ← for agents · ↓ to manage", uint8(37), true)
	f.Add("999999999999999 shells · ↓ to manage", uint8(0), true)
	f.Fuzz(func(t *testing.T, field string, row uint8, complete bool) {
		if len(field) > 512 {
			field = field[:512]
		}
		s := backgroundSnapshot(t, "permission-autodenied")
		_, bottom, _ := ruledBoxComposerBounds(s, claudeComposerSpec())
		y := int(row) % s.Height
		backgroundRow(&s, y, "⏵⏵ bypass permissions on · "+strings.ReplaceAll(field, "\n", " "))
		s.Complete = complete
		got := backgroundWorkState("claude", s)
		if (!complete || y <= bottom) && got != nil && *got {
			t.Fatalf("counter outside complete qualified footer counted: row=%d bottom=%d complete=%v", y, bottom, complete)
		}
	})
}
