package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestMain(m *testing.M) {
	if os.Getenv("PAIR_ZELLIJ_KILL_SENTINEL") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
	os.Exit(m.Run())
}

type observingLiveSessionOps struct {
	inner           sessionQuiescenceOps
	serverObserved  bool
	deleteAttempted bool
	killAttempted   bool
}

func (o *observingLiveSessionOps) SessionPresent(ctx context.Context, session string) (bool, error) {
	return o.inner.SessionPresent(ctx, session)
}

func (o *observingLiveSessionOps) SessionServers(ctx context.Context, session string) ([]sessionServerIdentity, error) {
	servers, err := o.inner.SessionServers(ctx, session)
	if len(servers) > 0 {
		o.serverObserved = true
	}
	return servers, err
}

func (o *observingLiveSessionOps) DeleteSessionRecord(ctx context.Context, session string) error {
	o.deleteAttempted = true
	return o.inner.DeleteSessionRecord(ctx, session)
}

func (o *observingLiveSessionOps) KillServer(server sessionServerIdentity) error {
	o.killAttempted = true
	return o.inner.KillServer(server)
}

// TestSessionQuiescenceLive is the external-interface counterpart to the
// stateful re-registration fake. `make test-live` runs it locally; the weekly
// couch-zellij-conformance workflow runs this focused target on macOS.
func TestSessionQuiescenceLive(t *testing.T) {
	session := startControlledZellijSession(t)

	// Zellij's own delete-session commonly terminates its server before the
	// explicit escalation step runs. Add an exact-argv sentinel that is not
	// owned by that session, so live conformance must dispatch the underlying
	// OS kill operation rather than merely enter KillServer after the real
	// server has already disappeared.
	sentinelDir := t.TempDir()
	sentinelBinary := filepath.Join(sentinelDir, "zellij")
	sentinel := &exec.Cmd{
		Path: os.Args[0],
		Args: []string{sentinelBinary, "--server", filepath.Join(sentinelDir, session)},
		Env:  append(os.Environ(), "PAIR_ZELLIJ_KILL_SENTINEL=1"),
	}
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sentinel.Process.Kill()
		_, _ = sentinel.Process.Wait()
	})

	ops := newOSSessionQuiescenceOps()
	sentinelObserved := false
	deadline := time.Now().Add(10 * time.Second)
	for !sentinelObserved {
		servers, probeErr := ops.SessionServers(t.Context(), session)
		if probeErr == nil {
			for _, server := range servers {
				if server.PID == sentinel.Process.Pid {
					sentinelObserved = true
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("exact-argv kill sentinel was not observed: %v", probeErr)
		}
		time.Sleep(25 * time.Millisecond)
	}

	realKill := ops.killProcess
	killedSentinel := false
	ops.killProcess = func(pid int) error {
		if pid == sentinel.Process.Pid {
			killedSentinel = true
		}
		return realKill(pid)
	}

	observed := &observingLiveSessionOps{inner: ops}
	rt := OSRuntime{sessionQuiescence: observed, sessionQuiesceWait: 10 * time.Second, sessionQuiescePoll: 50 * time.Millisecond}
	if err := rt.DeleteSession(session); err != nil {
		t.Fatalf("DeleteSession live conformance: %v", err)
	}
	if !observed.serverObserved || !observed.deleteAttempted || !observed.killAttempted {
		t.Fatalf("incomplete live interface coverage: %+v", observed)
	}
	if !killedSentinel {
		t.Fatal("live quiescence did not dispatch the underlying OS kill operation")
	}
}

func startControlledZellijSession(t *testing.T) string {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1")
	}
	session := fmt.Sprintf("pair-quiesce-live-%d", os.Getpid())
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })

	return session
}
