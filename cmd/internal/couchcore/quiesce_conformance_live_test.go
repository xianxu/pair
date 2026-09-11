package couchcore

// Live conformance for the ONE effect pair#230 taught the fake to model:
// quiescing a thread's session.
//
// The warm failure table and the rewritten cold-resume test both rest on what a
// quiesce leaves behind, and the fake asserts it: the session is gone, but its
// index entry is not, so PairSession answers "the binding is there, nothing is
// running under it" rather than erroring. If production and the fake ever
// disagree on that transition, every assertion built on it is testing a world
// that does not exist -- which is exactly how the ambiguous-ack test came to
// pass for the wrong reason (ARCH-MOCK).
//
// Gated on PAIR_LIVE_COUCH=1 with t.Skip and deliberately no build tag, so the
// file keeps compiling under `go test ./cmd/...` and cannot rot unnoticed.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestQuiesceLeavesTheBindingAndEndsTheSessionLive(t *testing.T) {
	liveOnly(t)

	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: ThreadTag(fmt.Sprintf("couch-%016x", os.Getpid()))}
	session := "pair-quiesce-conf-" + fmt.Sprint(os.Getpid())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })

	indexSessionForConformance(t, dataDir, address, session)
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)

	// Before: a real session, with a client attached.
	before, err := checker.PairSession(address)
	if err != nil || !before.Present || before.Name != session {
		t.Fatalf("before quiesce: binding = %+v, err = %v; want %q present", before, err, session)
	}

	if err := checker.Quiesce(address); err != nil {
		t.Fatalf("quiesce: %v", err)
	}

	// After: the session is gone, the BINDING is not. This is the exact shape
	// FakeThreadArtifactCollisionChecker.Quiesce models.
	deadline := time.Now().Add(10 * time.Second)
	for {
		after, err := checker.PairSession(address)
		if err == nil && after.Name == session && !after.Present {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("after quiesce: binding = %+v, err = %v; want %q recorded but not present", after, err, session)
		}
		time.Sleep(100 * time.Millisecond)
	}

	observed, err := checker.DetachedSessions(ctx, []DetachedCandidate{{Address: address, Agent: "claude", NativeID: "native-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Fatalf("DetachedSessions() = %+v after a quiesce, want none", observed)
	}
}

func indexSessionForConformance(t *testing.T, dataDir string, address ThreadAddress, name string) {
	t.Helper()
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: name, ScopeKey: address.RepoScope, RepoRoot: "/repo", RepoName: "repo", Tag: string(address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(paths.SessionBindings(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}
