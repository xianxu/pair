package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"
)

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
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1")
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		t.Fatal("zellij is required for live session-quiescence conformance")
	}
	session := fmt.Sprintf("pair-quiesce-live-%d", os.Getpid())
	_ = exec.Command("zellij", "delete-session", session, "--force").Run()

	cmd := exec.Command("zellij", "--session", session)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_ = exec.Command("zellij", "delete-session", session, "--force").Run()
	})
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := ptmx.Read(buf); err != nil {
				return
			}
		}
	}()

	ops := newOSSessionQuiescenceOps()
	deadline := time.Now().Add(10 * time.Second)
	for {
		present, probeErr := ops.SessionPresent(t.Context(), session)
		if probeErr == nil && present {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("throwaway zellij session never appeared: %v", probeErr)
		}
		time.Sleep(100 * time.Millisecond)
	}

	observed := &observingLiveSessionOps{inner: ops}
	rt := OSRuntime{sessionQuiescence: observed, sessionQuiesceWait: 10 * time.Second, sessionQuiescePoll: 50 * time.Millisecond}
	if err := rt.DeleteSession(session); err != nil {
		t.Fatalf("DeleteSession live conformance: %v", err)
	}
	if !observed.serverObserved || !observed.deleteAttempted || !observed.killAttempted {
		t.Fatalf("incomplete live interface coverage: %+v", observed)
	}
}
