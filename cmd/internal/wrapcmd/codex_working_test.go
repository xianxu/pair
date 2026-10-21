package wrapcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestRecognizeCodexWorkingCapturedRenderedFrame(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "drive", "codex", "0.163.0", "working-prefix.raw"))
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
	if err := model.Feed([]byte(syntheticCodexWork("", false))); err != nil {
		t.Fatal(err)
	}
	if RecognizeCodexWorking(model.Snapshot()) {
		t.Fatal("fresh idle composer still recognized as Working")
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
			if err := model.Feed([]byte(syntheticCodexWork(line, false))); err != nil {
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
	if err := model.Feed([]byte(syntheticCodexWork("", false) + "\x1b[?2026h\x1b[3;1H• Working (1s • esc to interrupt)\x1b[14;3H\x1b[?2026l")); err != nil {
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
	raw, err := os.ReadFile(filepath.Join("testdata", "drive", "codex", "0.163.0", "working-prefix.raw"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	p := &proxy{
		agentBasename: "codex", notifyModeActive: notifyModeDefault,
		stdout: notificationFileWriter(t, outer), lastSlug: time.Now(),
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
	p.handleChunk([]byte(syntheticCodexWork("", false)), &rolling)
	if !p.notificationLifecycle.GracePending || p.codexWorkingRendered {
		t.Fatalf("captured final lifecycle = %+v rendered=%t", p.notificationLifecycle, p.codexWorkingRendered)
	}
	p.processLifecycleObservation(TurnObservation{
		Kind: ObservationGraceExpired, Token: p.notificationLifecycle.GraceToken,
	})
	p.flushStdout("fixture-complete")

	written, err := os.ReadFile(outer)
	if err != nil {
		t.Fatal(err)
	}
	return written
}

func TestCodexWorkingNotificationIsSuppressedByCouch(t *testing.T) {
	wrapperBytes := captureCodexVisualNotification(t)
	if want := notifyosc.Encode("agent stopped working"); !bytes.Equal(wrapperBytes, want) {
		t.Fatalf("wrapper notification = %q, want %q", wrapperBytes, want)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	con := couchtty.New(host, reader)
	tracePath := filepath.Join(t.TempDir(), "couch.trace")
	if err := con.SetEventTrace(tracePath, time.Now()); err != nil {
		t.Fatal(err)
	}
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
	one.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return con.Deliver(ctx, "c1", batch) })
	two.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return con.Deliver(ctx, "c2", batch) })
	con.Attach("c1", "one", one)
	con.Attach("c2", "two", two)
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("Couch console did not stop")
		}
	})

	two.Feed(wrapperBytes)
	// A positive acknowledgement of suppression prevents a no-output check
	// from passing before the asynchronous child publication reaches Couch.
	waitForCodexCouch(t, func() bool {
		trace, err := os.ReadFile(tracePath)
		return err == nil && strings.Contains(string(trace), "\tnotify-suppressed\tlegacy/c2\t")
	}, "notification suppression acknowledgement")
	wantStatus := couchtty.RenderStatusRow(80, couchtty.StatusModel{Actors: []couchtty.StatusActor{
		{Label: "one", Active: true}, {Label: "two"},
	}}).Body
	waitForCodexCouch(t, func() bool { return strings.Contains(codexCouchRendered(t, host.Written()), ansi.Strip(wantStatus)) }, "status row")
	if _, err := writer.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitForCodexCouch(t, func() bool {
		return strings.Contains(codexCouchRendered(t, host.Written()), "threads")
	}, "switcher render")
	// Check the complete host history, including the status frame before the
	// switcher opened. No desktop envelope or notification body may escape.
	written := host.Written()
	if strings.Contains(written, "\x1b]777;notify;") || strings.Contains(written, "agent stopped working") {
		t.Fatalf("Couch leaked the child notification: %q", written)
	}
}

func waitForCodexCouch(t *testing.T, ready func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(min(10*time.Second, testwait.Budget(t)))
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

// Rendering and its teardown share one bound; a stuck reply drain must not
// consume the whole package deadline.
func codexCouchRendered(t *testing.T, raw string) string {
	t.Helper()
	model, err := newTerminalModel(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	text, err := boundedCodexRender(t, raw, model)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func boundedCodexRender(t *testing.T, raw string, model *terminalModel) (string, error) {
	t.Helper()
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := codexCouchRenderedModel(raw, model)
		done <- result{text, err}
	}()
	timer := time.NewTimer(min(10*time.Second, testwait.Budget(t)))
	defer timer.Stop()
	select {
	case got := <-done:
		return got.text, got.err
	case <-timer.C:
		t.Fatal("Codex fixture rendering or reply-drain cleanup exceeded its budget")
		return "", nil
	}
}

func codexCouchRenderedModel(raw string, model *terminalModel) (text string, err error) {
	// terminalModel owns pipe shutdown, the reply-drain join and disposal.
	defer func() { err = errors.Join(err, model.Close()) }()
	if err := model.Feed([]byte(raw)); err != nil {
		return "", err
	}
	return model.emulator.String(), nil
}

func TestCodexCouchRenderOwnsReplyDrain(t *testing.T) {
	shutdownErr := errors.New("reply pipe close failed after shutdown")
	for _, tc := range []struct {
		name string
		err  error
	}{{"success", nil}, {"close error", shutdownErr}} {
		t.Run(tc.name, func(t *testing.T) {
			model, err := newTerminalModel(80, 24)
			if err != nil {
				t.Fatal(err)
			}
			actualCloser := model.replyCloser
			joinedBeforeDisposal := false
			model.replyCloser = closeFunc(func() error {
				err := actualCloser.Close()
				// The real drainer must reach EOF while the emulator is still
				// usable. This orders the assertion without scheduler delays.
				<-model.drainDone
				_, probeErr := model.emulator.Write(nil)
				joinedBeforeDisposal = probeErr == nil
				return errors.Join(err, probeErr, tc.err)
			})
			text, err := boundedCodexRender(t, "\x1b]10;?\x07\x1b]11;?\x07ready", model)
			if want := "ready" + strings.Repeat("\n", 23); text != want {
				t.Errorf("rendered = %q, want %q", text, want)
			}
			if !errors.Is(err, tc.err) {
				t.Errorf("render error = %v, want %v", err, tc.err)
			}
			if !joinedBeforeDisposal {
				t.Error("renderer did not close the reply pipe and join its drainer before disposal")
			}
			select {
			case <-model.drainDone:
			case <-time.After(min(10*time.Second, testwait.Budget(t))):
				t.Fatal("renderer left its reply drainer running")
			}
			if _, err := model.emulator.Write(nil); !errors.Is(err, io.ErrClosedPipe) {
				t.Errorf("emulator remains usable after render cleanup: %v", err)
			}
		})
	}
}

// The old capture was an unframed status-only delta; its text cannot prove
// current work without the agent's composer in the same coherent viewport.
func TestCodexWorkingOldStatusFragmentDoesNotProveWork(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tty", "codex", "0.152.0", "working.raw"))
	if err != nil {
		t.Fatal(err)
	}
	model := newTerminalModelForTest(t, 120, 38)
	for _, b := range raw {
		if err := model.Feed([]byte{b}); err != nil {
			t.Fatal(err)
		}
		if RecognizeCodexWorking(model.Snapshot()) {
			t.Fatal("unqualified status fragment proved work")
		}
	}
}
