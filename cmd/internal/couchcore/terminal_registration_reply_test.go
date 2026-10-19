package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"strconv"
)

// Writes land on a real filesystem; the observer runs synchronously between
// publications, so the listing consumer sees the intermediate state every run.
type registrationReadinessObserver struct {
	osfs.FS
	writes     int
	failAt     int
	failure    error
	afterWrite func(int)
}

func (fs *registrationReadinessObserver) WriteAtomic(path, data string) error {
	fs.writes++
	if fs.writes == fs.failAt {
		return fs.failure
	}
	if err := fs.FS.WriteAtomic(path, data); err != nil {
		return err
	}
	fs.afterWrite(fs.writes)
	return nil
}

func TestRegistrationFixtureReadinessPublication(t *testing.T) {
	for _, failAt := range []int{0, 1, 2} {
		t.Run(strconv.Itoa(failAt), func(t *testing.T) {
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			command := filepath.Join(dir, "zellij")
			if err := os.Symlink(binary, command); err != nil {
				t.Fatal(err)
			}
			listingPath, agentPath := filepath.Join(dir, "listing-ready"), filepath.Join(dir, "agent-ready")
			env := testpair.Environment(t, dir, "PAIR_TEST_COUCH_TERMINAL_FIXTURE=1", "PAIR_SESSION_NAME=pair-fixture", "PAIR_TEST_ZELLIJ_READY="+listingPath, "PAIR_TEST_ZELLIJ_LOG="+filepath.Join(dir, "calls"))
			record, err := readiness.Encode(readiness.ReadyRecord{Tag: "fixture", Agent: "claude", Session: "pair-fixture", Nonce: "nonce", PID: os.Getpid(), AgentIdentity: procutil.Identity(strconv.Itoa(os.Getpid()))})
			if err != nil {
				t.Fatal(err)
			}
			injected := errors.New("publication refused")
			observed := false
			fs := &registrationReadinessObserver{failAt: failAt, failure: injected, afterWrite: func(write int) {
				if write != 1 {
					return
				}
				observed = true
				if _, err := os.Stat(agentPath); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("agent-ready visible between publications: %v", err)
				}
				for _, row := range []struct{ flag, want string }{{"--short", "pair-fixture\n"}, {"--no-formatting", "pair-fixture [Created 1s ago]\n"}} {
					cmd := exec.CommandContext(t.Context(), command, "list-sessions", row.flag)
					cmd.Env = env
					out, err := cmd.CombinedOutput()
					if err != nil || string(out) != row.want {
						t.Errorf("listing between publications %s = %q, %v; want %q", row.flag, out, err, row.want)
					}
				}
			}}
			err = publishRegistrationFixtureReadiness(fs, listingPath, agentPath, string(record))
			if failAt != 0 {
				if !errors.Is(err, injected) || fs.writes != failAt {
					t.Fatalf("failed publication: writes=%d err=%v", fs.writes, err)
				}
				if _, err := os.Stat(agentPath); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("failed publication announced agent-ready: %v", err)
				}
			} else {
				if err != nil || fs.writes != 2 {
					t.Fatalf("publication: writes=%d err=%v", fs.writes, err)
				}
				if raw, err := os.ReadFile(agentPath); err != nil || string(raw) != string(record) {
					t.Errorf("agent-ready = %q, %v; want exact record %q", raw, err, record)
				}
			}
			if observed != (failAt != 1) {
				t.Errorf("between-write observer ran=%v, failAt=%d", observed, failAt)
			}
			raw, err := os.ReadFile(listingPath)
			if failAt == 1 {
				if !errors.Is(err, os.ErrNotExist) {
					t.Errorf("first write failure published listing: %q, %v", raw, err)
				}
			} else if err != nil || string(raw) != "ready" {
				t.Errorf("listing-ready not retained: %q, %v", raw, err)
			}
		})
	}
}

