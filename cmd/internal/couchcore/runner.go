package couchcore

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

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

type ExecRunner struct{}

var _ Runner = ExecRunner{}

// Start inherits this process's stdio. That is what "spawned sessions land in
// the current terminal" means for #145: pair spawns zellij, which needs the
// tty passed through the same way ZellijOps.LaunchSession does
// (runtime.go:31-34). #146 allocates ptys when it takes over routing.
func (ExecRunner) Start(dir string, argv, env []string) (Handle, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("start: empty argv")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
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
		h.code = waitCode(cmd)
		close(h.done)
	}()
	return h, nil
}

// waitCode waits for cmd and maps the result to an exit code.
func waitCode(cmd *exec.Cmd) int {
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if asExitError(err, &ee) {
		return ee.ExitCode()
	}
	return -1
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

// Alive reports whether the child has been reaped. It deliberately does NOT
// consult procutil.Alive: `kill -0` succeeds for a zombie, so a child that has
// exited but not been waited on would read as running.
func (h *execHandle) Alive() bool {
	select {
	case <-h.done:
		return false
	default:
		return true
	}
}

func (h *execHandle) Signal(sig os.Signal) error { return h.cmd.Process.Signal(sig) }

func (h *execHandle) Wait() int {
	<-h.done
	return h.code
}
