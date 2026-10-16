package wrapcmd

import (
	"bytes"
	"errors"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProxyDebugBatchesUntilClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	p := &proxy{debugLogPath: path, diagnosticOptions: func() diagnosticlog.Options { return diagnosticlog.Options{FlushTicks: make(chan time.Time)} }}
	p.debug("probe", "buffered")
	if p.debugWriter == nil {
		t.Fatal("debug writer did not open")
	}
	defer p.debugWriter.Close()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 0 {
		t.Fatalf("debug producer still writes every event immediately: %q", b)
	}
	if err = p.debugWriter.Close(); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "probe") {
		t.Fatalf("close lost accepted diagnostic: %q %v", b, err)
	}
}

func TestProxyDebugCannotLateOpenAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late.log")
	p := &proxy{debugLogPath: path}
	if err := p.closeDebug(); err != nil {
		t.Fatal(err)
	}
	p.debug("late", "must not create another worker")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("closed owner reopened diagnostics: %v", err)
	}
}

func TestProxyTraceUsesBufferedOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.log")
	t.Setenv("PAIR_WRAP_EVENTS", "1")
	p := &proxy{wrapEventsPath: path, agentBasename: "codex", diagnosticOptions: func() diagnosticlog.Options { return diagnosticlog.Options{FlushTicks: make(chan time.Time)} }}
	p.openWrapEvents([]string{"codex"})
	if p.wrapEventsFD == nil {
		t.Fatal("trace owner not opened")
	}
	defer p.closeWrapEvents()
	p.traceWrap("probe", nil)
	b, err := os.ReadFile(path)
	if err != nil || len(b) != 0 {
		t.Fatalf("trace bypassed buffer: %q %v", b, err)
	}
	if err := p.closeWrapEvents(); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), `"label":"start"`) || !strings.Contains(string(b), `"label":"probe"`) {
		t.Fatalf("trace records lost: %q %v", b, err)
	}
}

