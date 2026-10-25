package zellijquery

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// initializingNative models the relevant native contract: even a named read
// visits every server, and removing that probe client before pane execution is
// invalid. Its observations are independent of the guard's own state.
type initializingNative struct {
	mu               sync.Mutex
	initializing     bool
	scans, premature int
	reply            []byte
}

func (n *initializingNative) run(cmd *exec.Cmd) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.scans++
	if n.initializing {
		n.premature++
		return errors.New("probe removed before first pane")
	}
	_, err := cmd.Stdout.Write(n.reply)
	return err
}

func privateReadinessEnv(t *testing.T) ([]string, ReadyIdentity) {
	t.Helper()
	dir := t.TempDir()
	home, namespace := filepath.Join(dir, "home"), filepath.Join(dir, "native")
	for _, path := range []string{home, filepath.Join(namespace, "contract_version_1")} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return []string{"HOME=" + home, "ZELLIJ_SOCKET_DIR=" + namespace}, ReadyIdentity{
		Namespace: namespace, Scope: "0123456789abcdef", Tag: "1-test-1",
		Agent: "claude", Session: "private-session", Nonce: "start-private",
	}
}

func TestReadContextCannotProbeDuringCreatorInitialization(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	if err := creator.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	native := &initializingNative{initializing: true, reply: []byte(zellijtest.Reply(t, "clients", []zellijtest.Client{}).Stdout)}
	source := Source{Env: env, Run: native.run}
	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Millisecond)
	defer cancel()
	_, err = source.ReadContext(ctx, "--session", "already-ready-neighbor", "action", "list-clients")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked read: %v", err)
	}
	if native.scans != 0 || native.premature != 0 {
		t.Fatalf("native scans=%d premature=%d", native.scans, native.premature)
	}
	// The native state changes because the real pane ran, not because the guard
	// says so. Marker publication is a separate, exact-identity event.
	native.initializing = false
	if err := creator.PublishReady(t.Context(), identity); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ReadContext(t.Context(), "action", "list-clients"); err != nil {
		t.Fatal(err)
	}
	if native.scans != 1 || native.premature != 0 {
		t.Fatalf("native scans=%d premature=%d", native.scans, native.premature)
	}
}

func TestCreatorRejectsStaleReadyIdentity(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	if err := creator.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	stale := identity
	stale.Nonce = "start-previous"
	if err := creator.PublishReady(t.Context(), stale); err == nil {
		t.Fatal("stale pane released creator")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	calls := 0
	_, err = (Source{Env: env, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(ctx, "action", "list-clients")
	if !errors.Is(err, context.DeadlineExceeded) || calls != 0 {
		t.Fatalf("stale publication admitted native scan: calls=%d err=%v", calls, err)
	}
}

func TestCreatorPreflightPermitIsScopedToItsCallbackAndNamespace(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	native := &initializingNative{reply: []byte(zellijtest.Reply(t, "clients", []zellijtest.Client{}).Stdout)}
	source := Source{Env: env, Run: native.run}
	var escaped context.Context
	if err := creator.Preflight(t.Context(), func(ctx context.Context) error {
		escaped = ctx
		_, err := source.ReadContext(ctx, "action", "list-clients")
		if err != nil {
			return err
		}
		foreign, _ := privateReadinessEnv(t)
		_, err = (Source{Env: foreign, Run: native.run}).ReadContext(ctx, "action", "list-clients")
		if !errors.Is(err, ErrReadinessUnknown) {
			return errors.New("creator permit crossed namespace")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if native.scans != 1 {
		t.Fatalf("preflight scans=%d", native.scans)
	}
	if _, err := source.ReadContext(escaped, "action", "list-clients"); !errors.Is(err, ErrReadinessUnknown) {
		t.Fatalf("escaped preflight permit: %v", err)
	}
	if native.scans != 1 {
		t.Fatal("expired permit admitted another scan")
	}
}
