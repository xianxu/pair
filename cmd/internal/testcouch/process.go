package testcouch

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type processIO interface {
	Table(context.Context) ([]procutil.Process, error)
	Signal(int, string, syscall.Signal) error
}

func resourceFor(p procutil.Process) Resource {
	return Resource{ID: strconv.Itoa(p.PID), Generation: p.Identity, Kind: p.Name}
}
func stopOwned(ctx context.Context, io processIO, s OwnedRun, sig syscall.Signal) (OwnedRun, error) {
	rows, err := io.Table(ctx)
	if err != nil {
		return ReduceRun(s, RunEvent{Kind: "unknown", Error: err.Error()}), err
	}
	byPID := make(map[int]procutil.Process, len(rows))
	for _, p := range rows {
		byPID[p.PID] = p
	}
	var failures error
	for _, r := range s.Resources {
		pid, e := strconv.Atoi(r.ID)
		if e != nil || pid <= 0 || r.Generation == "" {
			failures = errors.Join(failures, fmt.Errorf("invalid owned incarnation %q", r.ID))
			continue
		}
		p, exists := byPID[pid]
		if !exists || p.Zombie || p.Identity != "" && p.Identity != r.Generation {
			s = ReduceRun(s, RunEvent{Kind: "gone", Resource: r})
			continue
		}
		if p.Identity == "" {
			failures = errors.Join(failures, fmt.Errorf("identity unavailable for owned pid %d", pid))
			continue
		}
		if sig == 0 {
			continue
		}
		if e = io.Signal(pid, r.Generation, sig); errors.Is(e, syscall.ESRCH) {
			s = ReduceRun(s, RunEvent{Kind: "gone", Resource: r})
		} else if e != nil {
			failures = errors.Join(failures, e)
		}
	}
	if failures != nil {
		s = ReduceRun(s, RunEvent{Kind: "unknown", Error: failures.Error()})
	}
	return s, failures
}

type nativeProcesses struct{}

func (nativeProcesses) Table(ctx context.Context) ([]procutil.Process, error) {
	rows, _, err := (procutil.NativeResourceReader{}).Table(ctx)
	return rows, err
}
func (nativeProcesses) Signal(pid int, generation string, sig syscall.Signal) error {
	// Recheck immediately before signalling; an unreadable identity never
	// grants authority, even when the previous cohort snapshot had one.
	if actual := procutil.Identity(strconv.Itoa(pid)); actual == "" || actual != generation {
		return fmt.Errorf("pid %d generation changed or unreadable", pid)
	}
	return syscall.Kill(pid, sig)
}
