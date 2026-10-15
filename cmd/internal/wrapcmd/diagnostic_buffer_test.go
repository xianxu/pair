package wrapcmd

import (
	"bytes"
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
