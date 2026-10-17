//go:build darwin

package couchcore

import "github.com/xianxu/pair/cmd/internal/procutil"

// processGroupMembers resolves an ambiguous signal probe without reducing
// group ownership to the direct child. Include zombies conservatively too.
func processGroupMembers(pid int) (bool, error) {
	return procutil.ProcessGroupMembers(pid)
}
