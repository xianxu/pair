package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestCoordinatedCouchRegistersActualRootRuntime(t *testing.T) {
	t.Parallel()

	namespace := testCouchNamespace(t)
	root := t.TempDir()
	now := time.Now()
	artifacts := constructorLifecycleArtifacts{FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker(), lifecycle: &fakeControllerLifecycle{model: pairlifecycletest.New(now)}, dataDir: root}
	_, err := New(namespace, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(namespace.Dir()), FixedClock{T: now}, NewFixedIDGen("id"), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	runtimes, err := coordinator.ReadRuntimes()
	if err != nil || len(runtimes) != 1 || runtimes[0].Process.PID != os.Getpid() || runtimes[0].Role != storagegc.CouchRuntimeRole || runtimes[0].DiagnosticProtocol != storagegc.DiagnosticProtocolVersion {
		t.Fatalf("Couch not registered: %+v %v", runtimes, err)
	}
	firstID := runtimes[0].ID
	if err := coordinator.RegisterRuntime(context.Background(), runtimes[0].Process, storagegc.CouchRuntimeRole); err != nil {
		t.Fatal(err)
	}
	repeated, err := coordinator.ReadRuntimes()
	if err != nil || len(repeated) != 1 || repeated[0].ID != firstID || repeated[0].DiagnosticProtocol != storagegc.DiagnosticProtocolVersion {
		t.Fatalf("generic re-registration lost capability or ID: %+v %v", repeated, err)
	}
	entries, err := os.ReadDir(filepath.Join(coordinator.Root, ".retention", "owners"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("Couch root registration initialized owner use: %v %v", entries, err)
	}
}
