package pairlifecycletest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// ControlledZellij is the portable live fixture shared by lifecycle
// conformance tests. It owns one throwaway session and its pty client.
type ControlledZellij struct {
	Session   string
	command   *exec.Cmd
	pty       *os.File
	startup   *zellijStartupCapture
	stubDir   string
	processes procutil.Source
}

// ZellijOptions selects fixture configuration without changing the operator's config.
type ZellijOptions struct {
	ConfigFile string
	LayoutFile string
}

func StartControlledZellij(ctx context.Context, session string) (*ControlledZellij, error) {
	return StartControlledZellijWithOptions(ctx, session, ZellijOptions{})
}

func StartControlledZellijWithOptions(ctx context.Context, session string, options ZellijOptions) (*ControlledZellij, error) {
	if session == "" {
		return nil, errors.New("controlled zellij session is empty")
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		return nil, fmt.Errorf("zellij is required: %w", err)
	}
	_ = exec.Command("zellij", "delete-session", session, "--force").Run()
	var args []string
	if options.ConfigFile != "" {
		args = append(args, "--config", options.ConfigFile)
	}
	args = append(args, "--session", session)

	command := exec.Command("zellij", args...)
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		return nil, err
	}
	capture := &zellijStartupCapture{}
	fixture := &ControlledZellij{Session: session, command: command, pty: terminal, startup: capture}
	clientOutput := make(chan string, 1)
	go func() {
		_, _ = io.Copy(capture, terminal)
		clientOutput <- capture.String()
	}()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	// Zellij's list-sessions connects to every server socket. Before the first
	// client initializes session data, closing that probe can crash Zellij's
	// RemoveClient handler. Wait for a server-rendered cursor-position/reset
	// instruction, rather than the client's earlier terminal setup bytes.
	for !zellijScreenRender.MatchString(capture.String()) {
		select {
		case output := <-clientOutput:
			return nil, errors.Join(fmt.Errorf("controlled zellij client exited before initial screen render: %s", output), fixture.Close())
		case <-ctx.Done():
			return nil, errors.Join(fmt.Errorf("controlled zellij session %q did not emit initial screen render: %w", session, ctx.Err()), fixture.Close())
		case <-ticker.C:
		}
	}
	for {
		// This controlled conformance input intentionally has no Pair launch marker.
		// The observed server render above is this fixture's prerequisite; use the
		// approved explicit fixture observation while retaining namespace exclusion.
		output, listErr := (zellijquery.Source{ExplicitRecovery: true}).Output(ctx, "list-sessions", "--no-formatting")
		if listErr == nil && hasLiveSessionRow(string(output), session) {
			if options.LayoutFile != "" {
				out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "new-tab", "--layout", options.LayoutFile).CombinedOutput()
				if err != nil {
					return nil, errors.Join(fmt.Errorf("controlled zellij layout: %s: %w", out, err), fixture.Close())
				}
			}
			return fixture, nil
		}
		select {
		case output := <-clientOutput:
			return nil, errors.Join(fmt.Errorf("controlled zellij client exited before readiness: %s", output), fixture.Close())
		case <-ctx.Done():
			return nil, errors.Join(fmt.Errorf("controlled zellij session %q did not become ready: %w; last list=%q; client startup=%q", session, ctx.Err(), output, capture.String()), fixture.Close())
		case <-ticker.C:
		}
	}
}

// KillClient terminates the attached zellij CLIENT and leaves the server
// session running -- the exact shape Couch's detach produces.
//
// It closes the pty first, because a client blocked writing to a terminal
// nobody reads can outlive its SIGTERM.
func (f *ControlledZellij) KillClient() error {
	if f == nil || f.command == nil || f.command.Process == nil {
		return errors.New("controlled zellij has no client process")
	}
	return f.stopClient(syscall.SIGTERM)
}

// Consume owned handles before acting on them: detach and final cleanup must
// never close or reap the same handle twice, even when the first action fails.
func (f *ControlledZellij) stopClient(signal os.Signal) error {
	var result error
	terminal := f.pty
	f.pty = nil
	if terminal != nil {
		result = errors.Join(result, terminal.Close())
	}
	command := f.command
	f.command = nil
	if command != nil && command.Process != nil {
		if err := command.Process.Signal(signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
			result = errors.Join(result, err)
		}
		_, err := command.Process.Wait()
		result = errors.Join(result, err)
	}
	return result
}

func (f *ControlledZellij) Close() error {
	if f == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return f.close(ctx)
}

