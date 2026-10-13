package couchcmd

import (
	"fmt"
	"io"
	"os"
	"syscall"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

// loadOriginPolicy reads the operator's local policy afresh for a submission.
// Failure always supplies a private policy; only the local notice names the file.
func loadOriginPolicy(path string) (core.OriginPolicy, string) {
	invalid := func(reason string) (core.OriginPolicy, string) {
		return core.InvalidOriginPolicy(reason), fmt.Sprintf("Origin privacy policy %q: %s; submission remains private.", path, reason)
	}
	// Nonblocking open prevents a FIFO (including one substituted during a
	// concurrent edit) from hanging submission. Check the opened file itself.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if os.IsNotExist(err) {
		return core.OriginPolicy{}, ""
	}
	if err != nil {
		return invalid("cannot read file")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return invalid("cannot read file")
	}
	if !info.Mode().IsRegular() {
		return invalid("not a regular file")
	}
	if info.Size() > core.MaxOriginPolicyBytes {
		return invalid("file exceeds 64 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, core.MaxOriginPolicyBytes+1))
	if err != nil {
		return invalid("cannot read file")
	}
	policy, err := core.ParseOriginPolicy(data)
	if err != nil {
		return invalid(err.Error())
	}
	return policy, ""
}
