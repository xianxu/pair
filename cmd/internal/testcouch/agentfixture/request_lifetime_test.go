package agentfixture

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"
)

func startLifetimeFixture(t *testing.T) Config {
	t.Helper()
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard) }()
	t.Cleanup(func() {
		cancel()
		_ = writer.Close()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	waitFixture(t, func() bool { _, err := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return err == nil })
	return config
}

func TestUncertainResultCannotRetireCorrelatedFiles(t *testing.T) {
	c := fixtureConfig(t, "claude")
	if err := atomicJSON(filepath.Join(c.ControlDir, "ready.json"), Ready{PID: os.Getpid(), RunID: c.RunID}, false); err != nil {
		t.Fatal(err)
	}
	command := Command{ID: "uncertain", Kind: Exec, Args: []string{"--actors"}}
	if err := WriteCommand(c.ControlDir, command); err != nil {
		t.Fatal(err)
	}
	if err := CancelCommand(c.ControlDir, command.ID); err != nil {
		t.Fatal(err)
	}
	result := map[string]any{"id": command.ID, "exit_code": -1, "cleanup_uncertain": true, "error": "fixture CLI cleanup uncertain"}
	if err := atomicJSON(filepath.Join(c.ControlDir, "results", command.ID+".json"), result, true); err != nil {
		t.Fatal(err)
	}
	if err := RetireResult(c.ControlDir, command.ID); err == nil {
		t.Fatal("uncertain request was retired")
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		if _, err := os.Stat(filepath.Join(c.ControlDir, sub, command.ID+".json")); err != nil {
			t.Fatalf("uncertain %s evidence lost: %v", sub, err)
		}
	}
}

func TestRequestCancellationAcknowledgesJoinedChild(t *testing.T) {
	config := startLifetimeFixture(t)
	// The final result can be consumed before its publisher removes this name.
	if err := os.WriteFile(filepath.Join(config.ControlDir, "results", ".publish-held"), []byte("in progress"), 0600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(config.Root, "child-pid")
	if err := WriteCommand(config.ControlDir, Command{ID: "active", Kind: Exec, Timeout: time.Hour, Args: []string{"-c", fixtureLeaderPID, "fixture", marker}}); err != nil {
		t.Fatal(err)
	}
	waitFixture(t, func() bool {
		raw, err := os.ReadFile(marker)
		_, parse := strconv.Atoi(string(raw))
		return err == nil && parse == nil
	})
	raw, _ := os.ReadFile(marker)
	pid, _ := strconv.Atoi(string(raw))
	if err := CancelCommand(config.ControlDir, "active"); err != nil {
		t.Fatal(err)
	}
	var result Result
	waitFixture(t, func() bool { var err error; result, err = ReadResult(config.ControlDir, "active"); return err == nil })
	if result.Error == "" || result.ExitCode == 0 {
		t.Fatalf("cancel was acknowledged as success: %+v", result)
	}
	if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
		t.Fatalf("ack preceded child join: pid=%d err=%v", pid, err)
	}
	if err := RetireResult(config.ControlDir, "active"); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		entries, err := os.ReadDir(filepath.Join(config.ControlDir, sub))
		if err != nil || countMailboxEntries(entries) != 0 {
			t.Fatalf("terminal %s not reclaimed: entries=%v err=%v", sub, entries, err)
		}
	}
}

func TestQueuedRequestCancellationSuppressesSpawn(t *testing.T) {
	config := startLifetimeFixture(t)
	for _, id := range []string{"hold-0", "hold-1", "hold-2", "hold-3"} {
		marker := filepath.Join(config.Root, id)
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: Exec, Args: []string{"-c", `printf started > "$1"; exec sleep 30`, "fixture", marker}}); err != nil {
			t.Fatal(err)
		}
		waitFixture(t, func() bool { _, err := os.Stat(marker); return err == nil })
	}
	marker := filepath.Join(config.Root, "must-not-spawn")
	if err := WriteCommand(config.ControlDir, Command{ID: "queued", Kind: Exec, Args: []string{"-c", `printf wrong > "$1"`, "fixture", marker}}); err != nil {
		t.Fatal(err)
	}
	if err := CancelCommand(config.ControlDir, "queued"); err != nil {
		t.Fatal(err)
	}
	var result Result
	waitFixture(t, func() bool { var err error; result, err = ReadResult(config.ControlDir, "queued"); return err == nil })
	if result.Error == "" {
		t.Fatalf("queued cancellation succeeded: %+v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("canceled queued request executed: %v", err)
	}
}

func TestExpiredAdmissionDeadlineNeverStartsChild(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "must-not-spawn")
	raw, _ := json.Marshal(map[string]any{"id": "expired", "kind": "exec", "timeout_ns": int64(time.Hour), "deadline_unix_nano": time.Now().Add(-time.Second).UnixNano(), "args": []string{"-c", `printf wrong > "$1"`, "fixture", marker}})
	command, err := DecodeCommand(raw)
	if err != nil {
		t.Fatalf("absolute admission deadline not supported: %v", err)
	}
	result := runCLI(t.Context(), "/bin/sh", command)
	if !strings.Contains(result.Error, "deadline") {
		t.Fatalf("expired admission did not fail as deadline: %+v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expired request executed: %v", err)
	}
}

