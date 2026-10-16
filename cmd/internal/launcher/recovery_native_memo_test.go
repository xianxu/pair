package launcher

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// Exercise the real two-phase gate with several owners sharing one memo. The
// process table and file map are immutable; only the production memo is shared.
func TestRecoveryNativeOwnerConcurrentMemo(t *testing.T) {
	root := t.TempDir()
	snapshot := recoveryTerminalSnapshot{Commands: map[int]string{}}
	files := map[string][]byte{}
	for i := 0; i < 12; i++ {
		owner, err := artifactpath.NewStorageOwner(root, "abcdef1234567890", fmt.Sprintf("owner-%d", i))
		if err != nil {
			t.Fatal(err)
		}
		paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
		if err != nil {
			t.Fatal(err)
		}
		pid := 42 + 2*i
		birth := fmt.Sprintf("child-%d", i)
		snapshot.Processes = append(snapshot.Processes, procutil.Process{PID: pid, PPID: 1, Identity: fmt.Sprintf("wrapper-%d", i), Name: "pair-wrap"}, procutil.Process{PID: pid + 1, PPID: pid, Identity: birth, Name: "codex"})
		snapshot.Commands[pid] = "/bin/pair-wrap --scrollback-log " + paths.ScrollbackRaw("codex") + " codex"
		snapshot.Commands[pid+1] = "codex"
		ready, err := readiness.Encode(readiness.ReadyRecord{Tag: owner.Tag, Agent: "codex", Session: "session", Nonce: "nonce", PID: pid + 1, AgentIdentity: birth, LaunchOrdinal: 1})
		if err != nil {
			t.Fatal(err)
		}
		files[AgentReadyPath(owner.Directory(), owner.Tag, "codex")] = []byte(ready)
		row, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: owner.RepoScope, Tag: owner.Tag, Agent: "codex", RequestOrigin: sessionledger.RequestOriginResume, RequestedNativeID: fmt.Sprintf("native-%d", i)})
		if err != nil {
			t.Fatal(err)
		}
		files[paths.Ledger()] = append(row, '\n')
	}
	observe := func(context.Context, string) (recoveryTerminalSnapshot, error) { return snapshot, nil }
	var reads atomic.Int64
	read := func(_ context.Context, path string, _ int64) ([]byte, error) {
		reads.Add(1)
		if raw, ok := files[path]; ok {
			return raw, nil
		}
		return nil, fmt.Errorf("unexpected path %s", path)
	}
	check := RecoveryNativeOwnerCheck{DataDir: root, Agent: "codex", NativeID: "selected"}
	if err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, observe, read, noOrphanOwners); err != nil {
		t.Fatal(err)
	}
	if got := reads.Load(); got != 48 {
		t.Fatalf("metadata reads=%d, want all 12 owners at both fences", got)
	}
}
