package launcher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

func TestCreatorPaneReadinessReleasesBeforeAgentReady(t *testing.T) {
	dir := t.TempDir()
	namespace := filepath.Join(dir, "native")
	if err := os.MkdirAll(filepath.Join(namespace, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("ZELLIJ_SOCKET_DIR", namespace)
	a := &terminalOSAttempt{rt: OSRuntime{DataDir: filepath.Join(dir, "data")}, name: "private-session", socket: filepath.Join(namespace, "contract_version_1", "private-session"), socketRoot: namespace, scope: "0123456789abcdef", tag: "1-test-1", agent: "claude", nonce: "this-launch", create: true}
	lease, err := a.Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := a.creator.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	owner, err := artifactpath.NewStorageOwner(a.rt.globalDataDir(), a.scope, a.tag)
	if err != nil {
		t.Fatal(err)
	}
	path, err := panebirth.Evidence(owner.Directory(), a.tag, a.agent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	writePane := func(identity panebirth.LaunchIdentity) {
		t.Helper()
		raw, err := panebirth.EncodeRecord(panebirth.Record{PaneID: "0", Identity: identity})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	source := zellijquery.Source{Env: os.Environ(), Run: func(cmd *exec.Cmd) error { calls++; _, err := cmd.Stdout.Write([]byte("private reply")); return err }}
	assertBlocked := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
		defer cancel()
		_, err := source.ReadContext(ctx, "list-sessions", "--no-formatting")
		if !errors.Is(err, context.DeadlineExceeded) || calls != 0 {
			t.Fatalf("initializing native read: calls=%d err=%v", calls, err)
		}
	}
	assertBlocked()
	identity := a.creatorIdentity()
	stale := identity
	stale.Nonce = "previous-launch"
	writePane(stale)
	if err := a.observeCreateAdmission(t.Context()); err == nil {
		t.Fatal("stale pane admitted launch")
	}
	assertBlocked()
	writePane(identity)
	if err := a.observeCreateAdmission(t.Context()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("must still await absent AgentReady: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := source.ReadContext(ctx, "list-sessions", "--no-formatting"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("native reads after exact pane: %d", calls)
	}
	// Reobserving absent agent readiness must not attempt to republish through
	// the released guard or turn first-pane evidence into agent readiness.
	if err := a.observeCreateAdmission(t.Context()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repeated agent wait: %v", err)
	}
}
