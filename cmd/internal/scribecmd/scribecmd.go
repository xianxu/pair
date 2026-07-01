// Package scribecmd is a drop-in replacement for `script(1) -q -F LOG CMD` that
// supports pause/resume of the typescript via SIGUSR1/SIGUSR2.
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
//     exec pair scribe -log "$LOG" -- /bin/zsh   (or the bin/pair-scribe shim)
//
// Packaging: the logic lives here behind Run(); it is reached two ways with
// identical behavior — the standalone bin/pair-scribe shim (cmd/pair-scribe,
// which keeps ~/.local/bin/pair-scribe + the user's ~/.zshrc wiring working)
// and the `pair scribe` dispatcher route (#96, on #92's streaming seam). Both
// pass the real os.Stdin/os.Stdout, so the io.Reader→*os.File assertion below
// always succeeds in production.
package scribecmd

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

// Run is the process-facing entry for the scribe PTY logging wrapper. args is
// os.Args[1:] (no program name); stdin/stdout/stderr are the real streams. The
// returned int is the process exit code: 2 for a usage error, the wrapped
// child's exit code on success, 1 on a fatal error.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scribe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	logPath := fs.String("log", "", "typescript path (required)")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: scribe -log PATH -- CMD [ARGS...]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		// ContinueOnError already wrote the error + usage to stderr.
		return 2
	}
	rest := fs.Args()
	if *logPath == "" || len(rest) == 0 {
		fs.Usage()
		return 2
	}

	// O_APPEND so external truncation (or our own) doesn't leave a sparse
	// hole. macOS script(1) doesn't do this — we deliberately do.
	logFile, err := os.OpenFile(*logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fatalf(stderr, "opening %s: %v", *logPath, err)
	}
	defer logFile.Close()

	cmd := exec.Command(rest[0], rest[1:]...)
	cmd.Env = os.Environ()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fatalf(stderr, "pty.Start: %v", err)
	}
	defer func() { _ = ptmx.Close() }()

	// The *os.File view of stdin, needed for raw-mode + winsize propagation.
	// In production stdin IS os.Stdin so this is non-nil and behavior matches
	// the pre-extraction binary; a non-file reader (a test) leaves it nil and
	// the terminal ops are skipped.
	stdinFile, _ := stdin.(*os.File)

	// Propagate SIGWINCH so the child sees terminal resizes.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			if stdinFile != nil {
				_ = pty.InheritSize(stdinFile, ptmx)
			}
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
	var oldState *term.State
	if stdinFile != nil {
		s, err := term.MakeRaw(int(stdinFile.Fd()))
		if err != nil {
			return fatalf(stderr, "MakeRaw: %v", err)
		}
		oldState = s
		defer func() { _ = term.Restore(int(stdinFile.Fd()), oldState) }()
	}

	// stdin → pty (user keystrokes into the child)
	go func() {
		_, _ = io.Copy(ptmx, stdin)
	}()

	// pty → stdout (+ log when not paused). This is the only loop whose
	// completion we wait on — when the child exits, the pty closes and
	// this read returns EOF / error.
	buf := make([]byte, 4096)
	for {
		n, rerr := ptmx.Read(buf)
		if n > 0 {
			_, _ = stdout.Write(buf[:n])
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
	if stdinFile != nil {
		_ = term.Restore(int(stdinFile.Fd()), oldState)
	}

	werr := cmd.Wait()
	if exitErr, ok := werr.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	if werr != nil {
		return fatalf(stderr, "cmd.Wait: %v", werr)
	}
	return 0
}

func fatalf(w io.Writer, format string, a ...any) int {
	fmt.Fprintf(w, "scribe: "+format+"\n", a...)
	return 1
}
