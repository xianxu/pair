//go:build darwin

package diagnosticlog

import "golang.org/x/sys/unix"

func processEnvironment(pid int) (map[string]string, error) {
	b, e := unix.SysctlRaw("kern.procargs2", pid)
	if e != nil {
		return nil, e
	}
	return parseDarwinEnvironment(b)
}
