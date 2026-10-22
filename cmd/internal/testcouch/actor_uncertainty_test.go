package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

func waitMailboxCommand(t *testing.T, dir string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	var id string
	if err := waitUntil(ctx, func() (bool, error) {
		entries, err := os.ReadDir(filepath.Join(dir, "commands"))
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".json" {
				id = entry.Name()[:len(entry.Name())-5]
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

// A held private mailbox is a stateful fixture: publication and cancellation use
// the production filesystem adapter, while no consumer acknowledges execution.
func heldActorMailbox(t *testing.T) (*Fleet, string) {
	t.Helper()
	root := t.TempDir()
	worktree := filepath.Join(root, "repo")
	control := filepath.Join(root, "control")
	dir := agentfixture.ActorDirectory(control, worktree)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(agentfixture.Ready{PID: os.Getpid(), RunID: "held"})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "ready.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return &Fleet{Roots: RootsPlan{Root: root, Mutable: map[string]string{"control": control}}, Workspaces: Workspaces{Primary: worktree}, Options: Options{RuntimeBudget: time.Minute}, ctx: t.Context()}, dir
}

func assertUncertainAdmission(t *testing.T, f *Fleet, err error) {
	t.Helper()
	if !errors.Is(err, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("missing typed uncertainty: %v", err)
	}
	if f.model.Error == "" {
		t.Fatal("failure did not latch fleet uncertainty")
	}
	if _, err = f.ActorCLI(t.Context(), 0, "-c", "exit 0"); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("new CLI admitted: %v", err)
	}
	if err = f.Input([]byte("x")); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("new input admitted: %v", err)
	}
	if err = f.Resize(24, 80); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("new resize admitted: %v", err)
	}
}

func TestActorCLICancelPublicationFailureFencesLiveChild(t *testing.T) {
	f, dir := startActorCLIFixture(t)
	caller, cancel := context.WithCancel(t.Context())
	defer cancel()
	marker := filepath.Join(f.Roots.Root, "live-child")
	done := make(chan error, 1)
	go func() {
		_, err := f.ActorCLI(caller, 0, "-c", `printf '%s' "$$" > "$1"; exec sleep 30`, "fixture", marker)
		done <- err
	}()
	id := waitMailboxCommand(t, dir)
	ctx, end := context.WithTimeout(t.Context(), 4*time.Second)
	defer end()
	var pid int
	if err := waitUntil(ctx, func() (bool, error) {
		raw, err := os.ReadFile(marker)
		pid, _ = strconv.Atoi(string(raw))
		return err == nil && pid > 0, nil
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "cancellations")
	if err := os.Chmod(path, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0700); err != nil {
			t.Error(err)
		}
	})
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost caller cancellation: %v", err)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("failure fixture did not retain live child: %v", err)
	}
	assertUncertainAdmission(t, f, err)
	if _, err := os.Stat(filepath.Join(dir, "commands", id+".json")); err != nil {
		t.Fatalf("command evidence lost: %v", err)
	}
}

func TestActorCLIUnacknowledgedExitsFenceFleet(t *testing.T) {
	for _, kind := range []string{"ack-read", "ack-correlation", "ack-timeout", "retirement"} {
		t.Run(kind, func(t *testing.T) {
			f, dir := heldActorMailbox(t)
			caller, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := f.ActorCLI(caller, 0, "--actors"); done <- err }()
			id := waitMailboxCommand(t, dir)
			resultPath := filepath.Join(dir, "results", id+".json")
			if kind == "retirement" {
				commands := filepath.Join(dir, "commands")
				if err := os.Chmod(commands, 0500); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(commands, 0700); err != nil {
						t.Error(err)
					}
				})
				raw, err := json.Marshal(agentfixture.Result{ID: id})
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(resultPath, raw, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				cancel()
				ctx, end := context.WithTimeout(t.Context(), 4*time.Second)
				defer end()
				if err := waitUntil(ctx, func() (bool, error) {
					_, err := os.Stat(filepath.Join(dir, "cancellations", id+".json"))
					return err == nil, nil
				}); err != nil {
					t.Fatal(err)
				}
				if kind == "ack-correlation" {
					raw, err := json.Marshal(agentfixture.Result{ID: "another-request"})
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(resultPath, raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "ack-read" {
					if err := os.WriteFile(resultPath, []byte("malformed"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			var err error
			select {
			case err = <-done:
			case <-t.Context().Done():
				t.Fatal("fixture did not return")
			}
			assertUncertainAdmission(t, f, err)
			if kind == "ack-timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("missing acknowledgment deadline: %v", err)
			}
			if kind != "retirement" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost caller cancellation: %v", err)
			}
			if _, err := os.Stat(filepath.Join(dir, "commands", id+".json")); err != nil {
				t.Fatalf("command evidence lost: %v", err)
			}
			if kind == "retirement" {
				if _, err := os.Stat(resultPath); err != nil {
					t.Fatalf("result evidence lost: %v", err)
				}
			}
		})
	}
}

func TestActorCLITerminalFailureLeavesAdmissionOpen(t *testing.T) {
	f, _ := startActorCLIFixture(t)
	result, err := f.ActorCLI(t.Context(), 0, "-c", "exit 7")
	if err == nil || errors.Is(err, agentfixture.ErrCleanupUncertain) || result.ExitCode != 7 {
		t.Fatalf("terminal failure=%+v %v", result, err)
	}
	result, err = f.ActorCLI(t.Context(), 0, "-c", "printf continued")
	if err != nil || result.Stdout != "continued" {
		t.Fatalf("terminal result closed admission: %+v %v", result, err)
	}
}
