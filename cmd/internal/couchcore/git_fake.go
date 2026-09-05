package couchcore

import (
	"context"
	"fmt"
)

// GitCall is the fake's canned-reply key. It includes Dir deliberately:
// keying on argv alone makes "was git run in the right directory" -- the only
// bug Resolve can have -- untestable, and lets a test that names the PathOps
// seam pass with that seam deleted.
type GitCall struct {
	Dir  string
	Args string
}

type FakeGit struct {
	replies map[GitCall]string
	Ops     []string
}

var _ GitRunner = (*FakeGit)(nil)

func NewFakeGit(replies map[GitCall]string) *FakeGit {
	if replies == nil {
		replies = map[GitCall]string{}
	}
	return &FakeGit{replies: replies}
}

func (f *FakeGit) Run(dir string, args ...string) (string, error) {
	return f.RunContext(context.Background(), dir, args...)
}

// RunContext honours cancellation before answering, so a test can prove the
// caller's context actually reaches the subprocess seam rather than being
// carried and dropped.
func (f *FakeGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	key := GitCall{Dir: dir, Args: joinArgs(args)}
	f.Ops = append(f.Ops, dir+": "+key.Args)
	out, ok := f.replies[key]
	if !ok {
		return "", fmt.Errorf("fake git: no canned reply for %q in %s", key.Args, dir)
	}
	return trimTrailingNewline(out), nil
}
