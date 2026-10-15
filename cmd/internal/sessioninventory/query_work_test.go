package sessioninventory

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type queryWorkCounts struct {
	mu             sync.Mutex
	reads, decoded map[Artifact]int64
}

func (c *queryWorkCounts) ReadWork(a Artifact, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.reads == nil {
		c.reads = map[Artifact]int64{}
	}
	c.reads[a] += n
}
func (c *queryWorkCounts) DecodeWork(a Artifact, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.decoded == nil {
		c.decoded = map[Artifact]int64{}
	}
	c.decoded[a] += n
}
func TestOSRuntimeQueryWorkObserver(t *testing.T) {
	dir := t.TempDir()
	rt := NewOSRuntime(t.TempDir(), dir)
	configurable, ok := any(rt).(interface {
		WithWorkObserver(QueryWorkObserver) OSRuntime
	})
	if !ok {
		t.Fatal("OSRuntime cannot observe actual query work")
	}
	counts := &queryWorkCounts{}
	rt = configurable.WithWorkObserver(counts)
	a := Artifact{StorageRoot: "pair-data", RelativePath: "body.jsonl"}
	if err := os.WriteFile(filepath.Join(dir, a.RelativePath), []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.ReadFile(a, -1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := rt.ReadAt(a, 2, 3); err != nil {
		t.Fatal(err)
	}
	if counts.reads[a] != 9 || counts.decoded[a] != 0 {
		t.Fatalf("body IO incorrectly counted as decode: %#v %#v", counts.reads, counts.decoded)
	}
	if _, err := rt.ReadFile(a, 2); !errors.Is(err, ErrReadLimit) {
		t.Fatal(err)
	}
	if counts.reads[a] != 12 {
		t.Fatalf("overflow bytes actually read weren't counted: %d", counts.reads[a])
	}
	// A separate runtime owns no observer, even when its source is identical.
	if _, err := NewOSRuntime(t.TempDir(), dir).ReadFile(a, -1); err != nil {
		t.Fatal(err)
	}
	if counts.reads[a] != 12 {
		t.Fatal("observer leaked across runtimes")
	}
}
func TestOSRuntimeCatalogCountsIOAndStrictDecode(t *testing.T) {
	dir := t.TempDir()
	rt := NewOSRuntime(t.TempDir(), dir)
	configurable, ok := any(rt).(interface {
		WithWorkObserver(QueryWorkObserver) OSRuntime
	})
	if !ok {
		t.Fatal("catalog query observer unavailable")
	}
	counts := &queryWorkCounts{}
	rt = configurable.WithWorkObserver(counts)
	// ResolveSelectedScope accepts an absolute directory, as normal OSRuntime.
	const content = `{"version":1,"generation":0,"entries":[]}`
	const name = "session-inventory-catalog.json"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := rt.LoadSessionInventoryCatalog()
	if err != nil {
		t.Fatal(err)
	}
	a := Artifact{StorageRoot: "pair-data", RelativePath: name}
	if counts.reads[a] != int64(len(content)) || counts.decoded[a] != int64(len(content)) {
		t.Fatalf("catalog observer disconnected: %#v %#v", counts.reads, counts.decoded)
	}
	const malformed = `{"version":1,"generation":0,"entries":[],"unexpected":true}`
	if err := os.WriteFile(filepath.Join(dir, name), []byte(malformed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.LoadSessionInventoryCatalog(); !errors.Is(err, ErrCatalogCorrupt) {
		t.Fatalf("strict catalog semantics lost: %v", err)
	}
	want := int64(len(content) + len(malformed))
	if counts.reads[a] != want || counts.decoded[a] != want {
		t.Fatal("rejected decode work wasn't observed")
	}
}

func TestNativeValidationObserverCountsActualParserInputs(t *testing.T) {
	home := t.TempDir()
	rt := NewOSRuntime(home, t.TempDir())
	counts := &queryWorkCounts{}
	rt = rt.WithWorkObserver(counts)
	const id = "11111111-1111-4111-8111-111111111111"
	const row = `{"type":"user","sessionId":"` + id + `","timestamp":"2026-10-14T10:00:00Z"}`
	a := Artifact{StorageRoot: "claude-projects", RelativePath: "project/" + id + ".jsonl"}
	path := filepath.Join(home, ".claude", "projects", filepath.FromSlash(a.RelativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(row+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	observed, diagnostics := ObserveNamedArtifacts(rt, AgentClaude, []Artifact{a})
	if len(observed) != 1 {
		t.Fatalf("observations=%v diagnostics=%v", observed, diagnostics)
	}
	validated, diagnostics := ValidateTargetWork(rt, AgentClaude, observed)
	if len(validated) != 1 {
		t.Fatalf("validated=%v diagnostics=%v", validated, diagnostics)
	}
	var read, decode int64
	for key, n := range counts.reads {
		if key.StorageRoot == a.StorageRoot && key.RelativePath == a.RelativePath {
			read += n
		}
	}
	for key, n := range counts.decoded {
		if key.StorageRoot == a.StorageRoot && key.RelativePath == a.RelativePath {
			decode += n
		}
	}
	// Scanner and event normalization independently parse the framed record.
	if read != int64(len(row)+1) || decode != 2*int64(len(row)) {
		t.Fatalf("native actual bytes read=%d decode=%d", read, decode)
	}
	// Re-decoding cached raw state is work even when no new IO occurs.
	before := read
	_, _ = decodeScannerStateObserved(rt, a, []byte(`{invalid}`))
	read = 0
	decode = 0
	for key, n := range counts.reads {
		if key.StorageRoot == a.StorageRoot && key.RelativePath == a.RelativePath {
			read += n
		}
	}
	for key, n := range counts.decoded {
		if key.StorageRoot == a.StorageRoot && key.RelativePath == a.RelativePath {
			decode += n
		}
	}
	if read != before || decode != 2*int64(len(row))+int64(len(`{invalid}`)) {
		t.Fatal("cached-byte decoding escaped accounting")
	}
}
