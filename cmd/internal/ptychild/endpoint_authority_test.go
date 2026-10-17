package ptychild

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestChildHasOneTerminalAuthority(t *testing.T) {
	typ := reflect.TypeOf(Child{})
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Type == reflect.TypeOf((*Screen)(nil)) {
			t.Fatal("Child still retains duplicate Screen parser")
		}
	}
}

func TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction(t *testing.T) {
	child := NewFakeChild(nil)
	t.Cleanup(func() { child.Close() })
	child.ring = NewRing(32)
	child.Feed([]byte("\x1b[?1049h\x1b[?1002h\x1b[?1006h\x1b[HKEPT"))
	child.Feed([]byte(strings.Repeat("\x1b[2;1Hx", 64)))
	if bytes.Contains(child.Snapshot(), []byte("KEPT")) {
		t.Fatal("fixture did not evict original paint")
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var row strings.Builder
	for _, c := range frame.Cells[:frame.Geometry.Cols] {
		row.WriteString(c.Content)
	}
	if !strings.HasPrefix(row.String(), "KEPT") {
		t.Fatalf("eviction lost frame: %q", row.String())
	}
	modes := child.Endpoint().Modes()
	if !modes.AltScreen || modes.Tracking != 1002 || !modes.SGR {
		t.Fatalf("eviction lost modes: %+v", modes)
	}
	if err := child.FlushOutput(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestChildNotificationIsTypedOnceIndependentOfRawRetention(t *testing.T) {
	envelope := notifyosc.Encode("review ready")
	stream := append(append([]byte("before"), envelope...), []byte("after")...)
	for capacity := 1; capacity <= len(stream); capacity++ {
		child := NewFakeChild(nil)
		child.ring = NewRing(capacity)
		var notifications []terminal.Effect
		child.SetSink(func(_ context.Context, b OutputBatch) error {
			for _, e := range b.Terminal.Effects {
				if e.Kind == terminal.NotificationEffect {
					notifications = append(notifications, e)
				}
			}
			return nil
		})
		for _, b := range stream {
			child.Feed([]byte{b})
		}
		if err := child.FlushOutput(context.Background()); err != nil {
			child.Close()
			t.Fatal(err)
		}
		frame, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			child.Close()
			t.Fatal(err)
		}
		var row strings.Builder
		for _, c := range frame.Cells[:frame.Geometry.Cols] {
			row.WriteString(c.Content)
		}
		if row.String() != "beforeafter" || len(notifications) != 1 || notifications[0].Text != "review ready" {
			child.Close()
			t.Fatalf("capacity%d row%q notifications%+v", capacity, row.String(), notifications)
		}
		if !bytes.Equal(child.Snapshot(), stream[len(stream)-capacity:]) {
			child.Close()
			t.Fatalf("raw diagnostic retention changed at%d", capacity)
		}
		child.Close()
	}
}

func TestChildQueriesReplyOnlyToOriginEndpoint(t *testing.T) {
	child := NewFakeChild(nil)
	defer child.Close()
	child.Feed([]byte("\x1b[cvisible"))
	if err := child.Endpoint().Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(bytes.Join(child.Writes(), nil)); got != "\x1b[?1;2c" {
		t.Fatalf("origin query reply=%q", got)
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var row strings.Builder
	for _, c := range frame.Cells[:frame.Geometry.Cols] {
		row.WriteString(c.Content)
	}
	if row.String() != "visible" {
		t.Fatalf("query leaked into display cells: %q", row.String())
	}
}

func TestChildIngestBytesGeneratedPartitions(t *testing.T) {
	for seed := int64(0); seed < 12; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			stream := make([]byte, 512+rng.Intn(1024))
			for i := range stream {
				stream[i] = byte(32 + rng.Intn(95))
			}
			stream = append(stream, []byte("\x1b[31m世界\x1b[0m\x00")...)
			child := NewFakeChild(nil)
			t.Cleanup(func() { child.Close() })
			child.ring = NewRing(1 + rng.Intn(16))
			if got := child.IngestBytes(); got != 0 {
				t.Fatalf("initial total %d", got)
			}
			var admitted uint64
			for offset := 0; offset < len(stream); {
				n := min(1+rng.Intn(97), len(stream)-offset)
				chunk := stream[offset : offset+n]
				child.Feed(chunk)
				admitted += uint64(n)
				if got := child.IngestBytes(); got != admitted {
					t.Fatalf("total %d, want %d", got, admitted)
				}
				offset += n
			}
			child.Feed(nil)
			if got := child.IngestBytes(); got != uint64(len(stream)) {
				t.Fatalf("final total %d, stream length %d", got, len(stream))
			}
			if uint64(len(child.Snapshot())) >= admitted {
				t.Fatal("fixture did not evict diagnostic bytes")
			}
		})
	}
}

