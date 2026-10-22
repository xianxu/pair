package agentfixture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestCLILeaderExitCannotAcknowledgeLiveDescendant(t *testing.T) {
	for _, tc := range []struct{ name, redirect string }{
		{"inherited-pipes", ""},
		{"closed-pipes", " >/dev/null 2>&1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "descendant-pid")
			result := runCLI(t.Context(), "/bin/sh", Command{ID: "leader-exited", Timeout: 200 * time.Millisecond, Args: []string{"-c", `sleep 30` + tc.redirect + ` & printf '%s' "$!" > "$1"; exit 0`, "fixture", marker}})
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			pid, err := strconv.Atoi(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			generation := procutil.StrictIdentity(strconv.Itoa(pid))
			t.Cleanup(func() {
				if generation != "" && procutil.StrictIdentity(strconv.Itoa(pid)) == generation {
					if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
						t.Error(err)
					}
				}
			})
			if err := syscall.Kill(pid, 0); err == nil && !strings.Contains(result.Error, "cleanup uncertain") {
				t.Fatalf("request acknowledged while owned descendant is alive: pid=%d result=%+v", pid, result)
			}
		})
	}
}

func TestCLICompletionPaths(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		exit         int
	}{
		{"normal-exit", "exit 0", 0},
		{"command-error", "exit 7", 7},
		{"signal-exit", `kill -TERM "$$"`, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := runCLI(t.Context(), "/bin/sh", Command{ID: "completion", Args: []string{"-c", tc.script}})
			if r.CleanupUncertain || r.ExitCode != tc.exit {
				t.Fatalf("completion=%+v", r)
			}
			if (tc.exit == 0) != (r.Error == "") {
				t.Fatalf("error outcome=%+v", r)
			}
		})
	}
}

func TestCLILiveLeaderCancellationJoinsBeforeCompletion(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "leader")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan Result, 1)
	go func() {
		done <- runCLI(ctx, "/bin/sh", Command{ID: "cancel", Args: []string{"-c", `printf '%s' "$$" > "$1"; exec sleep 30`, "fixture", marker}})
	}()
	waitFixture(t, func() bool { _, e := os.Stat(marker); return e == nil })
	raw, e := os.ReadFile(marker)
	if e != nil {
		t.Fatal(e)
	}
	pid, e := strconv.Atoi(string(raw))
	if e != nil {
		t.Fatal(e)
	}
	cancel()
	select {
	case r := <-done:
		if r.CleanupUncertain || r.Error == "" {
			t.Fatalf("cancellation=%+v", r)
		}
		if e := syscall.Kill(pid, 0); e != syscall.ESRCH {
			t.Fatalf("child still alive: %v", e)
		}
	case <-t.Context().Done():
		t.Fatal("cancellation not joined")
	}
}

func TestCLIGroupObservationClassification(t *testing.T) {
	for _, tc := range []struct {
		name       string
		signal     error
		members    bool
		inspection error
		uncertain  bool
	}{
		{name: "absent", signal: syscall.ESRCH},
		{name: "live", uncertain: true},
		{name: "darwin-empty", signal: syscall.EPERM},
		{name: "permission-live", signal: syscall.EPERM, members: true, uncertain: true},
		{name: "inspection-unknown", signal: syscall.EPERM, inspection: errors.New("unreadable"), uncertain: true},
		{name: "signal-unknown", signal: syscall.EINVAL, uncertain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cliGroupError(tc.signal, tc.members, tc.inspection)
			if errors.Is(err, ErrCleanupUncertain) != tc.uncertain {
				t.Fatalf("observation=%v", err)
			}
		})
	}
}
