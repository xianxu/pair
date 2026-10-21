package pairlifecycletest

import (
	"context"
	"errors"
	"github.com/creack/pty"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type closeProcessReader struct {
	table   func() ([]procutil.Process, error)
	command procutil.CommandInfo
}

func (r closeProcessReader) Table() ([]procutil.Process, error) { return r.table() }
func (r closeProcessReader) Command(int) (procutil.CommandInfo, error) {
	return r.command, nil
}
func (r closeProcessReader) Identity(int) (string, error) { return "birth", nil }

func closeZellijStub(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestControlledZellijCloseWaitsForCacheWriter(t *testing.T) {
	closeZellijStub(t, "exit 0") // Shutdown request completes before the daemon.
	cache := t.TempDir()
	observed, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseWriter := func() { releaseOnce.Do(func() { close(release) }) }
	server := procutil.Process{PID: 42, Identity: "birth", Name: "zellij"}
	calls := 0
	wrote := false
	reader := closeProcessReader{
		command: procutil.CommandInfo{Executable: "/fixture/zellij", Args: []string{"zellij", "--server", "/fixture/owned-test"}},
		table: func() ([]procutil.Process, error) {
			calls++
			if calls == 1 { // Capture the live generation before requesting shutdown.
				return []procutil.Process{server}, nil
			}
			close(observed)
			<-release
			// The daemon's last write precedes its terminal kernel state.
			if err := os.WriteFile(filepath.Join(cache, "late-entry"), nil, 0600); err != nil {
				return nil, err
			}
			wrote = true
			server.Zombie = true
			return []procutil.Process{server}, nil
		},
	}
	fixture := &ControlledZellij{Session: "owned-test", stubDir: cache, processes: procutil.NewSource(reader)}
	done := make(chan error, 1)
	joined := make(chan struct{})
	go func() { defer close(joined); done <- fixture.Close() }()
	t.Cleanup(func() {
		releaseWriter()
		select {
		case <-joined:
		case <-testwait.After(t):
			t.Error("Close worker did not finish during cleanup")
		}
	})
	select {
	case <-observed:
	case err := <-done:
		t.Fatalf("Close returned before observing the cache writer's exit: %v", err)
	case <-testwait.After(t):
		t.Fatal("Close never observed the cache writer")
	}
	select {
	case err := <-done:
		t.Fatalf("Close returned with the writer held: %v", err)
	default:
	}
	releaseWriter()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-testwait.After(t):
		t.Fatal("Close did not join the released cache writer")
	}
	if !wrote {
		t.Fatal("fixture did not exercise the late cache write")
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("owned files remained after writer exit: %v", err)
	}
}

func TestControlledZellijShutdownEvidence(t *testing.T) {
	probeErr := errors.New("kernel observation failed")
	for _, tc := range []struct {
		name string
		rows []procutil.Process
		err  error
		want error
	}{
		{name: "absent"},
		{name: "zombie", rows: []procutil.Process{{PID: 42, Identity: "birth", Zombie: true}}},
		{name: "PID reused", rows: []procutil.Process{{PID: 42, Identity: "new-birth"}}},
		{name: "unrelated live server", rows: []procutil.Process{{PID: 43, Identity: "birth"}}},
		{name: "still running", rows: []procutil.Process{{PID: 42, Identity: "birth"}}, want: context.Canceled},
		{name: "identity unknown", rows: []procutil.Process{{PID: 42}}, want: context.Canceled},
		{name: "observation failure", err: probeErr, want: probeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := closeProcessReader{table: func() ([]procutil.Process, error) { return tc.rows, tc.err }}
			fixture := &ControlledZellij{processes: procutil.NewSource(reader)}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := fixture.waitServers(ctx, map[int]string{42: "birth"}); !errors.Is(err, tc.want) {
				t.Fatalf("shutdown evidence = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestControlledZellijCloseReportsShutdownErrors(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		wantError    bool
	}{
		{"missing resurrection cache", `printf 'Session: "owned-test" not found.\n' >&2; exit 2`, false},
		{"active session missing cache", `printf 'A session by the name "owned-test" exists and is active, but will be force killed and deleted.\nSession: "owned-test" not found.\n' >&2; exit 2`, false},
		{"unexpected diagnostic with missing cache", `printf 'unexpected failure\nSession: "owned-test" not found.\n' >&2; exit 2`, true},
		{"other exit 2", `printf 'invalid argument\n' >&2; exit 2`, true},
		{"another missing session", `printf 'Session: "other" not found.\n' >&2; exit 2`, true},
		{"other error", `printf 'failed\n' >&2; exit 1`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			closeZellijStub(t, tc.script)
			reader := closeProcessReader{table: func() ([]procutil.Process, error) { return nil, nil }}
			fixture := &ControlledZellij{Session: "owned-test", processes: procutil.NewSource(reader)}
			err := fixture.Close()
			if (err != nil) != tc.wantError {
				t.Fatalf("Close = %v, want error=%v", err, tc.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), "delete controlled zellij") {
				t.Fatalf("lost shutdown error context: %v", err)
			}
		})
	}
}

func TestControlledZellijCloseSelectsExactServer(t *testing.T) {
	closeZellijStub(t, "exit 0")
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"owned", []string{"zellij", "--server", "/fixture/owned-test"}, 2},
		{"another session", []string{"zellij", "--server", "/fixture/owned-test-extra"}, 1},
		{"client", []string{"zellij", "--session", "owned-test"}, 1},
		{"extra arguments", []string{"zellij", "--server", "/fixture/owned-test", "other"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			reader := closeProcessReader{
				command: procutil.CommandInfo{Executable: "/fixture/zellij", Args: tc.args},
				table: func() ([]procutil.Process, error) {
					calls++
					if calls == 1 {
						return []procutil.Process{{PID: 42, Identity: "birth", Name: "zellij"}}, nil
					}
					return nil, nil
				},
			}
			fixture := &ControlledZellij{Session: "owned-test", processes: procutil.NewSource(reader)}
			if err := fixture.Close(); err != nil {
				t.Fatal(err)
			}
			if calls != tc.want {
				t.Fatalf("process-table reads = %d, want %d", calls, tc.want)
			}
		})
	}
}

func TestControlledZellijCloseRetainsCaptureFailure(t *testing.T) {
	closeZellijStub(t, "exit 0")
	probeErr := errors.New("process census failed")
	reader := closeProcessReader{table: func() ([]procutil.Process, error) { return nil, probeErr }}
	cache := t.TempDir()
	fixture := &ControlledZellij{Session: "owned-test", stubDir: cache, processes: procutil.NewSource(reader)}
	if err := fixture.Close(); !errors.Is(err, probeErr) {
		t.Fatalf("Close = %v, want the failed census", err)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("removed owned files without server-exit evidence: %v", err)
	}
}

// Linux enumeration represents both failed stat reads and malformed stat data
// as a listed PID with unknown metadata. Both consumers must preserve it.
func TestControlledZellijCloseRetainsUnknownProcess(t *testing.T) {
	for _, phase := range []string{"capture", "shutdown poll"} {
		t.Run(phase, func(t *testing.T) {
			closeZellijStub(t, "exit 0")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			reader := closeProcessReader{
				command: procutil.CommandInfo{Args: []string{"zellij", "--server", "/fixture/owned-test"}},
				table: func() ([]procutil.Process, error) {
					calls++
					if phase == "shutdown poll" && calls == 1 {
						return []procutil.Process{{PID: 42, Identity: "birth", Name: "zellij"}}, nil
					}
					if phase == "shutdown poll" {
						cancel() // A bounded poll must fail rather than infer exit.
					}
					return []procutil.Process{{PID: 42}}, nil
				},
			}
			cache := t.TempDir()
			fixture := &ControlledZellij{Session: "owned-test", stubDir: cache, processes: procutil.NewSource(reader)}
			if err := fixture.close(ctx); err == nil {
				t.Fatal("unknown process authorized successful cleanup")
			} else if phase == "shutdown poll" && !errors.Is(err, context.Canceled) {
				t.Fatalf("poll lost cancellation: %v", err)
			}
			if _, err := os.Stat(cache); err != nil {
				t.Fatalf("removed owned files without exit evidence: %v", err)
			}
			want := 1
			if phase == "shutdown poll" {
				want = 2
			}
			if calls != want {
				t.Fatalf("table reads=%d, want %d", calls, want)
			}
		})
	}
}

func TestControlledZellijKillClientThenCloseRetiresHandlesOnce(t *testing.T) {
	closeZellijStub(t, "exit 0")
	command := exec.Command("sleep", "30")
	terminal, err := pty.Start(command)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	reader := closeProcessReader{
		command: procutil.CommandInfo{Executable: "/fixture/zellij", Args: []string{"zellij", "--server", "/fixture/owned-test"}},
		table: func() ([]procutil.Process, error) {
			calls++
			if calls == 1 {
				return []procutil.Process{{PID: 42, Identity: "birth", Name: "zellij"}}, nil
			}
			return nil, nil
		},
	}
	cache := t.TempDir()
	fixture := &ControlledZellij{Session: "owned-test", command: command, pty: terminal, stubDir: cache, processes: procutil.NewSource(reader)}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := fixture.KillClient(); err != nil {
		t.Fatal(err)
	}
	retired := fixture.command == nil && fixture.pty == nil
	if err := command.Process.Signal(os.Interrupt); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("client not reaped: %v", err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatalf("KillClient then Close: %v", err)
	}
	if !retired {
		t.Fatal("KillClient retained consumed handles")
	}
	if calls != 2 {
		t.Fatalf("exact server capture/exit proof reads=%d", calls)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("owned cache survived exact cleanup: %v", err)
	}
	if _, err := fixture.WriteInput([]byte("x")); err == nil || errors.Is(err, os.ErrClosed) {
		t.Fatalf("retired input handle: %v", err)
	}
}

func TestControlledZellijKillClientPreservesFirstCloseError(t *testing.T) {
	command := exec.Command("sleep", "30")
	terminal, err := pty.Start(command)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &ControlledZellij{command: command, pty: terminal}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := terminal.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fixture.KillClient(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("lost first owned close error: %v", err)
	}
	if fixture.command != nil || fixture.pty != nil {
		t.Fatal("cleanup error prevented handle retirement")
	}
	if err := fixture.Close(); err != nil {
		t.Fatalf("retired handles closed twice: %v", err)
	}
}
