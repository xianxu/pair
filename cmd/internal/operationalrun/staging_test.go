package operationalrun

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOperationalStageOwnerHelper(t *testing.T) {
	family := os.Getenv("OPERATIONAL_STAGE_FAMILY")
	if family == "" {
		return
	}
	s := Store{Namespace: os.Getenv("OPERATIONAL_STAGE_NAMESPACE")}
	task, err := s.task(RunKey{Hub: "hub", Task: "task", ID: "kept"})
	if err != nil {
		os.Exit(91)
	}
	release, err := taskLock(context.Background(), task, true)
	if err != nil {
		os.Exit(92)
	}
	defer release()
	stage := filepath.Join(task, ".collection-stage")
	if family == "run" {
		stage = filepath.Join(task, ".run-stage-123")
		if os.Mkdir(stage, 0700) != nil {
			os.Exit(93)
		}
		for _, name := range []string{"record.json", "output", ".record-234", ".output-345"} {
			if os.WriteFile(filepath.Join(stage, name), []byte(name), 0600) != nil {
				os.Exit(94)
			}
		}
	} else {
		if os.WriteFile(stage, []byte("uncommitted journal"), 0600) != nil {
			os.Exit(95)
		}
	}
	// Retain the native process and actual task flock until the parent kills
	// it. SIGKILL bypasses normal defers and leaves a real abandoned stage.
	fmt.Println("ready")
	for {
		time.Sleep(time.Second)
	}
}

func TestCollectionStageRecoveredWithoutJournal(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	record := sample("kept", time.Now())
	if err := s.Publish(t.Context(), record.Key, record, []byte("immutable output")); err != nil {
		t.Fatal(err)
	}
	task, err := s.task(record.Key)
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(task, ".collection-stage")
	if err = os.WriteFile(stage, []byte("uncommitted journal"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = s.SweepPage(t.Context(), time.Now(), Cursor{}, true, 100); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("orphan collection-stage retained without journal: %v", err)
	}
	got, out, err := s.Read(t.Context(), record.Key)
	if err != nil || got.Key.ID != record.Key.ID || string(out) != "immutable output" {
		t.Fatalf("changed committed bundle: %#v %q %v", got, out, err)
	}
}

func TestOperationalStagesNativeCrashLiveOwnerAndReadOnlyPaths(t *testing.T) {
	for _, family := range []string{"collection", "run"} {
		t.Run(family, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			record := sample("kept", time.Now())
			if err := s.Publish(t.Context(), record.Key, record, []byte("immutable")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(record.Key)
			stage := filepath.Join(task, ".collection-stage")
			if family == "run" {
				stage = filepath.Join(task, ".run-stage-123")
			}
			program, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), program, "-test.run=^TestOperationalStageOwnerHelper$", "--")
			cmd.Env = append(os.Environ(), "OPERATIONAL_STAGE_FAMILY="+family, "OPERATIONAL_STAGE_NAMESPACE="+s.Namespace)
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			ready := make(chan error, 1)
			go func() {
				line, e := bufio.NewReader(out).ReadString('\n')
				if e == nil && line != "ready\n" {
					e = fmt.Errorf("unexpected helper output %q", line)
				}
				ready <- e
			}()
			select {
			case err = <-ready:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("stage writer never became ready")
			}
			if _, _, _, err = s.SweepPage(t.Context(), time.Now(), Cursor{}, true, 100); !errors.Is(err, ErrBusy) {
				t.Fatalf("live owner did not fence cleanup: %v", err)
			}
			if _, err = os.Lstat(stage); err != nil {
				t.Fatal("live stage removed", err)
			}
			if err = cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err = cmd.Wait(); err == nil {
				t.Fatal("fixture did not die abruptly")
			}
			waited = true
			if _, _, err = s.Read(t.Context(), record.Key); err != nil {
				t.Fatal(err)
			}
			if _, err = s.List(t.Context(), "hub", "task", 100); err != nil {
				t.Fatal(err)
			}
			if _, _, _, err = s.SweepPage(t.Context(), time.Now(), Cursor{}, false, 100); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Lstat(stage); err != nil {
				t.Fatal("read-only access mutated stage", err)
			}
			rows, _, _, err := s.SweepPage(t.Context(), time.Now(), Cursor{}, true, 100)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if row.Key.ID == "" && row.Reason != "" {
					t.Fatalf("cleanup blocked: %#v", row)
				}
			}
			if _, err = os.Lstat(stage); !os.IsNotExist(err) {
				t.Fatalf("abandoned %s stage retained: %v", family, err)
			}
			_, output, err := s.Read(t.Context(), record.Key)
			if err != nil || string(output) != "immutable" {
				t.Fatalf("committed bundle changed: %q %v", output, err)
			}
		})
	}
}

