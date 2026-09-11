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

// controlledZellijFixture re-starts the session under a handle the caller can
// manipulate. startControlledZellijSession keeps its fixture private, and the
// detach case needs to kill the client specifically.
func controlledZellijFixture(t *testing.T, session string) *pairlifecycletest.ControlledZellij {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })
	return fixture
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

// TestSessionDetachLive is the conformance counterpart for Couch's detach
// (`pair#170` M2), and the inverse assertion to TestSessionQuiescenceLive: that
// one proves a session can be made to GO, this one proves it STAYS when only
// its client dies.
//
// Everything detach rests on is this one external behaviour -- kill the client,
// keep the agent -- and it is modelled by a fake everywhere else in the suite.
// Without this, the fake models a state nothing confirms.
func TestSessionDetachLive(t *testing.T) {
	fixture := controlledZellijFixture(t, fmt.Sprintf("pair-detach-live-%d", os.Getpid()))
	session := fixture.Session

	find := func(form string, sessions []Session, err error) (SessionState, bool) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s snapshot of zellij sessions: %v", form, err)
		}
		for _, candidate := range sessions {
			if candidate.Name == session {
				return candidate.State, true
			}
		}
		return "", false
	}
	state := func() (SessionState, bool) {
		t.Helper()
		sessions, err := (ZellijSource{}).Snapshot()
		return find("full", sessions, err)
	}
	// The narrowed forms (pair#228) against the real binary, at each stable
	// stage: the named form classifies exactly as the full form does and returns
	// only the session it was asked about; the liveness form reports the session
	// live without classifying it. Elsewhere they are checked against StubZellij.
	narrowedFormsAgree := func(want SessionState) {
		t.Helper()
		named, err := (ZellijSource{}).SnapshotSessionsContext(t.Context(), []string{session})
		if got, ok := find("named", named, err); !ok || got != want || len(named) != 1 {
			t.Fatalf("named snapshot = %+v, want only %s, %s", named, session, want)
		}
		live, err := (ZellijSource{}).LivenessContext(t.Context())
		if got, ok := find("liveness", live, err); !ok || got != SessionLive {
			t.Fatalf("liveness snapshot: state = %q present = %v, want live", got, ok)
		}
	}

	if got, ok := state(); !ok || got != SessionAttached {
		t.Fatalf("before detach: state = %q present = %v, want attached", got, ok)
	}
	narrowedFormsAgree(SessionAttached)

	if err := fixture.KillClient(); err != nil {
		t.Fatalf("kill the zellij client: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		got, ok := state()
		if ok && got == SessionDetached {
			narrowedFormsAgree(SessionDetached)
			return
		}
		if !ok {
			t.Fatal("the session vanished with its client -- detach would have nothing to reattach to")
		}
		if time.Now().After(deadline) {
			t.Fatalf("session state = %q after the client died, want detached", got)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
