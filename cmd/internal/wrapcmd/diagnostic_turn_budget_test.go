package wrapcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

type diagnosticTurnMeasurement struct {
	bytes, metadata int64
	records         int
}

// These are explicitly delimited synthetic turns, not inferred human turns.
// Each sends 64 128-byte output chunks and flushes stdout once per chunk.
func measureDiagnosticTurns(t *testing.T, turns int) diagnosticTurnMeasurement {
	t.Helper()
	t.Setenv("PAIR_WRAP_EVENTS", "1")
	t.Setenv("PAIR_SLUG", "0")
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "wrap-events-budget.jsonl")
	var payload, metadata atomic.Int64
	p := &proxy{wrapEventsPath: path, agentBasename: "codex"}
	p.diagnosticOptions = func() diagnosticlog.Options {
		return diagnosticlog.Options{FlushTicks: make(chan time.Time), ObserveIO: func(effect diagnosticlog.IOEffect) {
			if effect.Operation != "write" {
				return
			}
			if effect.Path == path {
				payload.Add(int64(effect.Bytes))
			} else {
				metadata.Add(int64(effect.Bytes))
			}
		}}
	}
	p.openWrapEvents([]string{"owned-diagnostic-turn-fixture"})
	writer, ok := p.wrapEventsFD.(*diagnosticlog.BufferedWriter)
	if !ok {
		t.Fatal("production buffered diagnostic owner did not open")
	}
	t.Cleanup(func() { _ = p.closeWrapEvents() })
	payload.Store(0)
	metadata.Store(0)
	raw, err := os.Create(filepath.Join(dir, "capture-owner"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	p.scrollbackFD = raw
	p.capture = newCaptureBuffer(io.Discard, io.Discard)
	defer p.capture.Close()
	p.stdoutPump = newStdoutPump(io.Discard)
	chunk := bytes.Repeat([]byte("x"), 128)
	var rolling []byte
	for turn := range turns {
		at := time.Unix(int64(turn+1), 0)
		p.now = func() time.Time { return at }
		id := fmt.Sprintf("owned-turn-%d", turn)
		p.processLifecycleRecord(sessionwatch.LifecycleRecord{TurnID: id, Outcome: "started"})
		if !p.notificationLifecycle.Active || p.notificationLifecycle.TurnID != id {
			t.Fatal("production lifecycle did not open the explicit turn")
		}
		for range 64 {
			p.handleChunk(chunk, &rolling)
			p.flushStdout("owned-turn-chunk")
		}
		p.processLifecycleRecord(sessionwatch.LifecycleRecord{TurnID: id, Outcome: "completed"})
		if p.notificationLifecycle.Active || !p.notificationLifecycle.Completed {
			t.Fatal("production lifecycle did not close the explicit turn")
		}
		if err := writer.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.closeWrapEvents(); err != nil {
		t.Fatal(err)
	}
	stats := writer.Stats()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DroppedBytes != 0 || stats.PendingBytes != 0 || stats.WrittenBytes != int64(len(data)) || payload.Load() != int64(len(data)) {
		t.Fatalf("measurement lost data: stats=%+v file=%d payload=%d", stats, len(data), payload.Load())
	}
	labels := map[string]int{}
	records, rawBytes := 0, 0
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var record struct {
			Label  string `json:"label"`
			RawLen int    `json:"raw_len"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}
		records++
		labels[record.Label]++
		if record.Label == "master-chunk" {
			rawBytes += record.RawLen
		}
	}
	for _, label := range []string{"master-chunk", "stdout-queue", "scrollback-write"} {
		if labels[label] != turns*64 {
			t.Fatalf("%s count=%d want=%d", label, labels[label], turns*64)
		}
	}
	if rawBytes != turns*64*128 || records > turns*(4*64+2)+2 || len(data) > turns*(128<<10)+(8<<10) || metadata.Load() > int64(turns+2)*(8<<10) {
		t.Fatalf("turn budget exceeded: turns=%d raw=%d records=%d diagnostic=%d metadata=%d", turns, rawBytes, records, len(data), metadata.Load())
	}
	t.Logf("owned turns=%d chunks/turn=64 raw-bytes/turn=8192 diagnostic-bytes=%d bytes/turn=%.1f metadata-bytes=%d records=%d dropped=%d", turns, len(data), float64(len(data))/float64(turns), metadata.Load(), records, stats.DroppedBytes)
	return diagnosticTurnMeasurement{int64(len(data)), metadata.Load(), records}
}

func TestDiagnosticTurnCountedBudget(t *testing.T) {
	var measurements []diagnosticTurnMeasurement
	for _, turns := range []int{8, 16} {
		t.Run(fmt.Sprint(turns), func(t *testing.T) {
			measurements = append(measurements, measureDiagnosticTurns(t, turns))
		})
	}
	if len(measurements) != 2 {
		t.Fatal("both turn workloads must produce measurements")
	}
	first, second := measurements[0], measurements[1]
	if second.bytes > 2*first.bytes+(16<<10) || second.metadata > 2*first.metadata+(16<<10) || second.records > 2*first.records+2 {
		t.Fatalf("doubling turns exceeded linear budget: first=%+v second=%+v", first, second)
	}
}
