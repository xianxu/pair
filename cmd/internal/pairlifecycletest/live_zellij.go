package pairlifecycletest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
	fixture := &ControlledZellij{Session: session, command: command, pty: terminal}
	clientOutput := make(chan string, 1)
	go func() {
		var startup bytes.Buffer
		_, _ = io.Copy(&startup, io.LimitReader(terminal, 16384))
		_, _ = io.Copy(io.Discard, terminal)
		clientOutput <- startup.String()
	}()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		output, listErr := exec.CommandContext(ctx, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
		if listErr == nil && strings.Contains(string(output), session) {
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
			return nil, fmt.Errorf("controlled zellij session %q did not become ready: %w", session, ctx.Err())
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
