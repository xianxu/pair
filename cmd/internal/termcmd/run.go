// Package termcmd provides the right-side user terminal wrapper for Pair's
// workbench layout.
package termcmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"golang.org/x/term"
)

type Runtime interface {
	ListPanesJSON() ([]byte, error)
	LastLeftPaneID() (string, error)
	RecordLastLeftPaneID(string) error
	RunZellijAction(args ...string) error
	ShellCommand() (string, []string)
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunWithRuntime(args, stdin, stdout, stderr, OSRuntime{})
}

func RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	fs := flag.NewFlagSet("term", flag.ContinueOnError)
	fs.SetOutput(stderr)
	testShortcut := fs.String("test-shortcut", "", "exercise a workbench shortcut without starting a shell")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: pair term [--test-shortcut CHORD]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *testShortcut != "" {
		chord, ok := namedChord(*testShortcut)
		if !ok {
			fmt.Fprintf(stderr, "term: unknown shortcut %q\n", *testShortcut)
			return 2
		}
		if err := handleChord(chord, rt, stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "term: %v\n", err)
			return 1
		}
		return 0
	}
	return runShell(stdin, stdout, stderr, rt)
}

func namedChord(name string) (workbenchshortcut.Chord, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alt+j":
		return workbenchshortcut.ChordAltJ, true
	case "alt+k":
		return workbenchshortcut.ChordAltK, true
	case "alt+t":
		return workbenchshortcut.ChordAltT, true
	case "alt+w":
		return workbenchshortcut.ChordAltW, true
	case "alt+r":
		return workbenchshortcut.ChordAltR, true
	case "alt+/":
		return workbenchshortcut.ChordAltSlash, true
	case "alt+shift+c":
		return workbenchshortcut.ChordAltShiftC, true
	case "ctrl+alt+c":
		return workbenchshortcut.ChordCtrlAltC, true
	default:
		return workbenchshortcut.ChordUnknown, false
	}
}

func handleChord(chord workbenchshortcut.Chord, rt Runtime, stdin io.Reader, stdout io.Writer) error {
	panes, err := focusedWorkbenchPanes(rt)
	if err != nil {
		return err
	}
	lastLeft, err := rt.LastLeftPaneID()
	if err != nil {
		return err
	}
	decision := workbenchshortcut.Decide(workbenchshortcut.ShortcutInput{
		Role:           workbenchshortcut.RoleForPane(panes.focused),
		Chord:          chord,
		FocusedPaneID:  panes.focused.ID,
		LastLeftPaneID: lastLeft,
		DraftPaneID:    panes.draft.ID,
	})
	return runDecision(decision, panes, rt, stdin, stdout)
}

type workbenchPanes struct {
	focused  zellijpane.Pane
	draft    zellijpane.Pane
	terminal zellijpane.Pane
}

func focusedWorkbenchPanes(rt Runtime) (workbenchPanes, error) {
	data, err := rt.ListPanesJSON()
	if err != nil {
		return workbenchPanes{}, err
	}
	var out workbenchPanes
	for _, pane := range zellijpane.Parse(data) {
		if pane.IsFocused {
			out.focused = pane
		}
		switch workbenchshortcut.RoleForPane(pane) {
		case workbenchshortcut.PaneRoleLeftDraft:
			out.draft = pane
		case workbenchshortcut.PaneRoleRightTerminal:
			out.terminal = pane
		}
	}
	if out.focused.ID == "" {
		return workbenchPanes{}, fmt.Errorf("no focused zellij pane found")
	}
	return out, nil
}