func TestRegistrationTerminalFixtureReturnsExplicitEmptyObservations(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.Symlink(binary, path); err != nil {
		t.Fatal(err)
	}
	env := testpair.Environment(t, dir,
		"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1",
		"PAIR_TEST_ZELLIJ_LOG="+filepath.Join(dir, "calls"),
		"PAIR_TEST_ZELLIJ_READY="+filepath.Join(dir, "not-ready"),
	)
	for _, args := range [][]string{{"list-sessions", "--short"}, {"list-sessions", "--no-formatting"}} {
		cmd := exec.CommandContext(t.Context(), path, args...)
		cmd.Env = env
		var stderr strings.Builder
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		if err == nil || cmd.ProcessState.ExitCode() != 1 || len(stdout) != 0 || stderr.String() != "No active zellij sessions found.\n" {
			t.Errorf("%v = stdout %q stderr %q error %v; want explicit no-active-sessions exit1", args, stdout, stderr.String(), err)
		}
	}
	cmd := exec.CommandContext(t.Context(), path, "--session", "fixture", "action", "list-clients")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil || string(out) != "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" {
		t.Fatalf("clients = %q, %v; want header-only detached observation", out, err)
	}
}

func TestRegistrationTerminalFixtureDistinguishesListingForms(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.Symlink(binary, path); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(dir, "ready")
	if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	env := testpair.Environment(t, dir, "PAIR_TEST_COUCH_TERMINAL_FIXTURE=1", "PAIR_SESSION_NAME=pair-fixture", "PAIR_TEST_ZELLIJ_READY="+ready, "PAIR_TEST_ZELLIJ_LOG="+filepath.Join(dir, "calls"))
	for _, row := range []struct{ flag, want string }{{"--short", "pair-fixture\n"}, {"--no-formatting", "pair-fixture [Created 1s ago]\n"}} {
		cmd := exec.CommandContext(t.Context(), path, "list-sessions", row.flag)
		cmd.Env = env
		out, err := cmd.Output()
		if err != nil || string(out) != row.want {
			t.Errorf("%s = %q, %v; want %q", row.flag, out, err, row.want)
		}
	}
}

func TestRegistrationNestedWrapperUsesDisposableEnvironment(t *testing.T) {
	root := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pair-wrap", "claude"} {
		if err := os.Symlink(exe, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	report, release := filepath.Join(root, "env.json"), filepath.Join(root, "release")
	foreignEvents := filepath.Join(t.TempDir(), "foreign-events")
	if err := os.WriteFile(foreignEvents, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	// Inject poison into the fixture parent deliberately. The nested wrapper
	// must reconstruct explicit authority rather than passing this onward.
	env := testpair.Environment(t, root,
		"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1", "PAIR_TAG=fixture", "PAIR_AGENT=claude",
		"PAIR_SESSION_NAME=fixture", "PAIR_LAUNCH_NONCE=nonce",
		"PAIR_AGENT_READY_PATH="+filepath.Join(root, "agent-ready"),
		"PAIR_TEST_ZELLIJ_READY="+filepath.Join(root, "ready"),
		"PAIR_TEST_ZELLIJ_RELEASE="+release, "PAIR_TEST_ZELLIJ_ENV_REPORT="+report,
		"PAIR_SCROLLBACK_EVENTS_PATH="+foreignEvents, "COUCH_TREE=foreign-owner",
		"UNLISTED_FIXTURE_SECRET=never-forward",
	)
	cmd := exec.Command(filepath.Join(root, "pair-wrap"))
	cmd.Dir, cmd.Env = root, env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(release, []byte("release"), 0600)
		if err := cmd.Wait(); err != nil {
			t.Errorf("nested wrapper: %v", err)
		}
	})
	var raw []byte
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		raw, err = os.ReadFile(report)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var observed []string
	if err := json.Unmarshal(raw, &observed); err != nil {
		t.Fatalf("nested environment report: %v", err)
	}
	if got := environmentValue(observed, "HOME"); got != root {
		t.Fatalf("nested HOME=%q, want %q", got, root)
	}
	for _, key := range []string{"PAIR_SCROLLBACK_EVENTS_PATH", "COUCH_TREE", "UNLISTED_FIXTURE_SECRET"} {
		if value := environmentValue(observed, key); value != "" {
			t.Errorf("nested process inherited %s=%q", key, value)
		}
	}
	if raw, err := os.ReadFile(foreignEvents); err != nil || string(raw) != "preserve" {
		t.Fatalf("foreign fixture capture changed: %v", err)
	}
}
