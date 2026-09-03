package couchcore

import (
	"context"
	"os/exec"
	"strings"
)

// GitRunner is the seam over the git binary. couch needs two calls:
// rev-parse --show-toplevel (canonical worktree) and rev-parse
// --git-common-dir (repository identity, which keys saved launch preferences
// -- added at pair#170 M4 when the fleet-policy provider that used to supply
// it was deleted).
//
// ARCH-DRY, stated plainly: this duplicates reviewcmd.Runtime.Git
// (reviewcmd/run.go:34-35, impl runtime.go:61, fake run_test.go:69)
// byte-for-byte in signature, and that seam's own --show-toplevel call at
// run.go:222 goes through it. A shared cmd/internal/gitrun would be the DRY
// move. It is not taken here because every command package in this repo owns
// its own Runtime, and lifting one seam out of that pattern for a single new
// consumer is a larger change than this issue's purpose. Revisit at the third
// consumer -- pair#170 M4 added the second call but not a second seam, so that
// threshold has not moved.
type GitRunner interface {
	Run(dir string, args ...string) (string, error)
	// RunContext is the cancellable form. It exists because the repository
	// identity is resolved on the preview worker, where an unbounded
	// subprocess hangs the start form with no way out. The fleet-policy call
	// this replaced had a 5s bound; dropping the seam's last bounded IO
	// alongside it would have been a silent regression (pair#170 M4).
	RunContext(ctx context.Context, dir string, args ...string) (string, error)
}

type ExecGit struct{}

var _ GitRunner = ExecGit{}

func (ExecGit) Run(dir string, args ...string) (string, error) {
	return ExecGit{}.RunContext(context.Background(), dir, args...)
}

func (ExecGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
