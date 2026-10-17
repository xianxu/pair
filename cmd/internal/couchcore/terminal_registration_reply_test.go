package couchcore

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

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