func TestChildIngestBytesCountsFailedAdmission(t *testing.T) {
	child := NewFakeChild([]byte("seed"))
	t.Cleanup(func() { child.Close() })
	other := NewFakeChild([]byte("other child"))
	t.Cleanup(func() { other.Close() })
	child.SetSink(func(context.Context, OutputBatch) error { return nil })
	child.publication.close()
	if err := child.ingest([]byte("publication failed")); err == nil {
		t.Fatal("expected publication cancellation")
	}
	child.Endpoint().EndInput()
	if err := child.ingest([]byte("input ended")); err == nil {
		t.Fatal("expected ended-input refusal")
	}
	want := uint64(len("seed") + len("publication failed") + len("input ended"))
	if got := child.IngestBytes(); got != want {
		t.Fatalf("total %d, want all admitted bytes %d", got, want)
	}
	if got := other.IngestBytes(); got != uint64(len("other child")) {
		t.Fatalf("independent child total %d", got)
	}
}

func TestChildIngestBytesConcurrentMonotone(t *testing.T) {
	child := NewFakeChild(nil)
	t.Cleanup(func() { child.Close() })
	const writers, chunks = 4, 128
	var wg sync.WaitGroup
	var want uint64
	for writer := 0; writer < writers; writer++ {
		n := writer + 1
		want += uint64(n * chunks)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < chunks; i++ {
				child.Feed(bytes.Repeat([]byte("x"), n))
			}
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	var previous uint64
	timeout := testwait.After(t)
	for {
		got := child.IngestBytes()
		if got < previous || got > want {
			t.Fatalf("total %d outside [%d,%d]", got, previous, want)
		}
		previous = got
		select {
		case <-done:
			if got := child.IngestBytes(); got != want {
				t.Fatalf("final total %d, want %d", got, want)
			}
			return
		case <-timeout:
			t.Fatal("timed out waiting for concurrent ingestion")
		default:
			runtime.Gosched()
		}
	}
}

func TestChildIngestBytesBeforeFeedLock(t *testing.T) {
	child := NewFakeChild(nil)
	t.Cleanup(func() { child.Close() })
	child.feedMu.Lock()
	done := make(chan struct{})
	go func() { child.Feed([]byte("queued")); close(done) }()
	admitted := testwait.Wait(t, func() bool { return child.IngestBytes() != 0 })
	got := child.IngestBytes()
	// Release before reporting any failure so cleanup and the waiting feed can
	// complete even when admission was incorrectly placed behind this lock.
	child.feedMu.Unlock()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("timed out waiting for queued feed")
	}
	if !admitted {
		t.Fatal("timed out waiting for ingest admission before feed lock")
	}
	if got != uint64(len("queued")) {
		t.Fatalf("admission total behind feed lock: %d", got)
	}
}

// This measures the complete ingest seam, not emulator CPU attribution.
func BenchmarkChildIngestBytes(b *testing.B) {
	for _, n := range []int{1, 64, 4096} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			child := NewFakeChild(nil)
			defer child.Close()
			chunk := bytes.Repeat([]byte("x"), n)
			b.SetBytes(int64(n))
			b.ResetTimer()
			for b.Loop() {
				child.Feed(chunk)
			}
		})
	}
}
