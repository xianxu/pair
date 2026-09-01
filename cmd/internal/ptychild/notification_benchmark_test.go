package ptychild

import (
	"bytes"
	"crypto/sha256"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func BenchmarkScreenNotificationSkip(b *testing.B) {
	var screen Screen
	screen.Feed(append([]byte(notifyosc.Prefix), bytes.Repeat([]byte{'x'}, notifyosc.MaxMessageBytes+1)...))
	_ = screen.TakeOutputParts()
	chunk := bytes.Repeat([]byte{'x'}, 4096)
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.Feed(chunk)
		_ = screen.TakeOutputParts()
	}
}

func TestNotificationSkipTarget(t *testing.T) {
	if os.Getenv("PAIR_MENU_PERF_TARGET") != "m2-max" {
		t.Skip("set PAIR_MENU_PERF_TARGET=m2-max on the target Apple M2 Max")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Fatalf("target protocol requires darwin/arm64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	stop := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func(seed byte) {
			defer workers.Done()
			buf := bytes.Repeat([]byte{seed}, 4096)
			for {
				select {
				case <-stop:
					return
				default:
					_ = sha256.Sum256(buf)
				}
			}
		}(byte(i + 1))
	}
	defer func() { close(stop); workers.Wait() }()

	var screen Screen
	screen.Feed(append([]byte(notifyosc.Prefix), bytes.Repeat([]byte{'x'}, notifyosc.MaxMessageBytes+1)...))
	_ = screen.TakeOutputParts()
	chunk := bytes.Repeat([]byte{'x'}, 4096)
	const iterations = 4096
	started := time.Now()
	for i := 0; i < iterations; i++ {
		screen.Feed(chunk)
		_ = screen.TakeOutputParts()
	}
	elapsed := time.Since(started)
	throughput := float64(len(chunk)*iterations) / elapsed.Seconds() / (1024 * 1024)
	t.Logf("target=m2-max workers=4 bytes=%d elapsed=%s throughput=%.2f MiB/s", len(chunk)*iterations, elapsed, throughput)
	if throughput < 10 {
		t.Fatalf("notification skip throughput %.2f MiB/s, want >=10", throughput)
	}
	if len(screen.notifyCandidate) > len(notifyosc.Prefix)+notifyosc.MaxMessageBytes+1 || screen.Pending() > maxPending {
		t.Fatalf("pending memory exceeded bounds: notification=%d terminal=%d", len(screen.notifyCandidate), screen.Pending())
	}
}
