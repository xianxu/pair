package couchcore

import (
	"os/exec"
	"strings"
)

// GitRunner is the seam over the git binary. couch needs exactly one call:
// rev-parse --show-toplevel.
//
// ARCH-DRY, stated plainly: this duplicates reviewcmd.Runtime.Git
// (reviewcmd/run.go:34-35, impl runtime.go:61, fake run_test.go:69)
// byte-for-byte in signature, and that seam's own --show-toplevel call at
// run.go:222 goes through it. A shared cmd/internal/gitrun would be the DRY
// move. It is not taken here because every command package in this repo owns
// its own Runtime, and lifting one seam out of that pattern for a single new
// consumer is a larger change than this issue's purpose. Revisit at the third
// consumer.
type GitRunner interface {
	Run(dir string, args ...string) (string, error)
}

type ExecGit struct{}

var _ GitRunner = ExecGit{}

func (ExecGit) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
