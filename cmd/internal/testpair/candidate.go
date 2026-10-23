package testpair

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Placement explicitly selects the checkout and optional prebuilt candidate.
// An empty Candidate builds through the repository's sole Pair builder.
type Placement struct {
	Source, Destination, Candidate, Bash string
	Env                                  []string
}

// Runner must join the command and its owned process group before returning.
type Runner func(context.Context, string, []string, string, []string) (string, error)

func PlaceCandidate(ctx context.Context, p Placement, run Runner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("candidate placement requires a joined runner")
	}
	for _, path := range []string{p.Source, p.Destination, p.Bash} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return fmt.Errorf("candidate placement requires clean absolute paths: %q", path)
		}
	}
	script := filepath.Join(p.Source, "tests/lib/pair-test-binary.sh")
	if info, err := os.Stat(script); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("candidate builder unavailable: %s", script)
	}
	if p.Candidate != "" {
		info, err := os.Stat(p.Candidate)
		if err != nil || !filepath.IsAbs(p.Candidate) || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("invalid explicit Pair candidate %q", p.Candidate)
		}
	}
	env := make([]string, 0, len(p.Env)+1)
	for _, entry := range p.Env {
		if !strings.HasPrefix(entry, "PAIR_TEST_BINARY=") {
			env = append(env, entry)
		}
	}
	env = append(env, "PAIR_TEST_BINARY="+p.Candidate)
	out, err := run(ctx, p.Bash, []string{script, p.Destination}, p.Source, env)
	if err != nil {
		return fmt.Errorf("place Pair candidate: %w: %s", err, out)
	}
	return nil
}
