package pairlifecycletest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// ControlledZellij is the portable live fixture shared by lifecycle
// conformance tests. It owns one throwaway session and its pty client.
type ControlledZellij struct {
	Session string
	command *exec.Cmd
	pty     *os.File
	startup *zellijStartupCapture
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
			_ = fixture.Close()
			return nil, fmt.Errorf("controlled zellij client exited before initial screen render: %s", output)
		case <-ctx.Done():
			_ = fixture.Close()
			return nil, fmt.Errorf("controlled zellij session %q did not emit initial screen render: %w", session, ctx.Err())
		case <-ticker.C:
		}
	}
	for {
		output, listErr := exec.CommandContext(ctx, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
		if listErr == nil && hasLiveSessionRow(string(output), session) {
			if options.LayoutFile != "" {
				out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "new-tab", "--layout", options.LayoutFile).CombinedOutput()
				if err != nil {
					_ = fixture.Close()
					return nil, fmt.Errorf("controlled zellij layout: %s: %w", out, err)
				}
			}
			return fixture, nil
		}
		select {
		case output := <-clientOutput:
			_ = fixture.Close()
			return nil, fmt.Errorf("controlled zellij client exited before readiness: %s", output)
		case <-ctx.Done():
			_ = fixture.Close()
			return nil, fmt.Errorf("controlled zellij session %q did not become ready: %w; last list=%q; client startup=%q", session, ctx.Err(), output, capture.String())
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
	if f.pty != nil {
		_ = f.pty.Close()
	}
	if err := f.command.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	_, err := f.command.Process.Wait()
	return err
}

func (f *ControlledZellij) Close() error {
	if f == nil {
		return nil
	}
	var result error
	if f.Session != "" {
		result = errors.Join(result, exec.Command("zellij", "delete-session", f.Session, "--force").Run())
	}
	if f.pty != nil {
		result = errors.Join(result, f.pty.Close())
		f.pty = nil
	}
	if f.command != nil && f.command.Process != nil {
		_ = f.command.Process.Kill()
		_, _ = f.command.Process.Wait()
		f.command = nil
	}
	return result
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