func TestOperationalStageReplacementRefused(t *testing.T) {
	for _, kind := range []string{"collection-file", "collection-link", "run-directory", "run-payload", "run-new-member"} {
		t.Run(kind, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			record := sample("kept", time.Now())
			if err := s.Publish(t.Context(), record.Key, record, []byte("immutable")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(record.Key)
			stage := filepath.Join(task, ".collection-stage")
			run := strings.HasPrefix(kind, "run-")
			if run {
				stage = filepath.Join(task, ".run-stage-123")
				if err := os.Mkdir(stage, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stage, "output"), []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(stage, []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			outside := filepath.Join(t.TempDir(), "outside")
			heldPayload := filepath.Join(t.TempDir(), "held-payload")
			if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
				t.Fatal(err)
			}
			changed := false
			s.fault = func(step string) error {
				if changed || step != "cleanup-collection-captured" && step != "cleanup-run-captured" {
					return nil
				}
				changed = true
				switch kind {
				case "collection-file":
					if err := os.Rename(stage, stage+".held"); err != nil {
						return err
					}
					return os.WriteFile(stage, []byte("replacement"), 0600)
				case "collection-link":
					if err := os.Rename(stage, stage+".held"); err != nil {
						return err
					}
					return os.Symlink(outside, stage)
				case "run-directory":
					if err := os.Rename(stage, stage+".held"); err != nil {
						return err
					}
					if err := os.Mkdir(stage, 0700); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(stage, "output"), []byte("replacement"), 0600)
				case "run-payload":
					path := filepath.Join(stage, "output")
					if err := os.Rename(path, heldPayload); err != nil {
						return err
					}
					return os.WriteFile(path, []byte("replacement"), 0600)
				case "run-new-member":
					return os.WriteFile(filepath.Join(stage, "unexpected"), []byte("replacement"), 0600)
				}
				return nil
			}
			rows, _, _, err := s.SweepPage(t.Context(), time.Now(), Cursor{}, true, 100)
			if err != nil {
				t.Fatal(err)
			}
			blocked := false
			for _, row := range rows {
				if row.Key.ID == "" && row.Reason != "" {
					blocked = true
				}
			}
			if !changed || !blocked {
				t.Fatalf("replacement not refused: changed=%v rows=%#v", changed, rows)
			}
			if _, err = os.Lstat(stage); err != nil {
				t.Fatal("replacement stage removed", err)
			}
			if output, err := os.ReadFile(outside); err != nil || string(output) != "outside" {
				t.Fatal("followed replacement link", string(output), err)
			}
			if kind == "collection-file" {
				if output, err := os.ReadFile(stage); err != nil || string(output) != "replacement" {
					t.Fatal("replacement file lost", string(output), err)
				}
			}
			if kind == "run-payload" || kind == "run-directory" {
				if output, err := os.ReadFile(filepath.Join(stage, "output")); err != nil || string(output) != "replacement" {
					t.Fatal("replacement payload lost", string(output), err)
				}
			}
			if kind == "run-payload" {
				if output, err := os.ReadFile(heldPayload); err != nil || string(output) != "old" {
					t.Fatal("held original payload changed", string(output), err)
				}
			}
			if kind == "run-new-member" {
				if output, err := os.ReadFile(filepath.Join(stage, "output")); err != nil || string(output) != "old" {
					t.Fatal("partial cleanup before changed-set refusal", string(output), err)
				}
			}
			_, output, err := s.Read(t.Context(), record.Key)
			if err != nil || string(output) != "immutable" {
				t.Fatal("committed bundle changed", string(output), err)
			}
		})
	}
}

