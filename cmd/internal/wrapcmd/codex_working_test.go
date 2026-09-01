package wrapcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestRecognizeCodexWorkingCapturedRenderedFrame(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tty", "codex", "0.152.0", "working.raw"))
	if err != nil {
		t.Fatal(err)
	}
	model := newTerminalModelForTest(t, 120, 38)
	sawWorking := false
	for start := 0; start < len(raw); start += 16 {
		end := start + 16
		if end > len(raw) {
			end = len(raw)
		}
		if err := model.Feed(raw[start:end]); err != nil {
			t.Fatal(err)
		}
		sawWorking = sawWorking || RecognizeCodexWorking(model.Snapshot())
	}
	if !sawWorking {
		t.Fatal("captured Codex Working frames were not recognized")
	}
	if RecognizeCodexWorking(model.Snapshot()) {
		t.Fatal("captured final frame still recognized as Working")
	}
}

func TestRecognizeCodexWorkingRejectsLookalikes(t *testing.T) {
	tests := map[string]string{
		"quoted copy":       "> • Working (1s • esc to interrupt)",
		"prose prefix":      "Codex says • Working (1s • esc to interrupt)",
		"worked completion": "• Worked for 1s",
		"other status":      "• Booting MCP server (1s • esc to interrupt)",
		"missing interrupt": "• Working (1s)",
	}
	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			model := newTerminalModelForTest(t, 120, 38)
			if err := model.Feed([]byte("\x1b[32;1H" + line + "\x1b[35;1H› \x1b[?25h")); err != nil {
				t.Fatal(err)
			}
			if RecognizeCodexWorking(model.Snapshot()) {
				t.Fatalf("lookalike %q was recognized", line)
			}
		})
	}
}

func TestRecognizeCodexWorkingRequiresStatusLocation(t *testing.T) {
	model := newTerminalModelForTest(t, 120, 38)
	if err := model.Feed([]byte("\x1b[10;1H• Working (1s • esc to interrupt)\x1b[35;1H› \x1b[?25h")); err != nil {
		t.Fatal(err)
	}
	if RecognizeCodexWorking(model.Snapshot()) {
		t.Fatal("Working prose outside the live status region was recognized")
	}
}

func TestHandleChunkPublishesCodexWorkingPresenceAndDisappearance(t *testing.T) {
	written := captureCodexVisualNotification(t)
	if want := notifyosc.Encode("agent stopped working"); !bytes.Equal(written, want) {
		t.Fatalf("outer notification = %q, want %q", written, want)
	}
}

func captureCodexVisualNotification(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "tty", "codex", "0.152.0", "working.raw"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	sidecar := filepath.Join(dir, "outer-path")
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := &proxy{
		agentBasename: "codex", notifyModeActive: notifyModeDefault,
		outerTTYFile: sidecar, lastSlug: time.Now(),
	}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatal(err)
	}
	defer p.closeTerminal()
	var rolling []byte
	sawWorking := false
	for start := 0; start < len(raw); start += 16 {
		end := start + 16
		if end > len(raw) {
			end = len(raw)
		}
		p.handleChunk(raw[start:end], &rolling)
		sawWorking = sawWorking || p.codexWorkingRendered
	}
	if !sawWorking {
		t.Fatal("production adapter did not observe captured Working presence")
	}
	if !p.notificationLifecycle.GracePending || p.codexWorkingRendered {
		t.Fatalf("captured final lifecycle = %+v rendered=%t", p.notificationLifecycle, p.codexWorkingRendered)
	}
	p.processLifecycleObservation(TurnObservation{
		Kind: ObservationGraceExpired, Token: p.notificationLifecycle.GraceToken,
	})
	written, err := os.ReadFile(outer)
	if err != nil {
		t.Fatal(err)
	}
	return written
}

func TestCodexWorkingNotificationReachesCouchStatusAndSwitcher(t *testing.T) {
	wrapperBytes := captureCodexVisualNotification(t)
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	con := couchtty.New(host, reader)
	addresses := []couchcore.ThreadAddress{
		{RepoScope: "legacy", Tag: "c1"},
		{RepoScope: "legacy", Tag: "c2"},
	}
	con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: addresses[0], Name: "one", State: couchcore.ThreadLive},
			{Address: addresses[1], Name: "two", State: couchcore.ThreadLive},
		}, nil
	})
	one, two := ptychild.NewFakeChild(nil), ptychild.NewFakeChild(nil)
	one.SetSink(func(batch ptychild.OutputBatch) { con.Deliver("c1", batch) })
	two.SetSink(func(batch ptychild.OutputBatch) { con.Deliver("c2", batch) })
	con.Attach("c1", "one", one)
	con.Attach("c2", "two", two)
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("Couch console did not stop")
		}
	})

	two.Feed(wrapperBytes)
	wantStatus := couchtty.RenderStatusRow(80, couchtty.StatusModel{Actors: []couchtty.StatusActor{
		{Label: "one", Active: true}, {Label: "two", Bell: true},
	}})
	waitForCodexCouch(t, func() bool { return strings.Contains(host.Written(), wantStatus) }, "pending status chip")
	if _, err := writer.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitForCodexCouch(t, func() bool { return strings.Contains(host.Written(), "agent stopped working") }, "switcher message")
}

func waitForCodexCouch(t *testing.T, ready func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for Couch %s", what)
}

func TestHandleChunkDoesNotPublishCodexStopWithoutPriorWorkingFrame(t *testing.T) {
	p := &proxy{agentBasename: "codex", notifyModeActive: notifyModeDefault}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatal(err)
	}
	defer p.closeTerminal()
	var rolling []byte
	p.handleChunk([]byte("\x1b[35;1H› \x1b[?25h"), &rolling)
	if p.notificationLifecycle.Active || p.notificationLifecycle.GracePending {
		t.Fatalf("idle frame changed lifecycle: %+v", p.notificationLifecycle)
	}
}

func FuzzRecognizeCodexWorkingArbitraryRenderedCells(f *testing.F) {
	f.Add([]byte("• Working (1s • esc to interrupt)"), uint8(31))
	f.Add([]byte("> • Working (1s • esc to interrupt)"), uint8(31))
	f.Add([]byte("Worked for 3m 19s"), uint8(4))
	f.Fuzz(func(t *testing.T, raw []byte, row uint8) {
		const width, height = 64, 38
		snapshot := terminalSnapshot{Width: width, Height: height, Cells: make([]uv.Cell, width*height)}
		y := int(row) % height
		for x, r := range []rune(string(raw)) {
			if x >= width {
				break
			}
			snapshot.Cells[y*width+x].Content = string(r)
		}
		_ = RecognizeCodexWorking(snapshot)
	})
}
