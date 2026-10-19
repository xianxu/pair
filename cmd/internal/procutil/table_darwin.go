//go:build darwin

package procutil

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// SZOMB in Darwin's sys/proc.h: awaiting collection by parent.
const darwinZombieStatus = 5

// Table reads every process's pid, parent, start identity and niceness in ONE sysctl
// (kern.proc.all), so a row's parent and identity describe the same process at
// the same moment (#399 close review BR-15). Identity matches Identity's format.
func Table() ([]Process, error) {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	out := make([]Process, 0, len(procs))
	for _, p := range procs {
		if p.Proc.P_pid <= 0 {
			continue
		}
		started := p.Proc.P_starttime
		out = append(out, Process{
			PID:      int(p.Proc.P_pid),
			PPID:     int(p.Eproc.Ppid),
			Identity: strconv.FormatInt(started.Sec, 10) + "." + strconv.FormatInt(int64(started.Usec), 10),
			Nice:     int(p.Proc.P_nice),
			Name:     unix.ByteSliceToString(p.Proc.P_comm[:]),
			Zombie:   p.Proc.P_stat == darwinZombieStatus,
			TTY:      darwinTTYName(p.Eproc.Tdev),
		})
	}
	return out, nil
}

// darwinNoDevice is NODEV, (dev_t)-1: no controlling terminal.
const darwinNoDevice = -1

func darwinTTYName(tdev int32) string {
	if tdev == darwinNoDevice || tdev == 0 {
		return ""
	}
	return strconv.FormatInt(int64(uint32(tdev)), 10)
}
