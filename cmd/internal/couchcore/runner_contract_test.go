package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// TestBlockedRunnerCancellationConformance keeps the fake, inherited-stdio,
// and pty implementations on one cancellation trace. Context selects the
// operation lifetime; the launch owner decides whether cancellation is still
// pre-ack (Cancel) or has crossed into exact post-ack quiescence (Signal).
func TestBlockedRunnerCancellationConformance(t *testing.T) {
	type runnerCase struct {
		name   string
		runner Runner
		child  func(BlockedHandle) FakeChild
	}
	fake := NewFakeRunner()
	cases := []runnerCase{
		{name: "fake", runner: fake, child: func(h BlockedHandle) FakeChild { return fake.Child(h.ID()) }},
		{name: "exec", runner: ExecRunner{LaunchHelper: os.Args[0]}},
		{name: "pty", runner: &PtyRunner{
			LaunchHelper: os.Args[0],
			Size:         func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} },
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/canceled-before-ack", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			marker := filepath.Join(t.TempDir(), "target-ran")
			h, err := tc.runner.StartBlocked(ctx, t.TempDir(), []string{"sh", "-c", "printf exec > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
				"PAIR_TEST_RUNNER_HELPER=1",
				"PAIR_TEST_TARGET_MARKER=" + marker,
			}, 2*time.Second)
			if err != nil {
				t.Fatalf("StartBlocked: %v", err)
			}
			cancel()
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("context error = %v", ctx.Err())
			}
			if err := h.Cancel(); err != nil {
				t.Fatalf("Cancel: %v", err)
			}
			code := h.Wait()
			if terminal, ok := h.(TerminalHandle); ok {
				if _, err := terminal.Terminal().Endpoint().Snapshot(time.Now()); !errors.Is(err, os.ErrClosed) {
					t.Fatalf("canceled unaccepted terminal was not disposed: %v", err)
				}
			}
			if h.Alive() {
				pid := h.PID()
				directErr := syscall.Kill(pid, 0)
				groupErr := syscall.Kill(-pid, 0)
				t.Fatalf("canceled blocked child remained alive: pid=%d exit=%d direct=%v group=%v", pid, code, directErr, groupErr)
			}
			if tc.child != nil && tc.child(h).ExecCount != 0 {
				t.Fatalf("canceled blocked fake execed: %+v", tc.child(h))
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("canceled blocked child ran target: %v", err)
			}
		})

		t.Run(tc.name+"/canceled-after-ack", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			h, err := tc.runner.StartBlocked(ctx, t.TempDir(), []string{"sh", "-c", "sleep 30"}, []string{
				"PAIR_TEST_RUNNER_HELPER=1",
			}, 2*time.Second)
			if err != nil {
				t.Fatalf("StartBlocked: %v", err)
			}
			if err := h.Acknowledge(); err != nil {
				t.Fatalf("Acknowledge: %v", err)
			}
			cancel()
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("context error = %v", ctx.Err())
			}
			if err := h.Signal(os.Kill); err != nil {
				t.Fatalf("Signal: %v", err)
			}
			_ = h.Wait()
			if h.Alive() {
				t.Fatal("canceled acknowledged child remained alive")
			}
			if tc.child != nil && tc.child(h).ExecCount != 1 {
				t.Fatalf("acknowledged fake exec count = %+v", tc.child(h))
			}
		})
	}
}

func TestRunnerCancellationRejectsCanceledContextBeforeStartingHelper(t *testing.T) {
	t.Parallel()

	fake := NewFakeRunner()
	runners := []struct {
		name   string
		runner Runner
	}{
		{name: "fake", runner: fake},
		{name: "exec", runner: ExecRunner{LaunchHelper: os.Args[0]}},
		{name: "pty", runner: &PtyRunner{
			LaunchHelper: os.Args[0],
			Size:         func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} },
		}},
	}
	for _, tc := range runners {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			h, err := tc.runner.StartBlocked(ctx, t.TempDir(), []string{"sh", "-c", "sleep 30"}, []string{
				"PAIR_TEST_RUNNER_HELPER=1",
			}, time.Second)
			if !errors.Is(err, context.Canceled) || h != nil {
				t.Fatalf("StartBlocked handle/error = %T, %v", h, err)
			}
		})
	}
	if len(fake.Ops) != 0 {
		t.Fatalf("pre-canceled fake recorded child effects: %v", fake.Ops)
	}
}

// Probe failures must never turn unknown group membership into absence.
func TestOwnedProcessGroupProbeRetainsUnknownMembership(t *testing.T) {
	for _, present := range []bool{false, true} {
		if !ownedProcessGroupProbeAlive(123, func(int, syscall.Signal) error { return syscall.EPERM }, func(pid int) (bool, error) {
			if pid != 123 {
				t.Fatalf("membership PID = %d", pid)
			}
			return present, errors.New("process table unavailable")
		}) {
			t.Fatal("failed membership query released ownership")
		}
	}
}

func TestOwnedProcessGroupProbeSkipsUnneededMembershipQuery(t *testing.T) {
	for _, tc := range []struct {
		name string
		pid  int
		err  error
		want bool
	}{
		{"live", 123, nil, true}, {"absent", 123, syscall.ESRCH, false}, {"invalid", 0, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ownedProcessGroupProbeAlive(tc.pid, func(int, syscall.Signal) error { return tc.err }, func(int) (bool, error) {
				t.Fatal("unneeded membership query")
				return false, nil
			})
			if got != tc.want {
				t.Fatalf("alive = %v, want %v", got, tc.want)
			}
		})
	}
}
