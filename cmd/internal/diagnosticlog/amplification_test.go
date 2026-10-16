package diagnosticlog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBufferedRealWriterAmplificationInvariant(t *testing.T) {
	path, _, opts := fixture(t)
	canonicalPath, err := canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	opts.FlushTicks = make(chan time.Time)
	var mu sync.Mutex
	payloadBytes, metadataBytes, writes, renames, syncs := 0, 0, 0, 0, 0
	opts.appendWrite = func(f *os.File, b []byte) (int, error) {
		n, err := f.Write(b)
		mu.Lock()
		payloadBytes += n
		writes++
		mu.Unlock()
		return n, err
	}
	opts.ObserveIO = func(e IOEffect) {
		mu.Lock()
		defer mu.Unlock()
		switch e.Operation {
		case "write":
			if e.Path != canonicalPath {
				metadataBytes += e.Bytes
				writes++
			}
		case "rename":
			if e.Path == filepath.Join(directory(canonicalPath), "state.json") {
				renames++
			}
		case "sync":
			syncs++
		}
	}
	w, err := OpenBuffered(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	// Opening publishes actual ownership/durable creation; account for the hot
	// stream separately, without subtracting any cost from subsequent writes.
	mu.Lock()
	payloadBytes = 0
	metadataBytes = 0
	writes = 0
	renames = 0
	syncs = 0
	mu.Unlock()
	var want bytes.Buffer
	for i := 0; i < 1000; i++ {
		record := []byte(fmt.Sprintf("{\"label\":\"synthetic\",\"seq\":%d,\"raw_len\":32}\n", i))
		want.Write(record)
		if n, err := w.Write(record); err != nil || n != len(record) {
			t.Fatalf("admission %d=%d %v", i, n, err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("stream changed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if payloadBytes != want.Len() {
		t.Fatalf("counter disconnected: actual payload=%d logical=%d", payloadBytes, want.Len())
	}
	total := payloadBytes + metadataBytes
	if total >= 3*want.Len() {
		t.Fatalf("amplification %.3fx >=3; payload=%d metadata=%d", float64(total)/float64(want.Len()), payloadBytes, metadataBytes)
	}
	if metadataBytes > 8192 || writes != 3 || renames != 1 || syncs != 0 {
		t.Fatalf("not one compact flush: metadata=%d writes=%d renames=%d syncs=%d", metadataBytes, writes, renames, syncs)
	}
	t.Logf("1000 records: payload=%d metadata=%d total=%d ratio=%.4fx writes=%d checkpoint-renames=%d hot-syncs=%d", payloadBytes, metadataBytes, total, float64(total)/float64(payloadBytes), writes, renames, syncs)
}
