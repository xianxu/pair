//go:build linux

package procutil

// Table reads each process's parent, start identity and niceness from ONE read of its
// /proc/<pid>/stat (#399 close review BR-15). Identity matches Identity's format.
func Table() ([]Process, error) { return readLinuxTable("/proc") }
