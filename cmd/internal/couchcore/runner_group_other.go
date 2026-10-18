//go:build !darwin

package couchcore

import "github.com/xianxu/pair/cmd/internal/procutil"

// Other platforms retain the conservative EPERM interpretation: permission
// denied does not establish absence. No additional process probe is needed.
func processGroupMembers(pid int) (bool, error) { return procutil.ProcessGroupMembers(pid) }