func TestOperationalStageUnsafeMembersRemainBlocked(t *testing.T) {
	for _, kind := range []string{"collection-directory", "collection-fifo", "collection-link", "run-directory-member", "run-fifo-member", "run-link-member", "run-unknown-member"} {
		t.Run(kind, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			record := sample("kept", time.Now())
			if err := s.Publish(t.Context(), record.Key, record, []byte("immutable")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(record.Key)
			stage := filepath.Join(task, ".collection-stage")
			payload := stage
			if strings.HasPrefix(kind, "run-") {
				stage = filepath.Join(task, ".run-stage-123")
				if err := os.Mkdir(stage, 0700); err != nil {
					t.Fatal(err)
				}
				payload = filepath.Join(stage, "output")
			}
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
				t.Fatal(err)
			}
			switch {
			case strings.Contains(kind, "directory"):
				if err := os.Mkdir(payload, 0700); err != nil {
					t.Fatal(err)
				}
			case strings.Contains(kind, "fifo"):
				if err := unix.Mkfifo(payload, 0600); err != nil {
					t.Fatal(err)
				}
			case strings.Contains(kind, "link"):
				if err := os.Symlink(outside, payload); err != nil {
					t.Fatal(err)
				}
			case strings.Contains(kind, "unknown"):
				payload = filepath.Join(stage, "unknown")
				if err := os.WriteFile(payload, []byte("unowned"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			rows, _, _, err := s.SweepPage(ctx, time.Now(), Cursor{}, true, 100)
			if err != nil {
				t.Fatal(err)
			}
			blocked := false
			for _, row := range rows {
				if row.Key.ID == "" && row.Reason != "" {
					blocked = true
				}
			}
			if !blocked {
				t.Fatal("unsafe staging entry was not reported")
			}
			if _, err = os.Lstat(payload); err != nil {
				t.Fatal("unsafe/unknown entry removed", err)
			}
			if data, err := os.ReadFile(outside); err != nil || string(data) != "outside" {
				t.Fatal("outside file changed", string(data), err)
			}
			_, output, err := s.Read(t.Context(), record.Key)
			if err != nil || string(output) != "immutable" {
				t.Fatal("committed bundle changed", string(output), err)
			}
		})
	}
}

func TestOperationalCollectionStageRecoveredByNextPublish(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	old := sample("kept", time.Now())
	if err := s.Publish(t.Context(), old.Key, old, []byte("immutable")); err != nil {
		t.Fatal(err)
	}
	task, _ := s.task(old.Key)
	stage := filepath.Join(task, ".collection-stage")
	if err := os.WriteFile(stage, []byte("uncommitted"), 0600); err != nil {
		t.Fatal(err)
	}
	next := sample("next", time.Now())
	if err := s.Publish(t.Context(), next.Key, next, []byte("new output")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatal("publisher retained stale journal stage", err)
	}
	_, output, err := s.Read(t.Context(), old.Key)
	if err != nil || string(output) != "immutable" {
		t.Fatal("existing bundle changed", string(output), err)
	}
}

func TestPublishReturnedErrorPreservesReplacement(t *testing.T) {
	testPublishReplacement(t, true)
}

func TestPublishRefusesReplacementBeforeRename(t *testing.T) {
	testPublishReplacement(t, false)
}

func testPublishReplacement(t *testing.T, returnedError bool) {
	t.Helper()
	for _, member := range []string{"root", "record.json", "output"} {
		t.Run(member, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			record := sample("new", time.Now())
			task, _ := s.task(record.Key)
			var foreign string
			s.fault = func(step string) error {
				if step != "publish-ready" {
					return nil
				}
				entries, err := os.ReadDir(task)
				if err != nil {
					t.Fatal(err)
				}
				var stage string
				for _, entry := range entries {
					if stageName.MatchString(entry.Name()) {
						stage = filepath.Join(task, entry.Name())
					}
				}
				if stage == "" {
					t.Fatal("missing stage")
				}
				if member == "root" {
					if err := os.Rename(stage, stage+"-original"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(stage, 0700); err != nil {
						t.Fatal(err)
					}
					foreign = filepath.Join(stage, "output")
				} else {
					foreign = filepath.Join(stage, member)
					if err := os.Rename(foreign, filepath.Join(task, "original-"+member)); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(foreign, []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
				if returnedError {
					return errors.New("returned publication failure")
				}
				return nil
			}
			if err := s.Publish(t.Context(), record.Key, record, []byte("own output")); err == nil {
				t.Fatal("missing error")
			}
			if _, err := os.Lstat(filepath.Join(task, "runs", record.Key.ID)); !os.IsNotExist(err) {
				t.Fatalf("published substituted bundle: %v", err)
			}
			if got, err := os.ReadFile(foreign); err != nil || string(got) != "foreign" {
				t.Fatalf("foreign replacement changed: %q %v", got, err)
			}
		})
	}
}
