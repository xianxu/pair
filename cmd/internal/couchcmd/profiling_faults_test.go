package couchcmd

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// This sink records the effects actually admitted by profileWriter, rather than
// inferring disk bounds from the writer's remaining counter.
type profileCountedSink struct {
	calls, bytes, closes int
	short                bool
}

func (s *profileCountedSink) Write(p []byte) (int, error) {
	s.calls++
	n := len(p)
	if s.short && n > 0 {
		n--
	}
	s.bytes += n
	return n, nil
}
func (s *profileCountedSink) Close() error { s.closes++; return nil }

type profilePartialStorage struct {
	reserves, creates int
	first             profileCountedSink
	cause             error
}

func (s *profilePartialStorage) Reserve(string) error { s.reserves++; return nil }
func (s *profilePartialStorage) Create(string, string) (io.WriteCloser, error) {
	s.creates++
	if s.creates == 2 {
		return nil, s.cause
	}
	return &s.first, nil
}

func TestProfilePartialCreationClosesOwnedOutputs(t *testing.T) {
	cause := errors.New("second output refused")
	storage := &profilePartialStorage{cause: cause}
	backend := &profileTestBackend{}
	window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: filepath.Join(t.TempDir(), "profiles")}, profileDependencies{storage: storage, backend: backend}, nil)
	if window != nil || !errors.Is(err, cause) {
		t.Fatalf("partial reservation: window=%v err=%v", window, err)
	}
	if storage.reserves != 1 || storage.creates != 2 || storage.first.closes != 1 || storage.first.calls != 0 {
		t.Fatalf("partial outputs: %+v", storage)
	}
	if starts, stops, snapshots := backend.counts(); starts != 0 || stops != 0 || snapshots != 0 {
		t.Fatalf("profiler started before all outputs existed: %d/%d/%d", starts, stops, snapshots)
	}
}

func TestProfileAllocationAfterFailureIsRetained(t *testing.T) {
	cause := errors.New("allocation-after write refused")
	clock := newProfileTestClock()
	backend := &profileTestBackend{}
	storage := profileFaultStorage{name: "allocs-after.pprof", writeErr: cause}
	window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: filepath.Join(t.TempDir(), "profiles")}, profileDependencies{storage: storage, backend: backend, clock: clock}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	awaitProfileWait(t, clock, time.Minute)
	clock.advance <- struct{}{}
	awaitProfileWait(t, clock, 30*time.Second)
	clock.advance <- struct{}{}
	select {
	case <-window.done:
	case <-testwait.After(t):
		t.Fatal("profile release did not finish")
	}
	if err := window.Close(); !errors.Is(err, cause) {
		t.Fatalf("allocation-after failure lost: %v", err)
	}
	if starts, stops, snapshots := backend.counts(); starts != 1 || stops != 1 || snapshots != 2 {
		t.Fatalf("release ownership/counts: %d/%d/%d", starts, stops, snapshots)
	}
}

func TestProfileWriterShortWriteStopsFurtherEffects(t *testing.T) {
	sink := &profileCountedSink{short: true}
	writer := &profileWriter{file: sink, remaining: 32}
	if n, err := writer.Write([]byte("abcd")); n != 3 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write: n=%d err=%v", n, err)
	}
	if n, err := writer.Write([]byte("later")); n != 0 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short-write failure not retained: n=%d err=%v", n, err)
	}
	for range 2 {
		if err := writer.Close(); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("close failure: %v", err)
		}
	}
	if sink.calls != 1 || sink.bytes != 3 || sink.closes != 1 {
		t.Fatalf("effects continued after failure: %+v", sink)
	}
}

func TestProfileCancelledParentHasNoEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	storage := &profilePartialStorage{}
	backend := &profileTestBackend{}
	window, err := StartProfileWindow(ctx, ProfileRequest{Dir: filepath.Join(t.TempDir(), "profiles")}, profileDependencies{storage: storage, backend: backend}, nil)
	if window != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled parent: window=%v err=%v", window, err)
	}
	if storage.reserves != 0 || storage.creates != 0 || storage.first.closes != 0 {
		t.Fatalf("cancelled request touched storage: %+v", storage)
	}
	if starts, stops, snapshots := backend.counts(); starts != 0 || stops != 0 || snapshots != 0 {
		t.Fatalf("cancelled request touched profiler: %d/%d/%d", starts, stops, snapshots)
	}
}

func TestProfileWriterSaturationBoundsBytesAndCallsAtNAnd2N(t *testing.T) {
	const n = 16
	for _, count := range []int{n, 2 * n} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			sink := &profileCountedSink{}
			writer := &profileWriter{file: sink, remaining: 35}
			var first error
			for i := range count {
				written, err := writer.Write([]byte("12345678"))
				switch {
				case i < 4:
					if written != 8 || err != nil {
						t.Fatalf("unsaturated write %d: n=%d err=%v", i, written, err)
					}
				case i == 4:
					if written != 3 || err == nil {
						t.Fatalf("saturating write: n=%d err=%v", written, err)
					}
					first = err
				default:
					if written != 0 || err != first {
						t.Fatalf("saturated write %d: n=%d err=%v", i, written, err)
					}
				}
			}
			if err := writer.Close(); err != first {
				t.Fatalf("close lost saturation: %v", err)
			}
			if sink.bytes != 35 || sink.calls != 5 || sink.closes != 1 {
				t.Fatalf("effects grew after saturation for %d inputs: %+v", count, sink)
			}
			// Negative control: bypassing the limiter makes this same sink
			// expose every input call and byte, including the doubled workload.
			unbounded := &profileCountedSink{}
			for range count {
				if _, err := unbounded.Write([]byte("12345678")); err != nil {
					t.Fatal(err)
				}
			}
			if unbounded.calls != count || unbounded.bytes != 8*count || unbounded.bytes <= sink.bytes {
				t.Fatalf("counter failed to expose an unbounded writer: %+v", unbounded)
			}
		})
	}
}
