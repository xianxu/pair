package sessionwatch

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// Double both foreign tree size and prior authorized history. Discovery
// counters are sampled after binding; steady idle work must remain identical.
func TestRunBoundFollowWorkIndependentOfTreeAndHistory(t *testing.T) {
	for _, n := range []int{4000, 8000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			dataDir := t.TempDir()
			native := &boundWorkRuntime{FakeRuntime: sessioninventorytest.NewFakeRuntime()}
			root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
			native.AddRoot(root)
			for i := 0; i < n; i++ {
				native.PutMetadataFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: fmt.Sprintf("2026/08/27/rollout-old-%08x-1111-4111-8111-%012x.jsonl", i+1, i+1)}, StableFileID: sessioninventory.StableFileID(fmt.Sprint(i)), Size: 30 << 20})
			}
			sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
			relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
			artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
			owned := codexLifecycleRound(sid, "turn", "hello")
			owned = append(owned, bytes.Repeat([]byte(`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"), n/4)...)
			native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "own", GenerationToken: "gen:1", MutationToken: "one"}, owned)
			native.SetProcess("1234", "native", nil, []string{filepath.Join(root.Path, relative)})
			paths := mustScopedPaths(t, dataDir, "work")
			base := newWatcherRuntime(native)
			rt := &boundWorkWatcher{watcherRuntime: base}
			catalog := &boundWorkCatalog{CatalogStoreRuntime: sessioninventory.CatalogOSRuntime{}}
			rt.catalogRuntime = catalog
			rt.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
			rt.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\nhello\n\n---\n\n")
			rt.files[paths.AgentPID()] = []byte("1234\n")
			rt.modTimes[paths.AgentPID()] = rt.now
			rt.identities["1234"] = "pair"
			sleeps, coldLists, coldReads := 0, 0, 0
			var coldBytes, coldDecode, coldLedger int64
			var coldCatalog, coldWrites, coldStats int
			rt.onSleep = func() {
				sleeps++
				if sleeps == 1 {
					coldLists = native.OperationCount(sessioninventorytest.OperationListFiles, "")
					coldReads = native.OperationCount(sessioninventorytest.OperationReadAt, "")
					coldBytes, coldDecode, coldLedger = native.readBytes, native.decoded, rt.ledgerBytes
					coldCatalog, coldWrites = catalog.reads, catalog.writes
					coldStats = native.OperationCount(sessioninventorytest.OperationStatFile, artifact.StorageRoot+":"+artifact.RelativePath)
				}
				if sleeps == 8 {
					rt.identities["1234"] = "exited"
				}
			}
			if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, Poll: time.Millisecond, FollowLifecycle: true, AppendLifecycle: func(string, LifecycleRecord) error { return nil }}, rt); err != nil {
				t.Fatal(err)
			}
			if sleeps != 8 {
				t.Fatalf("follow did not stay active: sleeps=%d", sleeps)
			}
			if delta := native.OperationCount(sessioninventorytest.OperationListFiles, "") - coldLists; delta != 0 {
				t.Errorf("bound follow tree listings=%d, want0 (cold%d)", delta, coldLists)
			}
			if delta := native.OperationCount(sessioninventorytest.OperationReadAt, "") - coldReads; delta != 0 {
				t.Errorf("unchanged bound follow body reads=%d, want0 (cold%d)", delta, coldReads)
			}
			if native.readBytes != coldBytes || native.decoded != coldDecode || rt.ledgerBytes != coldLedger {
				t.Errorf("idle bytes read/decode/ledger changed: %d/%d/%d", native.readBytes-coldBytes, native.decoded-coldDecode, rt.ledgerBytes-coldLedger)
			}
			if catalog.reads != coldCatalog || catalog.writes != coldWrites {
				t.Errorf("idle catalog reads/writes=%d/%d", catalog.reads-coldCatalog, catalog.writes-coldWrites)
			}
			if n := native.OperationCount(sessioninventorytest.OperationStatFile, artifact.StorageRoot+":"+artifact.RelativePath) - coldStats; n != 7 {
				t.Errorf("idle own stats=%d want7 independent of corpus", n)
			}

		})
	}
}

// Count decoder input independently of reads: warm authority must do neither.
type boundWorkRuntime struct {
	*sessioninventorytest.FakeRuntime
	readBytes, decoded int64
}

func (r *boundWorkRuntime) ReadAt(a sessioninventory.Artifact, off, limit int64) ([]byte, bool, error) {
	b, eof, err := r.FakeRuntime.ReadAt(a, off, limit)
	r.readBytes += int64(len(b))
	return b, eof, err
}
func (r *boundWorkRuntime) ReadFile(a sessioninventory.Artifact, limit int64) ([]byte, error) {
	b, err := r.FakeRuntime.ReadFile(a, limit)
	r.readBytes += int64(len(b))
	return b, err
}
func (r *boundWorkRuntime) DecodeWork(_ sessioninventory.Artifact, n int64) { r.decoded += n }

type boundWorkWatcher struct {
	*watcherRuntime
	ledgerBytes int64
}

func (r *boundWorkWatcher) ReadFile(path string) ([]byte, error) {
	b, err := r.watcherRuntime.ReadFile(path)
	if filepath.Ext(path) == ".jsonl" {
		r.ledgerBytes += int64(len(b))
	}
	return b, err
}

type boundWorkCatalog struct {
	sessioninventory.CatalogStoreRuntime
	reads, writes int
}

func (r *boundWorkCatalog) ReadFile(path string) ([]byte, error) {
	r.reads++
	return r.CatalogStoreRuntime.ReadFile(path)
}
func (r *boundWorkCatalog) Rename(a, b string) error {
	r.writes++
	return r.CatalogStoreRuntime.Rename(a, b)
}
