package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
)

type resourceDiscard struct{}

func (resourceDiscard) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return len(p), nil
}

// TestTerminalResourceProbe is an explicit measurement, not a machine-sensitive
// CI assertion. Run in an otherwise idle process:
// PAIR_TERMINAL_RESOURCE_PROBE=1 go test ./cmd/internal/terminal -run '^TestTerminalResourceProbe$' -count=1 -v
// HeapAlloc is live Go heap after GC, not RSS. The workload has bounded ordinary
// metadata; it does not model unique maximum-sized links in every screen cell.
func TestTerminalResourceProbe(t *testing.T) {
	if os.Getenv("PAIR_TERMINAL_RESOURCE_PROBE") != "1" {
		t.Skip("set PAIR_TERMINAL_RESOURCE_PROBE=1 for isolated resource measurements")
	}
	for _, g := range []Geometry{{80, 24}, {240, 80}} {
		for _, state := range []string{"empty", "typical", "history-saturated"} {
			for _, kind := range []string{"backend", "endpoint+published+caller-frame"} {
				t.Run(fmt.Sprintf("%dx%d/%s/%s", g.Cols, g.Rows, state, kind), func(t *testing.T) { resourceMeasure(t, g, state, kind) })
			}
		}
	}
}
func resourcePopulate(t *testing.T, g Geometry, state string, write func([]byte) error) {
	t.Helper()
	if state == "empty" {
		return
	}
	send := func(s string) {
		t.Helper()
		if err := write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	row := "\x1b[32mrow é中 " + strings.Repeat("x", g.Cols-12) + "\x1b[0m\r\n"
	send("\x1b]2;synthetic terminal resource workload\a\x1b]7;file://localhost/tmp/pair-resource\a")
	for i := 0; i < g.Rows*2; i++ {
		send(row)
	}
	send("\x1b[?1049h")
	for i := 0; i < g.Rows; i++ {
		send(row)
	}
	send("\x1b[?1049l")
	if state == "history-saturated" {
		for i := 0; i < vt.DefaultLimits().HistoryLines*2; i++ {
			send(row)
		}
	}
}
func resourceMeasure(t *testing.T, g Geometry, state, kind string) {
	const count = 16
	var before, after runtime.MemStats
	backends := make([]*vt.Emulator, 0, count)
	endpoints := make([]*Endpoint, 0, count)
	frames := make([]Frame, 0, count)
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	for i := 0; i < count; i++ {
		if kind == "backend" {
			e, err := vt.NewEmulatorWithLimits(g.Cols, g.Rows, vt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			e.SetReplyWriter(io.Discard)
			backends = append(backends, e)
			resourcePopulate(t, g, state, func(p []byte) error { _, err := e.Write(p); return err })
		} else {
			e, err := NewEndpoint(fmt.Sprintf("resource-%d", i), g, resourceDiscard{})
			if err != nil {
				t.Fatal(err)
			}
			endpoints = append(endpoints, e)
			backends = append(backends, e.backend)
			resourcePopulate(t, g, state, func(p []byte) error { _, err := e.Feed(p, time.Time{}); return err })
			f, err := e.Snapshot(time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			frames = append(frames, f)
		}
	}
	elapsed := time.Since(start)
	runtime.GC()
	runtime.ReadMemStats(&after)
	var total vt.Usage
	for _, e := range backends {
		u := e.Usage()
		total.RetainedBytes += u.RetainedBytes
		total.HistoryBytes += u.HistoryBytes
		total.HistoryCells += u.HistoryCells
		total.ScreenCells += u.ScreenCells
		total.ParserBytes += u.ParserBytes
	}
	report := struct {
		Kind, State                                 string
		Cols, Rows, Endpoints                       int
		LiveHeapDelta, HeapSysDelta, AllocatedBytes int64
		BackendUsage                                vt.Usage
		PopulateMillis                              int64
		GCs                                         uint32
	}{kind, state, g.Cols, g.Rows, count, int64(after.HeapAlloc) - int64(before.HeapAlloc), int64(after.HeapSys) - int64(before.HeapSys), int64(after.TotalAlloc - before.TotalAlloc), total, elapsed.Milliseconds(), after.NumGC - before.NumGC}
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
	runtime.KeepAlive(frames)
	runtime.KeepAlive(endpoints)
	runtime.KeepAlive(backends)
	if kind == "backend" {
		for _, e := range backends {
			e.Close()
		}
	} else {
		for _, e := range endpoints {
			e.Close()
		}
	}
}