func TestQueuedRequestDeadlineIncludesAdmissionWait(t *testing.T) {
	config := startLifetimeFixture(t)
	for _, id := range []string{"hold-0", "hold-1", "hold-2", "hold-3"} {
		marker := filepath.Join(config.Root, id)
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: Exec, Timeout: time.Hour, Args: []string{"-c", `printf started > "$1"; exec sleep 30`, "fixture", marker}}); err != nil {
			t.Fatal(err)
		}
		waitFixture(t, func() bool { _, err := os.Stat(marker); return err == nil })
	}
	marker := filepath.Join(config.Root, "expired-queued-spawn")
	if err := WriteCommand(config.ControlDir, Command{ID: "queued-expiry", Kind: Exec, DeadlineUnixNano: time.Now().Add(-time.Hour).UnixNano(), Args: []string{"-c", `printf wrong > "$1"`, "fixture", marker}}); err != nil {
		t.Fatal(err)
	}
	var result Result
	waitFixture(t, func() bool {
		var err error
		result, err = ReadResult(config.ControlDir, "queued-expiry")
		return err == nil
	})
	if !strings.Contains(result.Error, "deadline") {
		t.Fatalf("queued time not charged to deadline: %+v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expired queued request spawned: %v", err)
	}
}

func TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation(t *testing.T) {
	config := startLifetimeFixture(t)
	// Model the valid interval between linking a result and removing scratch.
	if err := os.WriteFile(filepath.Join(config.ControlDir, "results", ".publish-held"), []byte("in progress"), 0600); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < MaxRequests+1; cycle++ {
		command := Command{ID: "reused", Kind: Exec, Args: []string{"-c", "printf done"}}
		if err := WriteCommand(config.ControlDir, command); err != nil {
			t.Fatal(err)
		}
		if cycle%2 == 0 {
			if err := CancelCommand(config.ControlDir, command.ID); err != nil {
				t.Fatal(err)
			}
		}
		var result Result
		waitFixture(t, func() bool { var err error; result, err = ReadResult(config.ControlDir, command.ID); return err == nil })
		if cycle%2 != 0 && (result.Error != "" || result.Stdout != "done") {
			t.Fatalf("cycle%d inherited prior cancellation: %+v", cycle, result)
		}
		if err := RetireResult(config.ControlDir, command.ID); err != nil {
			t.Fatal(err)
		}
		for _, sub := range []string{"commands", "results", "cancellations"} {
			entries, err := os.ReadDir(filepath.Join(config.ControlDir, sub))
			if err != nil || countMailboxEntries(entries) != 0 {
				t.Fatalf("cycle%d leaked %s: entries=%v err=%v", cycle, sub, entries, err)
			}
		}
	}
}

// Excluding publication scratch must never conceal a real result or marker.
func TestRetiredMailboxCountsRealRecords(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  int
	}{
		{"scratch only", []string{".publish-held"}, 0},
		{"result remains", []string{".publish-held", "reused.json"}, 1},
		{"two results remain", []string{".publish-held", "first.json", "second.json"}, 2},
		{"unexpected entry remains", []string{".publish-held", "unexpected"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mailbox := fstest.MapFS{}
			for _, name := range tc.files {
				mailbox[name] = &fstest.MapFile{}
			}
			entries, err := fs.ReadDir(mailbox, ".")
			if err != nil {
				t.Fatal(err)
			}
			if got := countMailboxEntries(entries); got != tc.want {
				t.Fatalf("mailbox records=%d want=%d, entries=%v", got, tc.want, entries)
			}
		})
	}
}

func TestCLIAdmissionTransition(t *testing.T) {
	now := time.Unix(1700000000, 0)
	for _, test := range []struct {
		name     string
		canceled bool
		deadline time.Time
		active   int
		start    bool
		failure  error
	}{
		{name: "queued", deadline: now.Add(time.Hour), active: MaxConcurrentCLI},
		{name: "running", deadline: now.Add(time.Hour), active: MaxConcurrentCLI - 1, start: true},
		{name: "canceled-terminal-at-capacity", canceled: true, deadline: now.Add(time.Hour), active: MaxConcurrentCLI, failure: context.Canceled},
		{name: "expired-terminal-at-capacity", deadline: now, active: MaxConcurrentCLI, failure: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			start, err := decideCLIAdmission(Command{DeadlineUnixNano: test.deadline.UnixNano()}, test.canceled, test.active, now)
			if start != test.start || !errors.Is(err, test.failure) {
				t.Fatalf("start=%v failure=%v", start, err)
			}
		})
	}
}

func TestCLIAdmissionQueuedDeadlineRetainsItsAbsoluteBoundary(t *testing.T) {
	now := time.Unix(1700000000, 0)
	deadline := now.Add(time.Minute)
	command := Command{DeadlineUnixNano: deadline.UnixNano()}
	start, err := decideCLIAdmission(command, false, MaxConcurrentCLI, now)
	if start || err != nil {
		t.Fatalf("initial queue: start=%v err=%v", start, err)
	}
	start, err = decideCLIAdmission(command, false, MaxConcurrentCLI, deadline)
	if start || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("same queued deadline: start=%v err=%v", start, err)
	}
}
