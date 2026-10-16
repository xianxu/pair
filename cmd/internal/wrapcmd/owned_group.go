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
				timer := time.NewTimer(g.grace)
				<-timer.C
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
		}()
	})
}

func (g *ownedGroup) Wait() error {
	g.StartStop()
	<-g.done
	g.reap.Do(func() { g.err = g.cmd.Wait() })
	return g.err
}
