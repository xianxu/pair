package couchcore

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Runner is the seam over spawning child processes.
//
// Verified genuinely new: `grep -rn 'type Handle' cmd/` and
// `grep -rn 'Start(dir' cmd/` both return nothing. launcher.ProcOps is
// sidecar-named (SpawnSessionWatcher, SpawnTitlePoller, DevRebuild --
// runtime.go:82-92); ZellijOps.LaunchSession is blocking and zellij-specific;
// wrapcmd spawns its child inline and unseamed (wrap.go:2330-2332), which is
// an absence of a seam rather than a counter-example.
//
// Shape discipline follows ariadne's weavefs.Runner (one small interface, one
// production impl, a compile-time assertion), though this contract is larger:
// weave's Run(dir, argv) error is synchronous with no handle.
//
// A kill goes THROUGH this seam. osruntime.go:66-84 records what happens when
// it does not: a delete-session inlined below the seam meant a test asserting
// "a foreign session is never deleted" passed while the hazard sat untouched.
type Runner interface {
	Start(dir string, argv, env []string) (Handle, error)
	StartBlocked(dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error)
}

type BlockedHandle interface {
	Handle
	Acknowledge() error
	Cancel() error
}

type Handle interface {
	ID() string
	PID() int
	// Identity is a kernel start token, not a bare PID: PID reuse can
	// otherwise transfer ownership of a record to an unrelated process.
	// sessionwatch/run.go:143 re-authorizes for exactly this reason.
	Identity() string
	Alive() bool
	Signal(os.Signal) error
	Wait() int
}

type ExecRunner struct {
	LaunchHelper string
}

var _ Runner = ExecRunner{}

// Start inherits this process's stdio. That is what "spawned sessions land in
// the current terminal" means for #145: pair spawns zellij, which needs the
// tty passed through the same way ZellijOps.LaunchSession does
// (runtime.go:31-34). #146 allocates ptys when it takes over routing.
func (ExecRunner) Start(dir string, argv, env []string) (Handle, error) {
	return startExecChild(dir, argv, env, nil)
}

func (r ExecRunner) StartBlocked(dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	return startBlockedChild(startExecChild, r.LaunchHelper, dir, argv, env, timeout)
}

func startExecChild(dir string, argv, env []string, extraFiles []*os.File) (Handle, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("start: empty argv")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = mergeChildEnvironment(os.Environ(), env)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.ExtraFiles = extraFiles
	// Every Couch child is the leader of one owned process group. Pair may
	// start pre-handoff helpers before the zellij session exists; keeping those
	// helpers in this group gives the handle one production authority over the
	// complete pre-session incarnation rather than only its direct child.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s in %s: %w", argv[0], dir, err)
	}
	pid := cmd.Process.Pid
	h := &execHandle{
		cmd:      cmd,
		pid:      pid,
		identity: procutil.Identity(strconv.Itoa(pid)),
		done:     make(chan struct{}),
	}
	// Reap in the background rather than only on Wait().
	//
	// Without this, Alive() is wrong for an exited-but-unreaped child:
	// procutil.Alive is `kill -0`, which SUCCEEDS for a zombie, so a finished
	// child reads as running until somebody calls Wait. The live conformance
	// check caught exactly that -- a real child killed by SIGINT stayed
	// "alive" while the fake correctly reported it dead.
	//
	// Reaping here also makes liveness a closed channel rather than a syscall,
	// which is the same shape FakeRunner models.
	go func() {
		h.code = procutil.WaitCode(cmd)
		close(h.done)
	}()
	return h, nil
}

// mergeChildEnvironment makes every supplied child key authoritative over the
// inherited process environment. The last supplied value wins if a caller
// repeats a key, matching ordinary environment-overlay semantics without
// leaving duplicate entries for a child runtime to interpret differently.
func mergeChildEnvironment(inherited, supplied []string) []string {
	lastSupplied := make(map[string]int, len(supplied))
	for i, item := range supplied {
		lastSupplied[environmentKey(item)] = i
	}
	merged := make([]string, 0, len(inherited)+len(supplied))
	for _, item := range inherited {
		if _, overridden := lastSupplied[environmentKey(item)]; !overridden {
			merged = append(merged, item)
		}
	}
	for i, item := range supplied {
		if lastSupplied[environmentKey(item)] == i {
			merged = append(merged, item)
		}
	}
	return merged
}

func environmentKey(item string) string {
	key, _, _ := strings.Cut(item, "=")
	return key
}

type execHandle struct {
	cmd      *exec.Cmd
	pid      int
	identity string

	// done closes once the child has been reaped; code is written before the
	// close, so reading it after <-done needs no further synchronisation.
	done chan struct{}
	code int
}

func (h *execHandle) ID() string       { return strconv.Itoa(h.pid) }
func (h *execHandle) PID() int         { return h.pid }
func (h *execHandle) Identity() string { return h.identity }

// Alive reports whether any member of this handle's owned process group still
// exists. The direct child may already be reaped while a pre-handoff sidecar
// remains, so direct-child liveness is insufficient for rollback.
func (h *execHandle) Alive() bool { return ownedProcessGroupAlive(h.pid) }

func (h *execHandle) Signal(sig os.Signal) error { return signalOwnedProcessGroup(h.pid, sig) }

func (h *execHandle) Wait() int {
	<-h.done
	return h.code
}

func signalOwnedProcessGroup(pid int, sig os.Signal) error {
	unixSignal, ok := sig.(syscall.Signal)
	if !ok {
		return fmt.Errorf("signal owned process group %d: unsupported signal %T", pid, sig)
	}
	err := syscall.Kill(-pid, unixSignal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func ownedProcessGroupAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(-pid, syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