func runDecision(decision workbenchshortcut.ShortcutDecision, panes workbenchPanes, rt Runtime, stdin io.Reader, stdout io.Writer) error {
	if decision.Disposition != workbenchshortcut.DispositionHandle {
		return nil
	}
	if decision.RecordLastLeftPaneID != "" {
		if err := rt.RecordLastLeftPaneID(decision.RecordLastLeftPaneID); err != nil {
			return err
		}
	}
	switch decision.Action {
	case workbenchshortcut.ActionFocusPane:
		if decision.TargetPaneID == "" {
			return nil
		}
		return rt.RunZellijAction("focus-pane-id", decision.TargetPaneID)
	case workbenchshortcut.ActionFocusRightTerminal:
		if panes.terminal.ID == "" {
			return nil
		}
		return rt.RunZellijAction("focus-pane-id", panes.terminal.ID)
	default:
		return nil
	}
}

func runShell(stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	name, args := rt.ShellCommand()
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(stderr, "term: pty.Start: %v\n", err)
		return 1
	}
	defer func() { _ = ptmx.Close() }()

	stdinFile, _ := stdin.(*os.File)
	var oldState *term.State
	if stdinFile != nil {
		s, err := term.MakeRaw(int(stdinFile.Fd()))
		if err != nil {
			fmt.Fprintf(stderr, "term: MakeRaw: %v\n", err)
			return 1
		}
		oldState = s
		defer func() { _ = term.Restore(int(stdinFile.Fd()), oldState) }()

		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		defer signal.Stop(winch)
		go func() {
			for range winch {
				_ = pty.InheritSize(stdinFile, ptmx)
			}
		}()
		winch <- syscall.SIGWINCH
	}

	go pumpStdin(stdin, ptmx, rt, stdout)
	_, _ = io.Copy(stdout, ptmx)

	if stdinFile != nil && oldState != nil {
		_ = term.Restore(int(stdinFile.Fd()), oldState)
	}
	werr := cmd.Wait()
	if exitErr, ok := werr.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	if werr != nil {
		fmt.Fprintf(stderr, "term: cmd.Wait: %v\n", werr)
		return 1
	}
	return 0
}

func pumpStdin(stdin io.Reader, ptmx *os.File, rt Runtime, stdout io.Writer) {
	buf := make([]byte, 4096)
	var held []byte
	for {
		n, err := stdin.Read(buf)
		if n > 0 {
			data := append(held, buf[:n]...)
			held = nil
			if len(data) == 1 && data[0] == 0x1b {
				held = append(held, data...)
			} else if chord, ok := workbenchshortcut.DecodeChord(data); ok {
				_ = handleChord(chord, rt, stdin, stdout)
			} else {
				_, _ = ptmx.Write(data)
			}
		}
		if err != nil {
			if len(held) > 0 {
				_, _ = ptmx.Write(held)
			}
			return
		}
	}
}

type OSRuntime struct{}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state").Output()
}

func (OSRuntime) LastLeftPaneID() (string, error) {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: pairDataDir(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) RecordLastLeftPaneID(id string) error {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: pairDataDir(), Tag: os.Getenv("PAIR_TAG")}
	return store.Write(id)
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return runZellij(cmdArgs...)
}

func runZellij(args ...string) error {
	cmd := exec.Command("zellij", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (OSRuntime) ShellCommand() (string, []string) {
	pairHome := os.Getenv("PAIR_HOME")
	tag := os.Getenv("PAIR_TAG")
	if pairHome != "" && tag != "" && os.Getenv("PAIR_TERM_INNER_ZELLIJ") != "0" {
		return innerZellijCommand(pairHome, tag)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-i"}
}

func innerZellijCommand(pairHome, tag string) (string, []string) {
	return "env", []string{
		"-u", "ZELLIJ",
		"-u", "ZELLIJ_SESSION_NAME",
		"zellij",
		"--config-dir", filepath.Join(pairHome, "zellij", "terminal"),
		"attach", "--create", "pair-" + tag + "-terminal",
	}
}

func pairDataDir() string {
	if v := os.Getenv("PAIR_DATA_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v + "/pair"
	}
	if v := os.Getenv("HOME"); v != "" {
		return v + "/.local/share/pair"
	}
	return "."
}
