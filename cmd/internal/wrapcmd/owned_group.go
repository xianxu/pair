package wrapcmd

import (
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ownedGroup only receives the direct child created by this wrapper with
// Setsid. Keep its leader unreaped until the final group signal: that birth
// fences the process-group number against reuse. No discovered PID is accepted.
type ownedGroup struct {
	cmd   *exec.Cmd
	grace time.Duration
	stop  sync.Once
	reap  sync.Once
	done  chan struct{}
	err   error
}

func newOwnedGroup(cmd *exec.Cmd, grace time.Duration) *ownedGroup {
	return &ownedGroup{cmd: cmd, grace: grace, done: make(chan struct{})}
}

func (g *ownedGroup) StartStop() {
	g.stop.Do(func() {
		go func() {
			defer close(g.done)
			pid := g.cmd.Process.Pid
			if err := syscall.Kill(-pid, syscall.SIGTERM); err != syscall.ESRCH {
				for deadline := time.Now().Add(g.grace); time.Now().Before(deadline); time.Sleep(groupPoll) {
					if !groupHasLiveMember(pid) {
						return
					}
				}
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
		}()
	})
}

const groupPoll = 10 * time.Millisecond

// groupHasLiveMember reports whether the group still has a member a signal
// reaches. Darwin answers EPERM once only zombies remain (our unreaped leader,
// or members mid-exit) and ESRCH once it is empty; neither needs SIGKILL, so
// cleanup ends early instead of always spending the grace.
func groupHasLiveMember(pgid int) bool { return syscall.Kill(-pgid, 0) == nil }

func (g *ownedGroup) Wait() error {
	g.StartStop()
	<-g.done
	g.reap.Do(func() { g.err = g.cmd.Wait() })
	return g.err
}