func (f *ControlledZellij) close(ctx context.Context) error {
	var result error
	var observationErr error
	servers := map[int]string{}
	if f.Session != "" {
		rows, err := f.processes.SnapshotNamed(ctx, "zellij")
		observationErr = err
		for _, row := range rows {
			if row.Err != nil {
				observationErr = errors.Join(observationErr, row.Err)
			} else if len(row.Args) == 3 && row.Args[1] == "--server" && filepath.Base(row.Args[2]) == f.Session {
				servers[row.PID] = row.Identity
			}
		}
		out, err := exec.CommandContext(ctx, "zellij", "delete-session", f.Session, "--force").CombinedOutput()
		// With serialization disabled, native Zellij kills the session but
		// returns 2 if its resurrection directory does not exist. This says
		// nothing about server exit: verify that separately below.
		var exit *exec.ExitError
		missing := fmt.Sprintf("Session: %q not found.", f.Session)
		active := fmt.Sprintf("A session by the name %q exists and is active, but will be force killed and deleted.", f.Session)
		diagnostic := strings.TrimSpace(string(out))
		missingCache := errors.As(err, &exit) && exit.ExitCode() == 2 && (diagnostic == missing || diagnostic == active+"\n"+missing)
		if err != nil && !missingCache {
			result = errors.Join(result, fmt.Errorf("delete controlled zellij: %w: %s", err, out))
		}
	}
	result = errors.Join(result, f.stopClient(syscall.SIGKILL))
	// The daemon is reparented to PID 1: reaping the client cannot join it.
	// Keep its exact kernel generation until it can no longer write caches.
	waitErr := f.waitServers(ctx, servers)
	result = errors.Join(result, observationErr, waitErr)
	if observationErr == nil && waitErr == nil && f.stubDir != "" {
		result = errors.Join(result, os.RemoveAll(f.stubDir))
		f.stubDir = ""
	}
	return result
}

func (f *ControlledZellij) waitServers(ctx context.Context, servers map[int]string) error {
	if len(servers) == 0 {
		return nil
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		rows, err := f.processes.Table()
		if err != nil {
			return fmt.Errorf("observe controlled zellij shutdown: %w", err)
		}
		running := false
		for _, row := range rows {
			identity, owned := servers[row.PID]
			if owned && !row.Zombie && (row.Identity == identity || row.Identity == "") {
				running = true
			}
		}
		if !running {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("controlled zellij server did not exit: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// AddOwnerWrapperPane opens a pane whose live process is `pair wrap
// --scrollback-log <raw>`. Ownership probes accept only that live-command
// evidence (launcher.ClassifySessionOwner, #355), and a bare fixture session
// holds nothing but a shell. Zellij reports the pane's running argv, so the
// stub must stay that process: `pair` is /bin/sh, which reads the script file
// `wrap` from the pane's cwd and keeps the argv intact while it sleeps.
func (f *ControlledZellij) AddOwnerWrapperPane(ctx context.Context, rawPath string) error {
	if f == nil || f.Session == "" {
		return errors.New("controlled zellij has no session")
	}
	if f.stubDir == "" {
		dir, err := os.MkdirTemp("", "pair-owner-stub-")
		if err != nil {
			return err
		}
		// No exec: replacing the shell would replace the argv the probe reads.
		if err := errors.Join(os.Symlink("/bin/sh", filepath.Join(dir, "pair")),
			os.WriteFile(filepath.Join(dir, "wrap"), []byte("sleep 3600\n"), 0o600)); err != nil {
			_ = os.RemoveAll(dir)
			return err
		}
		f.stubDir = dir
	}
	out, err := exec.CommandContext(ctx, "zellij", "--session", f.Session, "action", "new-pane", "--cwd", f.stubDir, "--",
		filepath.Join(f.stubDir, "pair"), "wrap", "--scrollback-log", rawPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("controlled zellij owner pane: %s: %w", out, err)
	}
	return nil
}

// WriteInput sends terminal input through the real attached client, so Zellij's
// keybindings run. The action write/write-chars CLI would bypass that boundary.
func (f *ControlledZellij) WriteInput(p []byte) (int, error) {
	if f == nil || f.pty == nil {
		return 0, errors.New("controlled zellij has no attached terminal")
	}
	return f.pty.Write(p)
}

// Bounded startup capture remains readable while the PTY drain is running.
type zellijStartupCapture struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (c *zellijStartupCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(p)
	remaining := 16384 - c.data.Len()
	if len(p) > remaining {
		p = p[:remaining]
	}
	_, _ = c.data.Write(p)
	return n, nil
}
func (c *zellijStartupCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data.String()
}
func (f *ControlledZellij) StartupOutput() string { return f.startup.String() }

func hasLiveSessionRow(output, session string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == session && fields[1] == "[Created" && !strings.Contains(line, "EXITED") {
			return true
		}
	}
	return false
}

var zellijScreenRender = regexp.MustCompile(`\x1b\[[0-9]+;[0-9]+H\x1b\[m`)