func TestHandleChunkUsesOrderedCaptureBuffer(t *testing.T) {
	dir := t.TempDir()
	raw, events, err := openWitnessedCapture(filepath.Join(dir, "raw"), filepath.Join(dir, "events"), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	p := &proxy{scrollbackFD: raw, eventsFD: events, capture: newCaptureBuffer(raw, events), now: func() time.Time { return time.Unix(1, 0) }}
	defer p.closeCapture()
	p.stdoutPump = newStdoutPump(io.Discard)
	var rolling []byte
	want := []byte("\x1b[?1049h\x1b[31m界\x1b[?1049l\r\n")
	for _, fragment := range [][]byte{want[:11], want[11:16], want[16:]} {
		p.handleChunk(fragment, &rolling)
	}
	before, err := os.ReadFile(raw.Name())
	if err != nil || len(before) != 0 {
		t.Fatalf("handleChunk bypassed raw buffer: %q %v", before, err)
	}
	before, err = os.ReadFile(events.Name())
	if err != nil || len(before) != 0 {
		t.Fatalf("sidecar escaped before raw: %q %v", before, err)
	}
	if err := p.capture.Flush(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(raw.Name())
	if err != nil || !bytes.Equal(after, want) {
		t.Fatalf("captured terminal stream changed: %q %v", after, err)
	}
	if p.scrollbackOffset() != int64(len(want)) {
		t.Fatalf("logical offset=%d", p.scrollbackOffset())
	}
	after, err = os.ReadFile(events.Name())
	if err != nil || !bytes.Contains(after, []byte(`"offset":11`)) {
		t.Fatalf("sidecar offset mismatch: %q %v", after, err)
	}
}

func TestMasterPumpFlushesCaptureOnInjectedTick(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time)
	outputSeen := make(chan struct{}, 1)
	rawFlushed := make(chan struct{}, 1)
	raw := newDrainBuffer()
	events := newDrainBuffer()
	capture := newCaptureBuffer(writerFunc(func(data []byte) (int, error) {
		n, err := raw.Write(data)
		rawFlushed <- struct{}{}
		return n, err
	}), events)
	p := &proxy{ptmx: reader, scrollbackFD: reader, capture: capture, captureFlushTicks: ticks,
		stdoutPump:    newStdoutPump(writerFunc(func(data []byte) (int, error) { outputSeen <- struct{}{}; return len(data), nil })),
		agentBasename: "claude", now: time.Now, notifyModeActive: notifyModeDefault, captureWindow: defaultCaptureWindow}
	done := make(chan struct{})
	prepareMasterPumpTransport(t, p, done)
	go func() { p.masterPump(); close(done) }()
	defer func() { _ = writer.Close(); <-done; _ = capture.Close() }()
	if _, err := writer.Write([]byte("tick-capture")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-outputSeen:
	case <-t.Context().Done():
		t.Fatal("pump did not process output")
	}
	if got := raw.Bytes(); len(got) != 0 {
		t.Fatalf("capture flushed before tick: %q", got)
	}
	select {
	case ticks <- time.Unix(1, 0):
	case <-t.Context().Done():
		t.Fatal("pump did not accept capture tick")
	}
	select {
	case <-rawFlushed:
	case <-t.Context().Done():
		t.Fatal("capture tick did not flush")
	}
	if got := string(raw.Bytes()); got != "tick-capture" {
		t.Fatalf("capture=%q", got)
	}
}

func TestMasterPumpDrainsCaptureBeforeReturningOnEOF(t *testing.T) {
	for _, failFlush := range []bool{false, true} {
		t.Run(map[bool]string{false: "drain", true: "report-error"}[failFlush], func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			var raw bytes.Buffer
			writeFailure := errors.New("capture sink failed")
			sink := writerFunc(func(data []byte) (int, error) {
				if failFlush {
					return 0, writeFailure
				}
				return raw.Write(data)
			})
			capture := newCaptureBuffer(sink, io.Discard)
			debugPath := filepath.Join(t.TempDir(), "debug")
			p := &proxy{ptmx: reader, scrollbackFD: reader, capture: capture, captureFlushTicks: make(chan time.Time),
				stdoutPump: newStdoutPump(io.Discard), agentBasename: "claude", now: time.Now, notifyModeActive: notifyModeDefault,
				captureWindow: defaultCaptureWindow, debugLogPath: debugPath,
				diagnosticOptions: func() diagnosticlog.Options { return diagnosticlog.Options{FlushTicks: make(chan time.Time)} }}
			defer p.closeDebug()
			done := make(chan struct{})
			prepareMasterPumpTransport(t, p, done)
			go func() { p.masterPump(); close(done) }()
			defer func() { _ = writer.Close(); <-done; _ = capture.Close() }()
			if _, err := writer.Write([]byte("accepted-eof")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-t.Context().Done():
				t.Fatal("pump did not return after EOF")
			}
			if err := capture.WriteEvent("resize", map[string]any{"cols": 80, "rows": 24}); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("stopped capture owner accepted late resize: %v", err)
			}
			if capture.Offset() != int64(len("accepted-eof")) {
				t.Fatalf("capture offset=%d", capture.Offset())
			}
			if !failFlush {
				if raw.String() != "accepted-eof" {
					t.Fatalf("pump stopped its flush owner with accepted bytes pending: %q", raw.String())
				}
				return
			}
			if err := p.closeDebug(); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(debugPath)
			if err != nil || !bytes.Contains(b, []byte("SCROLLBACK-flush-fail")) || !bytes.Contains(b, []byte(writeFailure.Error())) {
				t.Fatalf("pump exit hid capture flush failure: %s %v", b, err)
			}
		})
	}
}

func TestMasterPumpRejectsInFlightResizeAfterOwnerEnds(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	resizeEntered, releaseResize, resizeDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var events bytes.Buffer
	capture := newCaptureBuffer(io.Discard, &events)
	p := &proxy{ptmx: reader, stdinFile: reader, scrollbackFD: reader, eventsFD: reader, capture: capture,
		captureFlushTicks: make(chan time.Time), stdoutPump: newStdoutPump(io.Discard),
		agentBasename: "claude", now: time.Now, notifyModeActive: notifyModeDefault, captureWindow: defaultCaptureWindow,
		getWinsize:    func(*os.File) (*pty.Winsize, error) { return &pty.Winsize{Cols: 80, Rows: 24}, nil },
		setPTYWinsize: func(*os.File, *pty.Winsize) error { close(resizeEntered); <-releaseResize; return nil }}
	go func() { p.setWinsize(); close(resizeDone) }()
	<-resizeEntered // Successful ioctl is in flight before PTY EOF; event follows later.
	pumpDone := make(chan struct{})
	prepareMasterPumpTransport(t, p, pumpDone)
	go func() { p.masterPump(); close(pumpDone) }()
	defer func() {
		_ = writer.Close()
		select {
		case <-releaseResize:
		default:
			close(releaseResize)
		}
		<-resizeDone
		<-pumpDone
		_ = capture.Close()
	}()
	if _, err := writer.Write([]byte("last-output")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	<-pumpDone
	close(releaseResize)
	<-resizeDone
	if n := capture.events.Buffered(); n != 0 {
		t.Fatalf("resize admitted %d sidecar bytes after its flush owner ended", n)
	}
	if bytes.Contains(events.Bytes(), []byte(`"type":"resize"`)) {
		t.Fatalf("late resize published after capture closed: %s", events.Bytes())
	}
}
