package sessioninventory_test

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestProofMigratorCoalescesOneNamedRoot(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	migrator := new(sessioninventory.ProofMigrator)
	key := sessioninventory.ProofMigrationKey{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, NativeID: "native-a"}
	work := func() (*sessionledger.AuthorizationProof, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return migrationProof("native-a"), nil
	}
	first := migrator.Request(key, work)
	<-started
	second := migrator.Request(key, work)
	close(release)
	for _, result := range []sessioninventory.ProofMigrationResult{<-first, <-second} {
		if result.Err != nil || result.Proof == nil || result.Proof.RootNativeID != "native-a" {
			t.Fatalf("result=%#v", result)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("work calls=%d, want 1", got)
	}
}

func TestProofMigratorFailurePublishesNoProofAndCanRetry(t *testing.T) {
	t.Parallel()
	migrator := new(sessioninventory.ProofMigrator)
	key := sessioninventory.ProofMigrationKey{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentClaude, NativeID: "native-a"}
	wantErr := errors.New("validation failed")
	if result := <-migrator.Request(key, func() (*sessionledger.AuthorizationProof, error) { return nil, wantErr }); !errors.Is(result.Err, wantErr) || result.Proof != nil {
		t.Fatalf("failed result=%#v", result)
	}
	if result := <-migrator.Request(key, func() (*sessionledger.AuthorizationProof, error) {
		return migrationProof("native-a"), nil
	}); result.Err != nil || result.Proof == nil {
		t.Fatalf("retry result=%#v", result)
	}
}

func TestProofMigratorRejectsEmptyOrWidenedResult(t *testing.T) {
	t.Parallel()
	migrator := new(sessioninventory.ProofMigrator)
	invalid := sessioninventory.ProofMigrationKey{Agent: sessioninventory.AgentCodex}
	if result := <-migrator.Request(invalid, func() (*sessionledger.AuthorizationProof, error) { return nil, nil }); result.Err == nil || result.Proof != nil {
		t.Fatalf("invalid key result=%#v", result)
	}
	key := sessioninventory.ProofMigrationKey{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, NativeID: "native-a"}
	if result := <-migrator.Request(key, func() (*sessionledger.AuthorizationProof, error) {
		return migrationProof("other"), nil
	}); result.Err == nil || result.Proof != nil {
		t.Fatalf("widened result=%#v", result)
	}
}

func migrationProof(root string) *sessionledger.AuthorizationProof {
	return &sessionledger.AuthorizationProof{
		Version: 1, RootNativeID: root, ScannerSchema: "codex-v1", ScannerState: []byte(`{"version":1}`),
		Artifacts: []sessionledger.ArtifactProof{{StorageRoot: "codex-sessions", RelativePath: "root.jsonl", StableFileID: "stable", MutationToken: "mutation"}},
	}
}
