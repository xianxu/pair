package sessioninventory_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestIncrementalPrelaunchOperationBudget(t *testing.T) {
	t.Parallel()
	const fileCount = 1573
	const corpusBytes = int64(358 << 20)
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(root)
	for i := 0; i < fileCount; i++ {
		id := fmt.Sprintf("%08x-1111-7111-8111-%012x", i+1, i+1)
		size := corpusBytes / fileCount
		if i == fileCount-1 {
			size += corpusBytes % fileCount
		}
		artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl", Kind: sessioninventory.ArtifactTranscript}
		runtime.PutMetadataFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: sessioninventory.StableFileID("stable-" + id), GenerationToken: sessioninventory.GenerationToken("gen-" + id), MutationToken: sessioninventory.MutationToken("mutation-" + id), Size: size})
	}

	observed, diagnostics := sessioninventory.ObserveAgentMetadata(runtime, sessioninventory.AgentCodex)
	if len(diagnostics) != 0 || len(observed) != fileCount {
		t.Fatalf("observations=%d diagnostics=%d", len(observed), len(diagnostics))
	}
	cold := sessioninventory.ReconcileCatalog(sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}, observed)
	if len(cold.Work) != fileCount || len(cold.Reused) != 0 {
		t.Fatalf("cold work=%d reused=%d", len(cold.Work), len(cold.Reused))
	}
	warmCatalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion, Generation: 1}
	for _, observation := range observed {
		warmCatalog.Entries = append(warmCatalog.Entries, sessioninventory.CatalogEntry{
			Agent: observation.Agent, Artifact: observation.Entry.Artifact,
			Fingerprint:   sessioninventory.ArtifactFingerprint{StableFileID: observation.Entry.StableFileID, GenerationToken: observation.Entry.GenerationToken, MutationToken: observation.Entry.MutationToken, Size: observation.Entry.Size},
			Authorization: sessioninventory.AuthorizationCandidate, ScannerSchema: observation.ScannerSchema, ProviderContract: observation.ProviderContract,
		})
	}
	warm := sessioninventory.ReconcileCatalog(warmCatalog, observed)
	if len(warm.Work) != 0 || len(warm.Reused) != fileCount {
		t.Fatalf("warm work=%d reused=%d", len(warm.Work), len(warm.Reused))
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationListFiles, root.Name); got != 1 {
		t.Fatalf("metadata listings=%d, want 1", got)
	}
	for _, operation := range []sessioninventorytest.Operation{sessioninventorytest.OperationReadFile, sessioninventorytest.OperationReadAt, sessioninventorytest.OperationSQLite, sessioninventorytest.OperationOpenFiles} {
		if got := runtime.OperationCount(operation, ""); got != 0 {
			t.Fatalf("%s operations=%d, want 0", operation, got)
		}
	}
}

func TestLiveIncrementalInventoryPrelaunch(t *testing.T) {
	if os.Getenv("PAIR_LIVE_SESSION_INVENTORY") != "1" {
		t.Skip("set PAIR_LIVE_SESSION_INVENTORY=1 for the installed metadata budget")
	}
	pairData := os.Getenv("PAIR_DATA_DIR")
	if pairData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		pairData = filepath.Join(home, ".local", "share", "pair")
	}
	runtime, err := sessioninventory.DefaultOSRuntime(pairData)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	observations, diagnostics := sessioninventory.ObserveAgentMetadata(runtime, sessioninventory.AgentClaude)
	elapsed := time.Since(started)
	if len(observations) == 0 {
		t.Fatal("no recognized installed Claude metadata")
	}
	if elapsed > time.Second {
		t.Fatalf("metadata prelaunch exceeded one second: files=%d diagnostics=%d elapsed=%s", len(observations), len(diagnostics), elapsed)
	}
	t.Logf("metadata prelaunch: files=%d diagnostics=%d elapsed=%s", len(observations), len(diagnostics), elapsed)
}
