package termcmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// RecoveryState keeps ending authority separate from errors encountered during
// teardown: exiting a shell never becomes permission to launch another one.
type RecoveryState uint8

const (
	RecoveryRunning RecoveryState = iota
	RecoveryCleanupNormal
	RecoveryCleanupFault
	RecoveryCleanupTerminated
	RecoveryShellReady
	RecoveryFinished
)

type RecoveryEvent uint8

const (
	RecoveryChildExit RecoveryEvent = iota
	RecoveryWrapperFailure
	RecoveryHostTermination
	RecoveryCleanupComplete
	RecoveryCleanupFailed
	RecoveryHandoff
)

type RecoveryEffect uint8

const (
	RecoveryNoEffect RecoveryEffect = iota
	RecoveryCleanup
	RecoveryExecShell
	RecoveryFinish
)

// RecoveryTransition is the sole owner of permission to hand off to a shell.
// Cleanup remains owed after termination until the IO owner completes teardown.
func RecoveryTransition(state RecoveryState, event RecoveryEvent) (RecoveryState, RecoveryEffect) {
	if state == RecoveryFinished {
		return state, RecoveryNoEffect
	}
	if event == RecoveryHostTermination {
		switch state {
		case RecoveryRunning:
			return RecoveryCleanupTerminated, RecoveryCleanup
		case RecoveryCleanupNormal, RecoveryCleanupFault, RecoveryCleanupTerminated:
			return RecoveryCleanupTerminated, RecoveryNoEffect
		case RecoveryShellReady:
			return RecoveryFinished, RecoveryFinish
		}
	}
	switch state {
	case RecoveryRunning:
		switch event {
		case RecoveryChildExit:
			return RecoveryCleanupNormal, RecoveryCleanup
		case RecoveryWrapperFailure:
			return RecoveryCleanupFault, RecoveryCleanup
		}
	case RecoveryCleanupNormal, RecoveryCleanupTerminated:
		if event == RecoveryCleanupComplete || event == RecoveryCleanupFailed {
			return RecoveryFinished, RecoveryFinish
		}
	case RecoveryCleanupFault:
		if event == RecoveryCleanupComplete {
			return RecoveryShellReady, RecoveryNoEffect
		}
		if event == RecoveryCleanupFailed {
			return RecoveryFinished, RecoveryFinish
		}
	case RecoveryShellReady:
		if event == RecoveryHandoff {
			return RecoveryFinished, RecoveryExecShell
		}
	}
	return state, RecoveryNoEffect
}

type sessionEnding struct {
	event          RecoveryEvent
	cause, cleanup error
}

func (e *sessionEnding) Error() string   { return errors.Join(e.cause, e.cleanup).Error() }
func (e *sessionEnding) Unwrap() []error { return []error{e.cause, e.cleanup} }

var errHostTermination = errors.New("host terminal terminated")

func hostTermination(host hostty.Host) <-chan os.Signal {
	if h, ok := host.(hostty.TerminationHost); ok {
		return h.Terminated()
	}
	return nil
}

// Real values survive channel closure; closure alone is not a signal.
func drainTermination(signals <-chan os.Signal) bool {
	terminated := false
	for {
		select {
		case sig, ok := <-signals:
			if !ok {
				return terminated
			}
			terminated = terminated || sig != nil
		default:
			return terminated
		}
	}
}

func waitInitialGeometry(host hostty.Host, timeout time.Duration) (ptychild.Size, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	resized, signals := host.Resized(), hostTermination(host)
	for {
		if drainTermination(signals) {
			return ptychild.Size{}, errHostTermination
		}
		size, err := host.Size()
		if err != nil {
			return ptychild.Size{}, err
		}
		if size.Rows > 0 && size.Cols > 0 {
			return size, nil
		}
		select {
		case _, ok := <-resized:
			if !ok {
				return ptychild.Size{}, errors.New("terminal resize source closed before valid geometry")
			}
		case sig, ok := <-signals:
			if ok && sig != nil {
				return ptychild.Size{}, errHostTermination
			}
			signals = nil
		case <-deadline.C:
			return ptychild.Size{}, errors.New("terminal size unavailable after 5s; reveal or resize the pane")
		}
	}
}

type nativeShellExec func(string, []string, []string, error) error

func execNativeShell(name string, args, env []string, _ error) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	return syscall.Exec(path, append([]string{name}, args...), env)
}

func unusableTerminal(err error) bool {
	return errors.Is(err, syscall.EIO) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.EBADF) || errors.Is(err, os.ErrClosed)
}

// runShellRecovery retains termination ownership across joined teardown. Once
// OSHost.Close stops signal.Notify, normal signal delivery remains active all
// the way through native exec; there is no replacement watcher or retry loop.
func runShellRecovery(host terminalHost, rt Runtime, observer terminal.Observer, usable bool, execShell nativeShellExec, env ...string) error {
	signals := hostTermination(host)
	err := runShellOnHostObserved(host, rt, observer, env...)
	return finishShellRecovery(signals, rt, usable, execShell, err)
}

func boundedRecoveryReason(err error) string {
	if err == nil {
		return "terminal wrapper unavailable"
	}
	detail := err.Error()
	// Width alone does not bound zero-width combining codepoints or long errors.
	if len(detail) > 2048 {
		detail = detail[:2048]
	}
	return rowtext.SanitizeAndFit(detail, 240)
}

func finishShellRecovery(signals <-chan os.Signal, rt Runtime, usable bool, execShell nativeShellExec, err error) error {
	event := RecoveryChildExit
	cleanupFailed := false
	if err != nil {
		var ending *sessionEnding
		if errors.As(err, &ending) {
			event, cleanupFailed = ending.event, ending.cleanup != nil
		} else {
			event = RecoveryWrapperFailure
		}
	}
	state, _ := RecoveryTransition(RecoveryRunning, event)
	if drainTermination(signals) {
		state, _ = RecoveryTransition(state, RecoveryHostTermination)
	}
	complete := RecoveryCleanupComplete
	if cleanupFailed || !usable || unusableTerminal(err) {
		complete = RecoveryCleanupFailed
	}
	state, _ = RecoveryTransition(state, complete)
	// Also accommodates hosts which publish a final real value at close.
	if drainTermination(signals) {
		state, _ = RecoveryTransition(state, RecoveryHostTermination)
	}
	_, effect := RecoveryTransition(state, RecoveryHandoff)
	if effect != RecoveryExecShell {
		return err
	}
	name, args := rt.ShellCommand()
	if handoffErr := execShell(name, args, os.Environ(), err); handoffErr != nil {
		return errors.Join(err, fmt.Errorf("native shell handoff: %w", handoffErr))
	}
	return nil
}
