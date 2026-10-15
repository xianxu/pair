package wrapcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

type captureCountWriter struct {
	bytes.Buffer
	calls int
	write func([]byte) (int, error)
}

func (w *captureCountWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.write != nil {
		return w.write(p)
	}
	return w.Buffer.Write(p)
}

func TestCaptureBufferFlushAndClosePreserveWitnessedBytes(t *testing.T) {
	for _, closeFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "flush", true: "close"}[closeFirst], func(t *testing.T) {
			dir := t.TempDir()
			raw, err := os.Create(filepath.Join(dir, "raw"))
			if err != nil {
				t.Fatal(err)
			}
			defer raw.Close()
			events, err := os.Create(filepath.Join(dir, "events"))
			if err != nil {
				t.Fatal(err)
			}
			defer events.Close()
			if err := PublishCaptureWitness(raw, events, artifactpath.CaptureWitness{ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, Nonce: "nonce"}); err != nil {
				t.Fatal(err)
			}
			witness, err := os.ReadFile(events.Name())
			if err != nil {
				t.Fatal(err)
			}
			b := newCaptureBuffer(raw, events)
			fragments := [][]byte{[]byte("\x1b[?1049h\x1b[2J"), {0xe4, 0xb8}, {0xad}, []byte("\x1b]0;title\x07"), []byte("\x1b[?1049l")}
			var want []byte
			for _, fragment := range fragments {
				n, err := b.WriteRaw(fragment)
				if err != nil || n != len(fragment) {
					t.Fatalf("write %d %v", n, err)
				}
				want = append(want, fragment...)
				if err := b.WriteEvent("resize", map[string]any{"cols": 80}); err != nil {
					t.Fatal(err)
				}
			}
			if b.Offset() != int64(len(want)) {
				t.Fatal("accepted offset", b.Offset())
			}
			before, err := os.ReadFile(raw.Name())
			if err != nil || len(before) != 0 {
				t.Fatalf("raw published before flush: %q %v", before, err)
			}
			beforeEvents, err := os.ReadFile(events.Name())
			if err != nil || !bytes.Equal(beforeEvents, witness) {
				t.Fatal("events published before flush", err)
			}
			if closeFirst {
				err = b.Close()
			} else {
				err = b.Flush()
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(raw.Name())
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("raw changed %q %v", got, err)
			}
			gotEvents, err := os.ReadFile(events.Name())
			if err != nil || !bytes.HasPrefix(gotEvents, witness) {
				t.Fatal("witness changed", err)
			}
			if _, err := artifactpath.DecodeCaptureWitness(gotEvents); err != nil {
				t.Fatal(err)
			}
			offset := 0
			lines := bytes.Split(bytes.TrimSpace(gotEvents[len(witness):]), []byte("\n"))
			if len(lines) != len(fragments) {
				t.Fatal("missing events", len(lines))
			}
			for i, line := range lines {
				offset += len(fragments[i])
				var event struct {
					Type   string `json:"type"`
					Offset int64  `json:"offset"`
					Cols   int    `json:"cols"`
				}
				if err := json.Unmarshal(line, &event); err != nil {
					t.Fatal(err)
				}
				if event.Type != "resize" || event.Offset != int64(offset) || event.Cols != 80 {
					t.Fatalf("event %+v", event)
				}
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := b.WriteRaw([]byte("late")); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatal("closed raw admitted", err)
			}
			if err := b.WriteEvent("late", nil); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatal("closed event admitted", err)
			}
			// The wrapper, not this buffer, retains ownership of both descriptors.
			if _, err := raw.Write([]byte("caller")); err != nil {
				t.Fatal(err)
			}
			if _, err := events.Write([]byte("caller")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCaptureBufferEventOverflowFlushesRawFirst(t *testing.T) {
	for _, payloadSize := range []int{0, 200} {
		t.Run(map[bool]string{true: "oversized", false: "capacity"}[payloadSize > 0], func(t *testing.T) {
			raw, events := &captureCountWriter{}, &captureCountWriter{}
			events.write = func(p []byte) (int, error) {
				if raw.Len() != 3 {
					t.Fatalf("sidecar outran raw: %q", raw.Bytes())
				}
				return events.Buffer.Write(p)
			}
			b := newCaptureBufferSize(raw, events, 64)
			if _, err := b.WriteRaw([]byte("raw")); err != nil {
				t.Fatal(err)
			}
			for range 4 {
				if err := b.WriteEvent("resize", map[string]any{"padding": strings.Repeat("x", payloadSize)}); err != nil {
					t.Fatal(err)
				}
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
			if events.calls == 0 {
				t.Fatal("events not flushed")
			}
		})
	}
}

func TestCaptureBufferRawErrorStopsEventPublication(t *testing.T) {
	sentinel := errors.New("raw failed")
	for _, phase := range []string{"direct-partial", "flush", "short-write"} {
		t.Run(phase, func(t *testing.T) {
			raw, events := &captureCountWriter{}, &captureCountWriter{}
			raw.write = func(p []byte) (int, error) {
				if phase == "short-write" {
					return 1, nil
				}
				return 1, sentinel
			}
			b := newCaptureBufferSize(raw, events, 16)
			if phase == "direct-partial" {
				n, err := b.WriteRaw(bytes.Repeat([]byte("r"), 32))
				if n != 1 || !errors.Is(err, sentinel) || b.Offset() != 1 {
					t.Fatalf("partial accepted %d offset %d err %v", n, b.Offset(), err)
				}
			} else {
				if _, err := b.WriteRaw([]byte("raw")); err != nil {
					t.Fatal(err)
				}
				if err := b.Flush(); err == nil {
					t.Fatal("raw flush failure lost")
				}
			}
			if err := b.WriteEvent("resize", nil); err == nil {
				t.Fatal("event admitted after raw error")
			}
			if _, err := b.WriteRaw([]byte("later")); err == nil {
				t.Fatal("raw admitted after sticky error")
			}
			if err := b.Close(); err == nil {
				t.Fatal("close lost raw error")
			}
			if events.Len() != 0 {
				t.Fatal("newer offsets published after raw error")
			}
		})
	}
}

func TestCaptureBufferCountsCoalescedWrites(t *testing.T) {
	raw, events := &captureCountWriter{}, &captureCountWriter{}
	b := newCaptureBuffer(raw, events)
	for range 1000 {
		if _, err := b.WriteRaw([]byte("fragment")); err != nil {
			t.Fatal(err)
		}
		if err := b.WriteEvent("time", nil); err != nil {
			t.Fatal(err)
		}
	}
	if raw.calls != 0 || events.calls != 0 {
		t.Fatalf("sub-capacity workload wrote early: raw=%d events=%d", raw.calls, events.calls)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if raw.calls != 1 || events.calls != 1 {
		t.Fatalf("writes not coalesced: raw=%d events=%d", raw.calls, events.calls)
	}
}

func TestCaptureBufferConcurrentAdmissionsAndFlush(t *testing.T) {
	raw, events := &captureCountWriter{}, &captureCountWriter{}
	b := newCaptureBufferSize(raw, events, 64)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(kind int) {
			defer wg.Done()
			for range 100 {
				var err error
				switch kind {
				case 0:
					_, err = b.WriteRaw([]byte("x"))
				case 1:
					err = b.WriteEvent("resize", nil)
				case 2:
					_ = b.Offset()
					err = b.Flush()
				}
				if err != nil {
					t.Error(err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if b.Offset() != 100 || raw.Len() != 100 {
		t.Fatal("lost concurrent raw", b.Offset(), raw.Len())
	}
	last := int64(0)
	for _, line := range bytes.Split(bytes.TrimSpace(events.Bytes()), []byte("\n")) {
		var event struct {
			Offset int64 `json:"offset"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		if event.Offset < last || event.Offset > 100 {
			t.Fatal("invalid concurrent offset", event.Offset)
		}
		last = event.Offset
	}
}

func TestCaptureBufferNilFlushClose(t *testing.T) {
	var b *captureBuffer
	if err := b.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureBufferOverflowRawFailureKeepsSidecarHidden(t *testing.T) {
	sentinel := errors.New("raw overflow flush failed")
	raw, events := &captureCountWriter{}, &captureCountWriter{}
	raw.write = func(p []byte) (int, error) { return 1, sentinel }
	b := newCaptureBufferSize(raw, events, 64)
	if _, err := b.WriteRaw([]byte("raw")); err != nil {
		t.Fatal(err)
	}
	if err := b.WriteEvent("time", nil); err != nil {
		t.Fatal(err)
	}
	if err := b.WriteEvent("resize", map[string]any{"padding": strings.Repeat("x", 100)}); !errors.Is(err, sentinel) {
		t.Fatal("overflow raw error lost", err)
	}
	if err := b.Close(); !errors.Is(err, sentinel) {
		t.Fatal("close raw error lost", err)
	}
	if events.calls != 0 {
		t.Fatal("sidecar flushed despite raw failure", events.calls)
	}
}

func TestCaptureBufferSidecarFailureIsSticky(t *testing.T) {
	sentinel := errors.New("events failed")
	raw, events := &captureCountWriter{}, &captureCountWriter{}
	events.write = func(p []byte) (int, error) { return 1, sentinel }
	b := newCaptureBuffer(raw, events)
	if _, err := b.WriteRaw([]byte("raw")); err != nil {
		t.Fatal(err)
	}
	if err := b.WriteEvent("time", nil); err != nil {
		t.Fatal(err)
	}
	if err := b.Flush(); !errors.Is(err, sentinel) {
		t.Fatal("sidecar flush error lost", err)
	}
	if raw.String() != "raw" {
		t.Fatal("raw was not flushed before sidecar", raw.String())
	}
	if err := b.WriteEvent("resize", nil); !errors.Is(err, sentinel) {
		t.Fatal("sidecar error not sticky", err)
	}
	if err := b.Close(); !errors.Is(err, sentinel) {
		t.Fatal("close error lost", err)
	}
	if events.calls != 1 {
		t.Fatal("failed sidecar was retried", events.calls)
	}
}
