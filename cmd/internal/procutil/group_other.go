//go:build !darwin

package procutil

// ProcessGroupMembers conservatively retains ownership on platforms without
// an independent exact-group inspection. Permission denied cannot prove absence.
func ProcessGroupMembers(int) (bool, error) { return true, nil }
