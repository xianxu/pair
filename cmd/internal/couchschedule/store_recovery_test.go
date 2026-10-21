package couchschedule

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const abandonedStateName = ".state-0123456789abcdef0123456789abcdef.tmp"

func TestLedgerCrashHelper(t *testing.T) {
	dir, step := os.Getenv("PAIR_LEDGER_RECOVERY_DIR"), os.Getenv("PAIR_LEDGER_RECOVERY_STEP")
	if dir == "" {
		return
	}
	store := StateStore{Dir: dir, stateStep: func(got, name string) error {
		if got != step {
			return nil
		}
		if os.Getenv("PAIR_LEDGER_RECOVERY_HOLD") == "1" {
			fmt.Fprintln(os.Stdout, name)
			_, _ = io.Copy(io.Discard, os.Stdin)
		}
		os.Exit(73)
		return nil
	}}
	if step == "post-rename" {
		store.BeforeDirectorySync = func() error { os.Exit(73); return nil }
	}
	if err := store.Update(t.Context(), func(l *Ledger) error {
		if l.Unknown == nil {
			l.Unknown = map[string]json.RawMessage{}
		}
		l.Unknown["marker"] = json.RawMessage(`"new"`)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func ledgerCrashCommand(t *testing.T, dir, step string) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLedgerCrashHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), "PAIR_LEDGER_RECOVERY_DIR="+dir, "PAIR_LEDGER_RECOVERY_STEP="+step)
	return cmd
}
func initializeRecoveryLedger(t *testing.T, dir string) {
	t.Helper()
	if err := (StateStore{Dir: dir}).Update(t.Context(), func(l *Ledger) error {
		l.Unknown = map[string]json.RawMessage{"marker": json.RawMessage(`"old"`)}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func ledgerTemps(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".state-") {
			names = append(names, entry.Name())
		}
	}
	return names
}

func TestLedgerAbruptPublicationRecovery(t *testing.T) {
	for _, step := range []string{"created", "written", "synced", "pre-rename", "post-rename"} {
		t.Run(step, func(t *testing.T) {
			dir := catalogRoot(t)
			initializeRecoveryLedger(t, dir)
			cmd := ledgerCrashCommand(t, dir, step)
			output, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 73 {
				t.Fatalf("helper did not crash at %s: %v %s", step, err, output)
			}
			expected := `"old"`
			if step == "post-rename" {
				expected = `"new"`
			} else if len(ledgerTemps(t, dir)) != 1 {
				t.Fatalf("crash did not retain stage: %v", ledgerTemps(t, dir))
			}
			store := StateStore{Dir: dir}
			before, err := store.View(t.Context())
			if err != nil || string(before.Unknown["marker"]) != expected {
				t.Fatalf("View promoted/changed snapshot: %+v %v", before, err)
			}
			if step != "post-rename" && len(ledgerTemps(t, dir)) != 1 {
				t.Fatal("View removed abandoned stage")
			}
			if err := store.Update(t.Context(), func(*Ledger) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if left := ledgerTemps(t, dir); len(left) != 0 {
				t.Fatalf("abandoned ledger stages survive successful Update: %v", left)
			}
			after, err := store.View(t.Context())
			if err != nil || string(after.Unknown["marker"]) != expected {
				t.Fatalf("recovery changed authority: %+v %v", after, err)
			}
		})
	}
}

func TestLedgerRecoveryExcludesLiveWriter(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	cmd := ledgerCrashCommand(t, dir, "created")
	cmd.Env = append(cmd.Env, "PAIR_LEDGER_RECOVERY_HOLD=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		stdin.Close()
		if !waited {
			_ = cmd.Wait()
		}
	}()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(line)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	called := false
	err = (StateStore{Dir: dir}).Update(ctx, func(*Ledger) error { called = true; return nil })
	if !errors.Is(err, context.DeadlineExceeded) || called {
		t.Fatalf("live-writer exclusion: callback=%v err=%v", called, err)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("live stage removed: %v", err)
	}
	stdin.Close()
	err = cmd.Wait()
	waited = true
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 73 {
		t.Fatalf("writer did not exit abruptly: %v", err)
	}
	if err := (StateStore{Dir: dir}).Update(t.Context(), func(*Ledger) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if left := ledgerTemps(t, dir); len(left) != 0 {
		t.Fatalf("dead writer retained stages: %v", left)
	}
}

func TestLedgerRecoveryIdentityAndOwnershipFences(t *testing.T) {
	for _, shape := range []string{"symlink", "hardlink", "directory", "oversized", "public-mode", "replacement"} {
		t.Run(shape, func(t *testing.T) {
			dir := catalogRoot(t)
			initializeRecoveryLedger(t, dir)
			outside := filepath.Join(catalogRoot(t), "outside")
			if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
				t.Fatal(err)
			}
			stage := filepath.Join(dir, abandonedStateName)
			switch shape {
			case "symlink":
				if err := os.Symlink(outside, stage); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(outside, stage); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(stage, 0700); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.WriteFile(stage, []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if shape == "oversized" {
				if err := os.Truncate(stage, maxScheduleStateBytes+1); err != nil {
					t.Fatal(err)
				}
			}
			if shape == "public-mode" {
				if err := os.Chmod(stage, 0644); err != nil {
					t.Fatal(err)
				}
			}
			store := StateStore{Dir: dir}
			if shape == "replacement" {
				store.stateStep = func(step, name string) error {
					if step == "cleanup-observed" && name == abandonedStateName {
						if err := os.Rename(stage, stage+".saved"); err != nil {
							return err
						}
						return os.WriteFile(stage, []byte("replacement"), 0600)
					}
					return nil
				}
			}
			called := false
			err := store.Update(t.Context(), func(*Ledger) error { called = true; return nil })
			if err == nil || called {
				t.Fatalf("unsafe residue admitted: callback=%v err=%v", called, err)
			}
			if _, err := os.Lstat(stage); err != nil {
				t.Fatalf("unsafe/replaced stage deleted: %v", err)
			}
			data, err := os.ReadFile(outside)
			if err != nil || string(data) != "outside" {
				t.Fatalf("outside file changed: %q %v", data, err)
			}
			if shape == "replacement" {
				data, err := os.ReadFile(stage)
				if err != nil || string(data) != "replacement" {
					t.Fatalf("replacement changed: %q %v", data, err)
				}
			}
		})
	}
}

func TestLedgerRecoveryIgnoresUnknownNamesAndSyncsCleanup(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	unknown := []string{".state-short.tmp", ".state-FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF.tmp", abandonedStateName + ".saved", "notes"}
	for _, name := range unknown {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := (StateStore{Dir: dir}).Update(t.Context(), func(*Ledger) error { return nil }); err != nil {
		t.Fatal(err)
	}
	for _, name := range unknown {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("unknown removed %s: %v", name, err)
		}
	}
	stage := filepath.Join(dir, abandonedStateName)
	if err := os.WriteFile(stage, []byte("residue"), 0600); err != nil {
		t.Fatal(err)
	}
	store := StateStore{Dir: dir, stateStep: func(step, _ string) error {
		if step == "cleanup-directory-sync" {
			return errors.New("cleanup sync refused")
		}
		return nil
	}}
	called := false
	if err := store.Update(t.Context(), func(*Ledger) error { called = true; return nil }); err == nil || called {
		t.Fatalf("cleanup durability error hidden: callback=%v err=%v", called, err)
	}
}

func TestLedgerRepeatedCrashesDoNotAccumulateStages(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	for i := 0; i < 3; i++ {
		output, err := ledgerCrashCommand(t, dir, "created").CombinedOutput()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 73 {
			t.Fatalf("crash %d: %v %s", i, err, output)
		}
		if left := ledgerTemps(t, dir); len(left) != 1 {
			t.Fatalf("crash %d accumulated stages: %v", i, left)
		}
	}
	if err := (StateStore{Dir: dir}).Update(t.Context(), func(*Ledger) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if left := ledgerTemps(t, dir); len(left) != 0 {
		t.Fatalf("stages remain: %v", left)
	}
}

func TestLedgerRecoveryInventoryBound(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	stage := filepath.Join(dir, abandonedStateName)
	if err := os.WriteFile(stage, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxLedgerDirectoryEntries; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("unknown-%04d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	called := false
	err := (StateStore{Dir: dir}).Update(t.Context(), func(*Ledger) error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("oversized recovery scan admitted: callback=%v err=%v", called, err)
	}
	if _, err := os.Stat(stage); err != nil {
		t.Fatalf("partial overflow cleanup removed stage: %v", err)
	}
}

func TestLedgerRecoveryCancellationPreservesObservedStage(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	stage := filepath.Join(dir, abandonedStateName)
	if err := os.WriteFile(stage, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	store := StateStore{Dir: dir, stateStep: func(step, _ string) error {
		if step == "cleanup-observed" {
			cancel()
		}
		return nil
	}}
	called := false
	err := store.Update(ctx, func(*Ledger) error { called = true; return nil })
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("cancelled recovery admitted: callback=%v err=%v", called, err)
	}
	if _, err := os.Stat(stage); err != nil {
		t.Fatalf("cancelled cleanup removed stage: %v", err)
	}
}

func TestLedgerReturnedErrorCleanupPreservesReplacement(t *testing.T) {
	dir := catalogRoot(t)
	initializeRecoveryLedger(t, dir)
	var replaced string
	store := StateStore{Dir: dir, stateStep: func(step, name string) error {
		if step != "created" {
			return nil
		}
		replaced = filepath.Join(dir, name)
		if err := os.Rename(replaced, replaced+".saved"); err != nil {
			return err
		}
		if err := os.WriteFile(replaced, []byte("replacement"), 0600); err != nil {
			return err
		}
		return errors.New("publication refused after replacement")
	}}
	if err := store.Update(t.Context(), func(*Ledger) error { return nil }); err == nil {
		t.Fatal("publication fault hidden")
	}
	raw, err := os.ReadFile(replaced)
	if err != nil || string(raw) != "replacement" {
		t.Fatalf("returned-error cleanup removed replacement: %q %v", raw, err)
	}
}
