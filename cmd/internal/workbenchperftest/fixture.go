package workbenchperftest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type Owner struct{ Scope, Tag, Agent, NativeID, ScopeDir, Ledger, Transcript string }
type Fixture struct {
	Home, DataDir string
	Owners        []Owner
	Counts        *Counter
}

// NewFixture streams actual ledger bytes for the lower-level and UI consumers;
// ledgers are valid launch/binding JSONL, including at the 20 x 30MB scale.
// Message tests can reuse these artifacts without consulting ledger contents.
// Sizes smaller than the required records retain the complete records.
func NewFixture(t testing.TB, slots, ledgerBytes int) Fixture {
	t.Helper()
	root := t.TempDir()
	f := Fixture{Home: filepath.Join(root, "home"), DataDir: filepath.Join(root, "pair"), Counts: &Counter{}}
	roots := sessioninventory.NewOSRuntime(f.Home, f.DataDir).NativeRoots(sessioninventory.AgentCodex)
	if len(roots) != 1 {
		t.Fatalf("expected one Codex native root, got %d", len(roots))
	}
	native := filepath.Join(roots[0].Path, "2026", "10", "14")
	if err := os.MkdirAll(native, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < slots; i++ {
		o := Owner{Scope: fmt.Sprintf("%016x", i+1), Tag: fmt.Sprintf("work-%02d", i+1), Agent: "codex", NativeID: fmt.Sprintf("019d1111-1111-7111-8111-%012d", i+1)}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: o.Scope, Tag: o.Tag})
		if err != nil {
			t.Fatal(err)
		}
		o.ScopeDir = paths.ScopeDir()
		if err := os.MkdirAll(o.ScopeDir, 0700); err != nil {
			t.Fatal(err)
		}
		o.Ledger = paths.Ledger()
		records := fmt.Sprintf("{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":%q,\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":%q,\"request_origin\":\"chosen-id\",\"baseline_complete\":true}\n{\"v\":3,\"kind\":\"binding\",\"scope_key\":%q,\"tag\":%q,\"agent\":\"codex\",\"launch_ordinal\":1,\"root_native_id\":%q,\"confirmation_reason\":\"chosen-id\"}\n", o.Scope, o.Tag, o.NativeID, o.Scope, o.Tag, o.NativeID)
		writeLedger(t, o.Ledger, records, ledgerBytes)
		o.Transcript = filepath.Join(native, "rollout-test-"+o.NativeID+".jsonl")
		body := fmt.Sprintf("{\"timestamp\":\"2026-10-14T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"parent_thread_id\":null,\"source\":\"cli\"}}\n", o.NativeID)
		if err := os.WriteFile(o.Transcript, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		f.Owners = append(f.Owners, o)
	}
	return f
}

func writeLedger(t testing.TB, path, records string, size int) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	padding := size - len(records)
	chunk := strings.Repeat(" ", 64<<10)
	for padding > 0 {
		n := len(chunk)
		if padding < n {
			n = padding
		}
		if _, err := file.WriteString(chunk[:n]); err != nil {
			t.Fatal(err)
		}
		padding -= n
	}
	if _, err := file.WriteString(records); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
