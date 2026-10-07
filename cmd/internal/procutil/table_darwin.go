//go:build darwin

package procutil

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// Table reads every process's pid, parent and start identity in ONE sysctl
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
		})
	}
	return out, nil
}
