package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func waitUntilTrue(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// The capability check is only meaningful if a runner can FAIL it. ExecRunner
// hands the child couch's own stdio and has no terminal to offer, which is what
// keeps the non-console runner a real injected seam rather than dead plumbing.
func TestExecRunnerHandleIsNotATerminalHandle(t *testing.T) {
	h, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 0"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = h.Signal(os.Kill) }()

	if _, ok := h.(TerminalHandle); ok {
		t.Fatal("ExecRunner's handle satisfies TerminalHandle; the capability check would be vacuous")
	}
}

func TestPtyRunnerHandleIsATerminalHandle(t *testing.T) {
	r := &PtyRunner{Size: func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} }}
	h, err := r.Start(t.TempDir(), []string{"sh", "-c", "sleep 5"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = h.Signal(os.Kill) }()

	th, ok := h.(TerminalHandle)
	if !ok {
		t.Fatal("PtyRunner's handle does not satisfy TerminalHandle")
	}
	if th.Terminal() == nil {
		t.Fatal("Terminal() == nil")
	}
}

// The size supplier must be honoured AT SPAWN, not applied afterwards: a
// full-screen agent harness that draws its first frame at 80x24 and reflows is
// a whole redraw the operator sees on every start.
func TestPtyRunnerHonoursItsSizeSupplierAtSpawn(t *testing.T) {
	r := &PtyRunner{Size: func() ptychild.Size { return ptychild.Size{Rows: 31, Cols: 99} }}
	h, err := r.Start(t.TempDir(), []string{"sh", "-c", "stty size; sleep 5"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = h.Signal(os.Kill) }()

	child := h.(TerminalHandle).Terminal()
	waitUntilTrue(t, "the child to report the spawn size", func() bool {
		return strings.Contains(string(child.Snapshot()), "31 99")
	})
}

// The sink is installed inside Start, not bolted on afterwards: a child that
// writes before the console attaches would otherwise lose those chunks from the
// live path.
func TestPtyRunnerInstallsTheSinkBeforeTheChildCanWrite(t *testing.T) {
	var mu sync.Mutex
	var got []string
	r := &PtyRunner{
		Size: func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} },
		Sink: func(ctx context.Context, id string, batch ptychild.OutputBatch) error {
			// The sink runs on the child's pump goroutine.
			mu.Lock()
			defer mu.Unlock()
			got = append(got, id)
			return nil
		},
	}
	h, err := r.Start(t.TempDir(), []string{"sh", "-c", "printf hello; sleep 5"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = h.Signal(os.Kill) }()

	waitUntilTrue(t, "the sink to see a chunk", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) > 0
	})
	mu.Lock()
	first := got[0]
	mu.Unlock()
	if first != h.ID() {
		t.Fatalf("sink received id %q, want the handle's id %q", first, h.ID())
	}
}

func TestPtyRunnerRejectsEmptyArgv(t *testing.T) {
	t.Parallel()

	r := &PtyRunner{Size: func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} }}
	if _, err := r.Start(t.TempDir(), nil, nil); err == nil {
		t.Fatal("Start with no argv returned nil error")
	}
}

func TestPtyRunnerBlockedStartPreservesTerminalAndWaitsForAcknowledgement(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), "target-ran")
	r := &PtyRunner{
		LaunchHelper: os.Args[0],
		Size:         func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} },
	}
	h, err := r.StartBlocked(context.Background(), t.TempDir(), []string{"sh", "-c", "printf exec > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_TARGET_MARKER=" + marker,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	if _, ok := h.(TerminalHandle); !ok {
		t.Fatal("blocked pty handle lost TerminalHandle capability")
	}
	time.Sleep(75 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("target ran before acknowledgement: %v", err)
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("Wait = %d", code)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "exec" {
		t.Fatalf("target marker = %q, %v", raw, err)
	}
}

// Live conformance for FakeRunner.SetChildExited (pair#489): on both real
// runners, Exited closes when the direct child exits while a member it left
// in its process group lives on, and Alive still answers for that group. The
// registration wait keys on the first; cleanup keys on the second.
func TestRealRunnersExitedPrecedesTheGroup(t *testing.T) {
	// The survivor ignores SIGHUP, which a pty session sends when its leader exits.
	// The FIFO handshake proves the trap is installed before that exit.
	argv := []string{"sh", "-c", `mkfifo ready || exit 1; (trap "" HUP; printf "ready\n" > ready; exec sleep 30) & IFS= read -r token < ready; [ "$token" = ready ] || exit 1; exit 0`}
	for name, start := range map[string]func() (Handle, error){
		"exec": func() (Handle, error) { return ExecRunner{}.Start(t.TempDir(), argv, nil) },
		"pty": func() (Handle, error) {
			r := &PtyRunner{Size: func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} }}
			return r.Start(t.TempDir(), argv, nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			h, err := start()
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			defer func() { _ = h.Signal(os.Kill) }()
			select {
			case <-h.Exited():
			case <-time.After(20 * time.Second):
				t.Fatal("Exited never closed after the direct child exited")
			}
			if !h.Alive() {
				t.Fatal("Alive = false while the backgrounded group member still runs")
			}
		})
	}
}
