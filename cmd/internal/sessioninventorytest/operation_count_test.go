package sessioninventorytest

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// #420: a root count must include root-level operations, which are keyed by
// the bare root name, not only per-artifact ones.
func TestOperationCountForRootIncludesRootLevelOperations(t *testing.T) {
	f := NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects"}
	f.AddRoot(root)
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/s.jsonl"}
	f.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte("{}\n"))
	_, _ = f.ListFiles(root)
	_, _ = f.StatFile(artifact)
	if got := f.OperationCountForRoot(OperationListFiles, root.Name); got != 1 {
		t.Fatalf("ListFiles root count = %d, want 1", got)
	}
	if got := f.OperationCountForRoot(OperationStatFile, root.Name); got != 1 {
		t.Fatalf("StatFile root count = %d, want 1", got)
	}
	if got := f.OperationCountForRoot(OperationListFiles, "other"); got != 0 {
		t.Fatalf("other root count = %d, want 0", got)
	}
}
