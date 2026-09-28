package wrapcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

var realStartWatcherProcess = startWatcherProcess

// Ordinary wrap tests suppress real watchers: os.Executable() is the test
// binary and Pair sessions supply a live PAIR_LAUNCH_ORDINAL. The detach
// acceptance helper explicitly opts into executing the real watcher command.
func TestMain(m *testing.M) {
	detachWatcherHelper()
	startWatcherProcess = func([]string, []string) error { return nil }
	os.Exit(m.Run())
}

func TestLaunchWatcherArgvNamesTheLaunchWrapRuns(t *testing.T) {
	bound := time.Date(2026, 9, 28, 11, 0, 0, 7, time.UTC)
	cwd, _ := os.Getwd()
	env := []string{"PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "PAIR_LAUNCH_ORDINAL=7"}
	got := launchWatcherArgv("/pair", []string{"/usr/bin/codex", "--no-alt-screen"}, env, bound)
	want := []string{"/pair", "session-watch", "codex", "work", cwd, "--scope-key", "scope", "--launch-ordinal", "7",
		"--pid-not-before", bound.Format(time.RFC3339Nano), "--", "--no-alt-screen"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("watcher argv = %v, want %v", got, want)
	}
	for name, tc := range map[string]struct {
		argv []string
		env  []string
	}{
		"no ordinal":        {[]string{"codex"}, []string{"PAIR_TAG=work", "PAIR_SCOPE_KEY=scope"}},
		"zero ordinal":      {[]string{"codex"}, []string{"PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "PAIR_LAUNCH_ORDINAL=0"}},
		"malformed ordinal": {[]string{"codex"}, []string{"PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "PAIR_LAUNCH_ORDINAL=x"}},
		"no tag":            {[]string{"codex"}, []string{"PAIR_SCOPE_KEY=scope", "PAIR_LAUNCH_ORDINAL=7"}},
		"no scope":          {[]string{"codex"}, []string{"PAIR_TAG=work", "PAIR_LAUNCH_ORDINAL=7"}},
		"unsupported agent": {[]string{"/bin/sh"}, env},
	} {
		if got := launchWatcherArgv("/pair", tc.argv, tc.env, bound); got != nil {
			t.Errorf("%s: watcher argv = %v, want none", name, got)
		}
	}
}

// The #329 regression at the spawn site: a wrap that starts an agent for a
// launch spawns exactly one watcher for that launch, bounded no later than the
// agent-pid file it will read. Before #329 wrap spawned none here -- the
// launcher did, in the pair client's process group, where detach killed it.
func TestWrapSpawnsOneWatcherForItsLaunch(t *testing.T) {
	isolateNotificationSockets(t)
	data := filepath.Join(t.TempDir(), "repos", "scope")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_RETENTION_START_ID", "")
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_TAG", "watch-test")
	t.Setenv("PAIR_SCOPE_KEY", "scope")
	t.Setenv("PAIR_LAUNCH_ORDINAL", "3")
	t.Setenv("HOME", t.TempDir())
	fakeCodex := filepath.Join(t.TempDir(), "codex")
	if err := os.Symlink("/bin/sh", fakeCodex); err != nil {
		t.Fatal(err)
	}
	pidPath := filepath.Join(data, "agent-pid-watch-test")

	type spawn struct {
		argv   []string
		pidMod time.Time
		pidErr error
	}
	spawns := make(chan spawn, 4)
	oldStartWatcher := startWatcherProcess
	defer func() { startWatcherProcess = oldStartWatcher }()
	startWatcherProcess = func(argv, _ []string) error {
		info, err := os.Stat(pidPath)
		s := spawn{argv: append([]string(nil), argv...), pidErr: err}
		if err == nil {
			s.pidMod = info.ModTime()
		}
		spawns <- s
		return nil
	}

	before := time.Now()
	code := Run([]string{fakeCodex, "-c", "sleep 0.3"}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("wrap exited %d", code)
	}
	close(spawns)
	var got []spawn
	for s := range spawns {
		got = append(got, s)
	}
	if len(got) != 1 {
		t.Fatalf("watcher spawns = %d, want exactly 1: %+v", len(got), got)
	}
	argv := got[0].argv
	if len(argv) < 5 || argv[1] != "session-watch" || argv[2] != "codex" || argv[3] != "watch-test" {
		t.Fatalf("watcher argv = %v", argv)
	}
	if !containsArgPair(argv, "--launch-ordinal", "3") || !containsArgPair(argv, "--scope-key", "scope") {
		t.Fatalf("watcher argv = %v, want this launch's ordinal and scope", argv)
	}
	if tail := strings.Join(argv[len(argv)-3:], " "); tail != "-- -c sleep 0.3" {
		t.Fatalf("watcher agent args = %q, want the agent's argv", tail)
	}
	if got[0].pidErr != nil {
		t.Fatalf("agent-pid file absent when the watcher spawned: %v", got[0].pidErr)
	}
	var bound time.Time
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "--pid-not-before" {
			bound, _ = time.Parse(time.RFC3339Nano, argv[i+1])
		}
	}
	if bound.Before(before) || got[0].pidMod.Before(bound) {
		t.Fatalf("pid bound %v must fall between test start %v and the agent-pid mtime %v", bound, before, got[0].pidMod)
	}
}

// The detach half of #329: Couch detach signals the pair client's whole process
// group. The watcher must be in a session of its own so that signal misses it.
func TestStartWatcherProcessEscapesTheSpawnersProcessGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	script := `echo $$ > "$0"; exec sleep 5`
	if err := realStartWatcherProcess([]string{"/bin/sh", "-c", script, pidFile}, os.Environ()); err != nil {
		t.Fatal(err)
	}
	var pid int
	deadline := time.Now().Add(3 * time.Second)
	for pid == 0 {
		if raw, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
		}
		if pid == 0 && time.Now().After(deadline) {
			t.Fatal("watcher child never reported its pid")
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	childGroup, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatal(err)
	}
	if childGroup == syscall.Getpgrp() {
		t.Fatalf("watcher shares the spawner's process group %d; a detach would kill it", childGroup)
	}
	if childGroup != pid {
		t.Fatalf("watcher pgid = %d, want its own group %d", childGroup, pid)
	}
}
