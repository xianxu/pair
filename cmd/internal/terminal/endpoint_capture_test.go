package terminal

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// Isolate capture-owned storage from the intentionally independent consumer
// clones and parser/history allocations. Both axes double independently.
func TestEndpointCaptureAllocationCounts(t *testing.T) {
	for _, cols := range []int{80, 160} {
		for _, captures := range []int{64, 128} {
			t.Run(fmt.Sprintf("cols%d/captures%d", cols, captures), func(t *testing.T) {
				e, err := NewEndpoint("count", Geometry{cols, 24}, ttyio.NewFake())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { e.Close() })
				e.mu.Lock()
				defer e.mu.Unlock()
				e.published = e.capture()
				allocs := testing.AllocsPerRun(3, func() {
					for i := 0; i < captures; i++ {
						e.published = e.capture()
					}
				})
				t.Logf("%d cells, %d captures: %.0f capture allocations", cols*24, captures, allocs)
				if allocs != 0 {
					t.Fatalf("fixed-geometry capture allocated %.0f objects; want zero after warmup", allocs)
				}
			})
		}
	}
}

func TestEndpointCaptureSyncReusesPrivateBacking(t *testing.T) {
	e, _ := newEndpointTest(t, "sync")
	now := time.Unix(100, 0)
	if _, err := e.Snapshot(now); err != nil {
		t.Fatal(err)
	}
	cells, rows := &e.published.Cells[0], &e.published.Rows[0]
	for i := 0; i < 128; i++ {
		if _, err := e.Feed([]byte("\x1b[?2026h\rX\x1b[?2026l"), now); err != nil {
			t.Fatal(err)
		}
		if _, err := e.Snapshot(now); err != nil {
			t.Fatal(err)
		}
		if &e.published.Cells[0] != cells || &e.published.Rows[0] != rows {
			t.Fatalf("synchronized capture %d replaced fixed-geometry backing", i)
		}
	}
}

func TestEndpointCaptureOverwritesErasedCellsAndRetainsConsumers(t *testing.T) {
	e, _ := newEndpointTest(t, "copies")
	now := time.Unix(100, 0)
	if _, err := e.Feed([]byte("\x1b[31m\x1b]8;;https://example.com\x1b\\ABCDEFGH!"), now); err != nil {
		t.Fatal(err)
	}
	old, err := e.Publication(now)
	if err != nil {
		t.Fatal(err)
	}
	wantOld := old.Clone()
	if old.Frame.Cells[0].Link.URL == "" || old.Frame.Rows[0].UsedColumns == 0 {
		t.Fatal("fixture did not populate link and row metadata")
	}
	if _, err := e.Feed([]byte("\x1b]8;;\x1b\\\x1b[0m\x1b[2J\x1b[H"), now); err != nil {
		t.Fatal(err)
	}
	fresh, err := e.Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range fresh.Cells {
		if !reflect.DeepEqual(c, Cell{Width: 1}) {
			t.Fatalf("stale cell %d: %+v", i, c)
		}
	}
	for i, row := range fresh.Rows {
		if row != (RowMetadata{}) {
			t.Fatalf("stale row %d: %+v", i, row)
		}
	}
	if !reflect.DeepEqual(old, wantOld) {
		t.Fatal("recapture mutated retained publication")
	}
	fresh.Cells[0].Content = "consumer edit"
	fresh.Rows[0].UsedColumns = 8
	again, err := e.Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	if again.Cells[0].Content != "" || again.Rows[0] != (RowMetadata{}) {
		t.Fatal("consumer edit mutated private backing")
	}
}

func TestEndpointCaptureZerosAbsentBackendCells(t *testing.T) {
	e, _ := newEndpointTest(t, "absent")
	now := time.Unix(100, 0)
	if _, err := e.Feed([]byte("ABCDEFGH\r\nIJKLMNOP\r\nQRSTUVWX"), now); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Snapshot(now); err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	// Deliberately give the actual backend a smaller screen than the endpoint:
	// normal public Resize keeps them coupled. This exercises CellAt's nil
	// fallback without adding a parser double or production test seam.
	if err := e.backend.ResizeChecked(1, 1); err != nil {
		t.Fatal(err)
	}
	e.published = e.capture()
	for y := 0; y < e.geometry.Rows; y++ {
		for x := 0; x < e.geometry.Cols; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if c := e.published.Cells[y*e.geometry.Cols+x]; !reflect.DeepEqual(c, Cell{}) {
				t.Fatalf("absent cell %d,%d retained %+v", x, y, c)
			}
		}
		if y > 0 && e.published.Rows[y] != (RowMetadata{}) {
			t.Fatalf("absent row %d retained metadata", y)
		}
	}
}

func TestEndpointCaptureResizeWhileHeld(t *testing.T) {
	e, _ := newEndpointTest(t, "resize")
	now := time.Unix(100, 0)
	if _, err := e.Feed([]byte("A\x1b[?2026h\rB"), now); err != nil {
		t.Fatal(err)
	}
	old, err := e.Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	wantOld := old.Clone()
	cells, rows := &e.published.Cells[0], &e.published.Rows[0]
	if err := e.Resize(e.geometry, func(Geometry) error { t.Fatal("no-op acknowledged"); return nil }); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("PTY refused")
	if err := e.Resize(Geometry{10, 5}, func(Geometry) error { return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	held, err := e.Snapshot(now.Add(100 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held, wantOld) || e.syncState != 1 || e.syncStarted != now {
		t.Fatal("no-op/refused resize changed hold")
	}
	if &e.published.Cells[0] != cells || &e.published.Rows[0] != rows {
		t.Fatal("no-op/refused resize changed backing")
	}
	for _, geometry := range []Geometry{{10, 5}, {4, 2}, {8, 4}} {
		epoch := e.epoch
		calls := 0
		if err := e.Resize(geometry, func(g Geometry) error {
			calls++
			if g != geometry {
				t.Fatal(g)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if calls != 1 || e.syncState != 0 {
			t.Fatal("successful resize did not acknowledge and cancel hold")
		}
		f, err := e.Snapshot(now)
		if err != nil {
			t.Fatal(err)
		}
		if f.Geometry != geometry || f.GeometryEpoch <= epoch {
			t.Fatal("geometry/epoch not advanced")
		}
		if len(e.published.Cells) != geometry.Cols*geometry.Rows || cap(e.published.Cells) != len(e.published.Cells) || len(e.published.Rows) != geometry.Rows || cap(e.published.Rows) != geometry.Rows {
			t.Fatal("capture retained storage for another geometry")
		}
		if &e.published.Cells[0] == cells || &e.published.Rows[0] == rows {
			t.Fatal("changed geometry retained old backing")
		}
		cells, rows = &e.published.Cells[0], &e.published.Rows[0]
		if !reflect.DeepEqual(old, wantOld) {
			t.Fatal("resize mutated retained frame")
		}
	}
}

func BenchmarkEndpointCapture(b *testing.B) {
	for _, cols := range []int{80, 160} {
		b.Run(fmt.Sprintf("cols%d", cols), func(b *testing.B) {
			e, err := NewEndpoint("bench", Geometry{cols, 24}, ttyio.NewFake())
			if err != nil {
				b.Fatal(err)
			}
			defer e.Close()
			e.mu.Lock()
			defer e.mu.Unlock()
			e.published = e.capture()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				e.published = e.capture()
			}
		})
	}
}
