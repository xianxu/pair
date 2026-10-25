package launcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// This negative control records the accepted interactive coverage limit: the
// production attach path can begin while another session's creator holds the
// namespace lock. Its interactive lifetime must also leave later creates free.
func TestInteractiveAttachOverlapsCreatorWithoutHoldingNamespaceForLifetime(t *testing.T) {
	root := t.TempDir()
	home, bin, namespace := filepath.Join(root, "home"), filepath.Join(root, "bin"), filepath.Join(root, "native")
	for _, dir := range []string{home, bin, namespace} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	started := filepath.Join(root, "interactive-started")
	fake := `#!/bin/sh
printf '%s\n' "$*" > "$PAIR_TEST_INTERACTIVE_STARTED"
exec /bin/sleep 60
`
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte(fake), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin)
	t.Setenv("ZELLIJ_SOCKET_DIR", namespace)
	t.Setenv("PAIR_TEST_INTERACTIVE_STARTED", started)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	canonical, err := zellijquery.ReadinessNamespace(os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	identity := zellijquery.ReadyIdentity{Namespace: canonical, Scope: "repo_1", Tag: "creator-1", Agent: "claude", Session: "initializing-session", Nonce: "creator-nonce"}
	creator, err := zellijquery.AcquireCreator(ctx, os.Environ(), identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	attempt, err := newTerminalOSAttempt(OSRuntime{}, "interactive-neighbor", "private-config", "", false)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := attempt.Acquire(ctx)
	if err != nil {
		t.Fatalf("interactive acquire waited for namespace creator: %v", err)
	}
	defer admission.Close()
	clientCtx, cancelClient := context.WithCancel(ctx)
	client, err := attempt.Start(clientCtx)
	if err != nil {
		cancelClient()
		t.Fatal(err)
	}
	// The script execs sleep, so this cancellation and Wait join the only owned
	// fake client through the production killOnCancel seam, with no child shell.
	defer func() {
		cancelClient()
		client.Wait()
		if client.(terminalOSClient).cmd.ProcessState == nil {
			t.Error("interactive fake was not reaped")
		}
	}()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		raw, err := os.ReadFile(started)
		if err == nil && len(raw) > 0 {
			if !strings.Contains(string(raw), "attach interactive-neighbor") {
				t.Fatalf("fake did not receive interactive argv: %q", raw)
			}
			break
		}
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("interactive Start did not run under held creator exclusion: %v", ctx.Err())
		case <-ticker.C:
		}
	}
	if err := syscall.Kill(client.(terminalOSClient).cmd.Process.Pid, 0); err != nil {
		t.Fatalf("interactive fake exited before overlap proof: %v", err)
	}
	if err := creator.Close(); err != nil {
		t.Fatal(err)
	}
	next := identity
	next.Session = "next-created-session"
	next.Nonce = "next-nonce"
	nextCtx, cancelNext := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancelNext()
	nextCreator, err := zellijquery.AcquireCreator(nextCtx, os.Environ(), next)
	if err != nil {
		t.Fatalf("interactive client held namespace for its lifetime: %v", err)
	}
	defer nextCreator.Close()
	if err := syscall.Kill(client.(terminalOSClient).cmd.Process.Pid, 0); err != nil {
		t.Fatalf("interactive fake was not still running during next creator admission: %v", err)
	}
}
