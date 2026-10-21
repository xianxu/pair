package agentfixture

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
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

func TestRequestCancellationAcknowledgesJoinedChild(t *testing.T) {
	config := startLifetimeFixture(t)
	marker := filepath.Join(config.Root, "child-pid")
	if err := WriteCommand(config.ControlDir, Command{ID: "active", Kind: Exec, Timeout: time.Hour, Args: []string{"-c", `printf '%s' "$$" > "$1"; exec sleep 30`, "fixture", marker}}); err != nil {
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
		if err != nil || len(entries) != 0 {
			t.Fatalf("terminal %s not reclaimed: entries=%d err=%v", sub, len(entries), err)
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
	if err := WriteCommand(config.ControlDir, Command{ID: "queued-expiry", Kind: Exec, Timeout: 50 * time.Millisecond, Args: []string{"-c", `printf wrong > "$1"`, "fixture", marker}}); err != nil {
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
			if err != nil || len(entries) != 0 {
				t.Fatalf("cycle%d leaked %s: %d %v", cycle, sub, len(entries), err)
			}
		}
	}
}

func TestCLIAdmissionTransition(t *testing.T) {
	now := time.Now()
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
