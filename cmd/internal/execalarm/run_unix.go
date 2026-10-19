//go:build darwin || linux

package execalarm

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

// Run validates SECONDS -- PROGRAM ARGS and replaces this process with PROGRAM.
// The command must leave SIGALRM at its default disposition and preserve its
// timer. Successful execution never returns.
func Run(args []string) error {
	seconds, argv, err := parse(args)
	if err != nil {
		return err
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("exec-alarm: resolve program: %w", err)
	}
	// Signal masks belong to OS threads. Keep the unblocked mask on the thread
	// that exec uses, even if the invoking environment blocked SIGALRM.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := unblockAlarm(); err != nil {
		return fmt.Errorf("exec-alarm: unblock SIGALRM: %w", err)
	}
	// A caught disposition becomes default on exec, unlike SIG_IGN. Notify also
	// repairs an explicitly ignored disposition before arming the timer.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGALRM)
	defer signal.Stop(ch)
	if err := armTimer(seconds); err != nil {
		return fmt.Errorf("exec-alarm: arm timer: %w", err)
	}
	// A periodic timer bounds the child even if Go consumed an earlier expiry
	// while this launcher was delayed before exec. At most one full interval
	// remains after exec; no parent or watcher is needed to maintain the alarm.
	err = syscall.Exec(path, argv, os.Environ())
	_ = armTimer(0)
	return fmt.Errorf("exec-alarm: exec: %w", err)
}
