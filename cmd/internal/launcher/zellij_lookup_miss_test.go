package launcher

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// Exact native 0.45.1 rows from the #761 three-session capture (186 bytes).
const lookupMissNativeListing = "\x1b[32;1mp761-f0ab63ebb7b4af9a-0\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n" +
	"\x1b[32;1mp761-f0ab63ebb7b4af9a-1\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n" +
	"\x1b[32;1mp761-f0ab63ebb7b4af9a-2\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n"
const lookupMissNativeStderr = "Session 'p761-f0ab63ebb7b4af9a-1' not found. The following sessions are active:\n"

func TestZellijLookupMissNativeReadRecovery(t *testing.T) {
	for _, tail := range [][]string{{"list-panes", "--json", "--command"}, {"list-panes", "--json", "--command", "--state"}, {"list-clients"}} {
		t.Run(tail[0]+tail[len(tail)-1], func(t *testing.T) {
			args := append([]string{"--session", "p761-f0ab63ebb7b4af9a-1", "action"}, tail...)
			good := "[]\n"
			if tail[0] == "list-clients" {
				good = "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
			}
			f := newEmptyReplyFake(t, emptyReplyFrame{Args: args, Stdout: lookupMissNativeListing, Stderr: lookupMissNativeStderr}, emptyReplyFrame{Args: args, Stdout: good})
			var events []ZellijQueryObservation
			out, err := f.source.runContext(WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { events = append(events, e) }), args...)
			if err != nil || string(out) != good || f.calls(t) != 2 {
				t.Fatalf("reply=%q err=%v calls=%d", out, err, f.calls(t))
			}
			if len(events) != 2 || events[0].Err == nil || events[1].Err != nil || !events[0].EffectiveDeadline.Equal(events[1].EffectiveDeadline) {
				t.Fatalf("events=%+v", events)
			}
		})
	}
}

func lookupMissSocketFixture(t *testing.T, name string) string {
	t.Helper()
	root, err := os.MkdirTemp("", "s761-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("ZELLIJ_SOCKET_DIR", root)
	dir := filepath.Join(root, "contract_version_1")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
	})
	return path
}

