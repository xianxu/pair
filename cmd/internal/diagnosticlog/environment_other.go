//go:build !darwin && !linux

package diagnosticlog

func processEnvironment(pid int) (map[string]string, error) { return nil, ErrUnknownWriters }
