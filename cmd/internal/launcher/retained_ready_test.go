package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreparingReadinessRetainsPreviousAgentEvidence(t *testing.T) {
	root := t.TempDir()
	rt := NewOSRuntime(root, "/pair")
	t.Setenv("PAIR_LAUNCH_NONCE", "")
	old := readiness.ReadyRecord{Tag: "work", Agent: "codex", Session: "same", Nonce: "old-nonce", PID: os.Getpid()}
	raw, _ := readiness.Encode(old)
	path := AgentReadyPath(root, "work", "codex")
	os.WriteFile(path, []byte(raw), 0600)
	expect, e := prepareLaunchReadinessWithNonce(rt, "work", "codex", "same", "new-nonce")
	if e != nil {
		t.Fatal(e)
	}
	got, e := os.ReadFile(path)
	if e != nil || string(got) != raw {
		t.Fatalf("preparation erased old liveness evidence: %q %v", got, e)
	}
	if MatchReadyRecord(expect, old, func(int) bool { return true }) == nil {
		t.Fatal("stale record satisfied new readiness")
	}
}

type retainedReadyWorld struct {
	*handoffWorld
	attempt *terminalOSAttempt
}

func (w *retainedReadyWorld) Acquire(ctx context.Context) (io.Closer, error) {
	w.event("acquire")
	lease, err := w.attempt.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return retainedReadyLease{lease, w.handoffWorld}, nil
}

type retainedReadyLease struct {
	io.Closer
	world *handoffWorld
}

func (l retainedReadyLease) Close() error {
	l.world.event("release")
	return l.Closer.Close()
}

func prepareRetainedReadyGuard(t *testing.T, a *terminalOSAttempt) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	a.socketRoot = filepath.Join(filepath.Dir(a.socket), "native")
	t.Setenv("ZELLIJ_SOCKET_DIR", a.socketRoot)
	a.socket = filepath.Join(a.socketRoot, "contract_version_1", a.name)
	a.nonce = "current-launch"
}

func (w *retainedReadyWorld) Check(ctx context.Context) error {
	w.event("check")
	return w.attempt.Check(ctx)
}
func TestColdLaunchChecksCrossAgentSurvivorUnderLease(t *testing.T) {
	root := t.TempDir()
	owner, _ := artifactpath.NewStorageOwner(root, "scope", "work")
	os.MkdirAll(owner.Directory(), 0700)
	name := fmt.Sprintf("pair528-retained-%d", os.Getpid())
	ready, _ := readiness.Encode(readiness.ReadyRecord{Tag: "work", Agent: "claude", Session: name, Nonce: "old", PID: os.Getpid()})
	os.WriteFile(AgentReadyPath(owner.Directory(), "work", "claude"), []byte(ready), 0600)
	world := newHandoffWorld()
	world.admission <- errors.New("unexpected start")
	attempt := &terminalOSAttempt{rt: *NewOSRuntime(root, "/pair"), name: name, scope: "scope", tag: "work", agent: "codex", socket: filepath.Join(root, "absent-socket"), create: true}
	prepareRetainedReadyGuard(t, attempt)
	_, e := CoordinateTerminalHandoff(context.Background(), &retainedReadyWorld{world, attempt}, time.Second)
	if e == nil || !strings.Contains(e.Error(), "live agent") {
		t.Fatalf("did not refuse survivor: %v", e)
	}
	if got := strings.Join(world.events, ","); got != "acquire,check,release" {
		t.Fatalf("guard outside lease or client started: %s", got)
	}
}

func TestColdLaunchUnknownReadinessRefusesUnderLease(t *testing.T) {
	for _, kind := range []string{"malformed", "oversized", "directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			owner, _ := artifactpath.NewStorageOwner(root, "scope", "work")
			if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
				t.Fatal(err)
			}
			path := AgentReadyPath(owner.Directory(), "work", "claude")
			switch kind {
			case "malformed":
				os.WriteFile(path, []byte("{"), 0600)
			case "oversized":
				os.WriteFile(path, []byte(strings.Repeat(" ", 8193)), 0600)
			case "directory":
				os.Mkdir(path, 0700)
			}
			world := newHandoffWorld()
			world.admission <- errors.New("unexpected start")
			attempt := &terminalOSAttempt{rt: *NewOSRuntime(root, "/pair"), name: fmt.Sprintf("pair528-unknown-%d", os.Getpid()), scope: "scope", tag: "work", agent: "codex", socket: filepath.Join(root, "absent"), create: true}
			prepareRetainedReadyGuard(t, attempt)
			_, e := CoordinateTerminalHandoff(context.Background(), &retainedReadyWorld{world, attempt}, time.Second)
			if e == nil || !strings.Contains(e.Error(), "readiness") {
				t.Fatalf("accepted unknown readiness: %v", e)
			}
			if got := strings.Join(world.events, ","); got != "acquire,check,release" {
				t.Fatal(got)
			}
		})
	}
}

func TestColdLaunchAcceptsProvenDeadRetainedAgent(t *testing.T) {
	cmd := exec.Command("true")
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	owner, _ := artifactpath.NewStorageOwner(root, "scope", "work")
	if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	ready, _ := readiness.Encode(readiness.ReadyRecord{Tag: "work", Agent: "claude", Session: "old", Nonce: "old", PID: cmd.Process.Pid})
	os.WriteFile(AgentReadyPath(owner.Directory(), "work", "claude"), []byte(ready), 0600)
	attempt := &terminalOSAttempt{rt: *NewOSRuntime(root, "/pair"), name: fmt.Sprintf("pair528-dead-%d", os.Getpid()), scope: "scope", tag: "work", agent: "codex", socket: filepath.Join(root, "absent"), create: true}
	prepareRetainedReadyGuard(t, attempt)
	lease, err := attempt.Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if e := attempt.Check(context.Background()); e != nil {
		t.Fatal("dead readiness blocked recovery", e)
	}
}
