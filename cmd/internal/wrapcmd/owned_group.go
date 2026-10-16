package wrapcmd

import (
	"os/exec"
	"strconv"
	"strings"
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

func (g *ownedGroup) Wait() error {
	g.StartStop()
	<-g.done
	g.reap.Do(func() { g.err = g.cmd.Wait() })
	return g.err
}

// procStatLiveInGroup reads one Linux /proc/<pid>/stat line: true when that
// process is in group pgid and not a zombie. Fields after the parenthesised
// command are: state ppid pgrp ...
func procStatLiveInGroup(raw []byte, pgid int) bool {
	end := strings.LastIndexByte(string(raw), ')')
	if end < 0 {
		return false
	}
	fields := strings.Fields(string(raw[end+1:]))
	if len(fields) < 3 || fields[0] == "Z" || fields[0] == "X" {
		return false
	}
	group, err := strconv.Atoi(fields[2])
	return err == nil && group == pgid
}