func TestZellijLookupMissExitOneRequiresSocketAndKeepsBound(t *testing.T) {
	args := []string{"--session", "pair-a", "action", "list-clients"}
	for _, live := range []bool{false, true} {
		t.Run(map[bool]string{false: "gone", true: "socket"}[live], func(t *testing.T) {
			if live {
				lookupMissSocketFixture(t, "pair-a")
			} else {
				t.Setenv("ZELLIJ_SOCKET_DIR", t.TempDir())
			}
			frames := []emptyReplyFrame{{Args: args, Exit: 1, Stderr: "There is no active session!\n"}}
			if live {
				frames = append(frames, emptyReplyFrame{Args: args, Stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"})
			}
			f := newEmptyReplyFake(t, frames...)
			_, err := f.source.runContext(t.Context(), args...)
			if live && (err != nil || f.calls(t) != 2) {
				t.Fatalf("live socket err=%v calls=%d", err, f.calls(t))
			}
			if !live && (err == nil || f.calls(t) != 1) {
				t.Fatalf("gone socket err=%v calls=%d", err, f.calls(t))
			}
		})
	}
	t.Run("three-attempt-cap", func(t *testing.T) {
		lookupMissSocketFixture(t, "pair-a")
		frame := emptyReplyFrame{Args: args, Exit: 1, Stderr: "There is no active session!\n"}
		f := newEmptyReplyFake(t, frame, frame, frame)
		_, err := f.source.runContext(t.Context(), args...)
		if err == nil || f.calls(t) != 3 {
			t.Fatalf("err=%v calls=%d", err, f.calls(t))
		}
	})
}

func TestZellijLookupMissCancellationDoesNotReissue(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	source := ZellijSource{queryRun: func(cmd *exec.Cmd) error {
		calls++
		_, _ = io.WriteString(cmd.Stdout, lookupMissNativeListing)
		_, _ = io.WriteString(cmd.Stderr, lookupMissNativeStderr)
		cancel()
		return nil
	}}
	_, err := source.runContext(ctx, "--session", "p761-f0ab63ebb7b4af9a-1", "action", "list-clients")
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestZellijLookupMissClassifierRejectsUnrelatedBytes(t *testing.T) {
	args := []string{"--session", "pair-a", "action", "list-panes", "--json", "--command"}
	listing := "\x1b[32;1mpair-a\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n"
	message := "Session 'pair-a' not found. The following sessions are active:\n"
	for _, tc := range []struct {
		name, out, err string
		exit           int
		socket, want   bool
	}{
		{"colored", listing, message, 0, false, true},
		{"plain", "pair-a [Created 3s ago]\n", message, 0, false, true},
		{"socket", "", "There is no active session!\n", 1, true, true},
		{"no-socket", "", "There is no active session!\n", 1, false, false},
		{"spaced-empty", " ", "There is no active session!\n", 1, true, false},
		{"wrong-exit", "", "There is no active session!\n", 2, true, false},
		{"no-newline", "", "There is no active session!", 1, true, false},
		{"wrong-target", listing, "Session 'other' not found. The following sessions are active:\n", 0, true, false},
		{"stderr-tail", listing, message + "unexpected\n", 0, true, false},
		{"missing-target", "other [Created 3s ago]\n", message, 0, true, false},
		{"exited-target", "pair-a [Created 3s ago] (EXITED - attach to resurrect)\n", message, 0, true, false},
		{"colored-exited", "\x1b[32;1mpair-a\x1b[m [Created \x1b[35;1m3s\x1b[m ago] (\x1b[31;1mEXITED\x1b[m - attach to resurrect)\n", message, 0, true, false},
		{"json", "[{", message, 0, true, false},
		{"unrelated-escape", listing + "\x1b[0m", message, 0, true, false},
		{"trailing-data", listing + "other", message, 0, true, false},
		{"duplicate", listing + listing, message, 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryableSessionLookupMiss(args, []byte(tc.out), []byte(tc.err), tc.exit, tc.socket); got != tc.want {
				t.Fatalf("retryable=%v want=%v", got, tc.want)
			}
		})
	}
	for _, badArgs := range [][]string{
		{"--session", "pair-a", "action", "close-pane"},
		{"--session", "pair-a", "action", "list-panes", "--json", "--future"},
		{"list-sessions", "--no-formatting"},
		{"--session", "../pair-a", "action", "list-clients"},
		{"--session", "pair-a\n", "action", "list-clients"},
	} {
		if retryableSessionLookupMiss(badArgs, []byte(listing), []byte(message), 0, true) || retryableSessionLookupMiss(badArgs, nil, []byte("There is no active session!\n"), 1, true) {
			t.Fatalf("unsupported argv retried: %q", badArgs)
		}
	}
}

func TestParseColoredSessionListingRequiresExactWholeNativeEnvelope(t *testing.T) {
	row := func(name, duration, suffix string) string {
		return "\x1b[32;1m" + name + "\x1b[m [Created \x1b[35;1m" + duration + "\x1b[m ago] " + suffix + "\n"
	}
	for _, tc := range []struct {
		name, raw string
		bad       bool
	}{
		{"capture", lookupMissNativeListing, false},
		{"opaque", row("space [Created 5s ago] name", "1h 2m 3s", ""), false},
		{"current", row("a", "3s", "(current)"), false},
		{"exited", row("a", "3s", "(\x1b[31;1mEXITED\x1b[m - attach to resurrect)"), false},
		{"unsupported-name-color", strings.Replace(row("a", "3s", ""), "[32;1m", "[32m", 1), true},
		{"unsupported-reset", strings.Replace(row("a", "3s", ""), "[m", "[0m", 1), true},
		{"invalid-duration", row("a", "yesterday", ""), true},
		{"name-escape", row("a\x1b[0m", "3s", ""), true},
		{"duration-escape", row("a", "3s\x1b[0m", ""), true},
		{"suffix-escape", row("a", "3s", "\x1b[0m"), true},
		{"uncolored-exited-suffix", row("a", "3s", "(EXITED - attach to resurrect)"), true},
		{"missing-native-suffix-separator", strings.Replace(row("a", "3s", ""), " ago] ", " ago]", 1), true},
		{"duplicate", row("a", "3s", "") + row("a", "1m", ""), true},
		{"mixed-plain", row("a", "3s", "") + "b [Created 3s ago]\n", true},
		{"blank-line", row("a", "3s", "") + "\n", true},
		{"trailing-escape", row("a", "3s", "") + "\x1b[0m", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			states, err := parseColoredSessionListing([]byte(tc.raw))
			if (err != nil) != tc.bad {
				t.Fatalf("states=%v err=%v wantbad=%v", states, err, tc.bad)
			}
			if tc.name == "exited" && !states["a"] {
				t.Fatal("exited state lost")
			}
			if tc.name == "opaque" {
				if _, ok := states["space [Created 5s ago] name"]; !ok {
					t.Fatal("opaque name changed")
				}
			}
		})
	}
}

func TestSessionSocketPresentRefusesUnsafeAndNonSocketPaths(t *testing.T) {
	path := lookupMissSocketFixture(t, "a")
	if !sessionSocketPresent("a") {
		t.Fatal("live exact socket not observed")
	}
	for _, name := range []string{"", ".", "..", "../a", "a/../a", "a\x00", "a\n", "a\t", "missing"} {
		if sessionSocketPresent(name) {
			t.Fatalf("unsafe/missing name accepted: %q", name)
		}
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "regular"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, filepath.Join(filepath.Dir(path), "alias")); err != nil {
		t.Fatal(err)
	}
	if sessionSocketPresent("regular") || sessionSocketPresent("alias") {
		t.Fatal("non-socket Lstat grants recovery")
	}
	t.Setenv("ZELLIJ_SOCKET_DIR", "relative")
	if sessionSocketPresent("a") {
		t.Fatal("invalid root accepted")
	}
}

func TestZellijLookupMissDoesNotReplayCollectorFailure(t *testing.T) {
	calls := 0
	collectorErr := errors.New("collector failed")
	source := ZellijSource{queryRun: func(cmd *exec.Cmd) error {
		calls++
		_, _ = io.WriteString(cmd.Stdout, lookupMissNativeListing)
		_, _ = io.WriteString(cmd.Stderr, lookupMissNativeStderr)
		return collectorErr
	}}
	_, err := source.runContext(t.Context(), "--session", "p761-f0ab63ebb7b4af9a-1", "action", "list-clients")
	if calls != 1 || !errors.Is(err, collectorErr) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func FuzzParseColoredSessionListing(f *testing.F) {
	f.Add([]byte(lookupMissNativeListing))
	f.Add([]byte("\x1b[32;1ma\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n"))
	f.Add([]byte("\x1b[32;1ma\x1b[m [Created \x1b[35;1m3s\x1b[m ago] (\x1b[31;1mEXITED\x1b[m - attach to resurrect)\n"))
	f.Add([]byte("[{"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		states, err := parseColoredSessionListing(raw)
		if err != nil {
			return
		}
		if len(states) == 0 {
			t.Fatal("empty listing accepted")
		}
		for name := range states {
			if strings.ContainsRune(name, '\x1b') || strings.TrimSpace(name) == "" {
				t.Fatalf("invalid opaque name %q", name)
			}
		}
		if _, err := parseColoredSessionListing(append(append([]byte{}, raw...), 0x1b)); err == nil {
			t.Fatal("trailing escape accepted")
		}
	})
}

func FuzzRetryableSessionLookupMiss(f *testing.F) {
	f.Add([]byte(lookupMissNativeListing), []byte(lookupMissNativeStderr), 0, true)
	f.Add([]byte{}, []byte("There is no active session!\n"), 1, true)
	f.Fuzz(func(t *testing.T, out, stderr []byte, exit int, socket bool) {
		args := []string{"--session", "p761-f0ab63ebb7b4af9a-1", "action", "list-panes", "--json", "--command"}
		if !retryableSessionLookupMiss(args, out, stderr, exit, socket) {
			return
		}
		if exit == 1 {
			if !socket || len(out) != 0 || string(stderr) != "There is no active session!\n" {
				t.Fatal("non-native exit1 admitted")
			}
		} else if exit != 0 || string(stderr) != lookupMissNativeStderr {
			t.Fatal("non-native success signature admitted")
		}
		for _, bad := range [][]string{{"--session", args[1], "action", "close-pane"}, {"--session", args[1], "action", "list-panes", "--json", "--future"}, {"list-sessions", "--no-formatting"}} {
			if retryableSessionLookupMiss(bad, out, stderr, exit, socket) {
				t.Fatalf("unknown/mutation argv admitted: %q", bad)
			}
		}
	})
}

func TestZellijLookupMissRejectsIncompleteOrUnrelatedNativeSignature(t *testing.T) {
	args := []string{"--session", "pair-a", "action", "list-clients"}
	colored := "\x1b[32;1mpair-a\x1b[m [Created \x1b[35;1m3s\x1b[m ago] \n"
	message := "Session 'pair-a' not found. The following sessions are active:\n"
	for _, tc := range []struct {
		name, out, stderr string
		exit              int
	}{
		{"wrong-message", colored, "other warning\n", 0},
		{"wrong-target", colored, "Session 'other' not found. The following sessions are active:\n", 0},
		{"exited-target", "pair-a [Created 3s ago] (EXITED - attach to resurrect)\n", message, 0},
		{"absent-target", "other [Created 3s ago]\n", message, 0},
		{"plain-warning", "other [Created 3s ago]\n", "other warning\n", 0},
		{"bad-json", "[{", message, 0},
		{"collector-exit", "", "There is no active session!\n", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEmptyReplyFake(t, emptyReplyFrame{Args: args, Stdout: tc.out, Stderr: tc.stderr, Exit: tc.exit})
			_, _ = f.source.runContext(t.Context(), args...)
			if calls := f.calls(t); calls != 1 {
				t.Fatalf("nonmatching signature replayed %d times", calls)
			}
		})
	}
}

func TestZellijLookupMissAttemptWorkScalesLinearly(t *testing.T) {
	for _, n := range []int{8, 16} {
		calls := 0
		source := ZellijSource{queryRun: func(cmd *exec.Cmd) error {
			calls++
			if calls%3 == 0 {
				_, _ = io.WriteString(cmd.Stdout, "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n")
			} else {
				_, _ = io.WriteString(cmd.Stdout, lookupMissNativeListing)
				_, _ = io.WriteString(cmd.Stderr, lookupMissNativeStderr)
			}
			return nil
		}}
		for range n {
			if _, err := source.ClientCountContext(t.Context(), "p761-f0ab63ebb7b4af9a-1"); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 3*n {
			t.Fatalf("%d reads used %d attempts", n, calls)
		}
	}
}

func TestZellijLookupMissJoinedExitIsCollectorFailure(t *testing.T) {
	lookupMissSocketFixture(t, "pair-a")
	args := []string{"--session", "pair-a", "action", "list-clients"}
	f := newEmptyReplyFake(t, emptyReplyFrame{Args: args, Exit: 1, Stderr: "There is no active session!\n"})
	collectorErr := errors.New("query output collector failed")
	f.source.queryRun = func(cmd *exec.Cmd) error { return errors.Join(zellijexec.Run(cmd), collectorErr) }
	_, err := f.source.runContext(t.Context(), args...)
	if !errors.Is(err, collectorErr) || f.calls(t) != 1 {
		t.Fatalf("joined collector error retried: calls=%d err=%v", f.calls(t), err)
	}
}
