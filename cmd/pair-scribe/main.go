// scribe — drop-in replacement for `script(1) -q -F LOG CMD` that supports
// pause/resume of the typescript via SIGUSR1/SIGUSR2.
//
// Why: the macOS `script(1)` typescript wrapping a long-lived interactive
// shell grows unbounded when the user runs TUIs (claude, nvim, lazygit, …)
// whose redraw streams flood the log with bytes that aren't useful for the
// ^Y "copy last command output" feature. We can't truncate script(1)'s log
// while it's running (its FD position goes stale, next write produces a
// sparse file and the binary can abort with `assertion: advance > 0`).
//
// scribe gives the zsh hooks a knob: preexec sends SIGUSR1 before running
// a known-TUI command to stop appending to the log; precmd sends SIGUSR2
// after to resume. Terminal output is never paused — only the on-disk log.
//
// Usage:
//     scribe -log PATH -- CMD [ARGS...]
//
// Drop-in replacement at zshrc's `exec script -q -F $LOG /bin/zsh` line:
//     exec scribe -log "$LOG" -- /bin/zsh
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func main() {
	logPath := flag.String("log", "", "typescript path (required)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s -log PATH -- CMD [ARGS...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if *logPath == "" || len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// O_APPEND so external truncation (or our own) doesn't leave a sparse
	// hole. macOS script(1) doesn't do this — we deliberately do.
	logFile, err := os.OpenFile(*logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fatalf("opening %s: %v", *logPath, err)
	}
	defer logFile.Close()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fatalf("pty.Start: %v", err)
	}
	defer func() { _ = ptmx.Close() }()

	// Propagate SIGWINCH so the child sees terminal resizes.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	winch <- syscall.SIGWINCH // initial size

	// SIGUSR1 pauses the log writer; SIGUSR2 resumes. atomic.Bool reads
	// in the byte-pump goroutine are wait-free, so signals during a heavy
	// stream take effect on the very next packet.
	var paused atomic.Bool
	pauseCh := make(chan os.Signal, 4)
	signal.Notify(pauseCh, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range pauseCh {
			switch s {
			case syscall.SIGUSR1:
				paused.Store(true)
			case syscall.SIGUSR2:
				paused.Store(false)
			}
		}
	}()

	// Stdin must be raw so keystrokes flow through to the child instead of
	// being line-buffered or interpreted by the controlling tty.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fatalf("MakeRaw: %v", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// stdin → pty (user keystrokes into the child)
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// pty → stdout (+ log when not paused). This is the only loop whose
	// completion we wait on — when the child exits, the pty closes and
	// this read returns EOF / error.
	buf := make([]byte, 4096)
	for {
		n, rerr := ptmx.Read(buf)
		if n > 0 {
			_, _ = os.Stdout.Write(buf[:n])
			if !paused.Load() {
				_, _ = logFile.Write(buf[:n])
			}
		}
		if rerr != nil {
			break
		}
	}

	// Restore tty before returning to the parent shell. The deferred
	// Restore handles the normal path; this is belt-and-suspenders for
	// the cmd.Wait error branch below.
	_ = term.Restore(int(os.Stdin.Fd()), oldState)

	werr := cmd.Wait()
	if exitErr, ok := werr.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	if werr != nil {
		fatalf("cmd.Wait: %v", werr)
	}
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "scribe: "+format+"\n", a...)
	os.Exit(1)
}
