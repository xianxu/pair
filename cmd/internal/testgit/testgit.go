// Package testgit contains invocation-scoped support for real Git test fixtures.
package testgit

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// FixtureTreeStats measures retained regular files and directories, including
// the owned root. LogicalBytes is the sum of regular-file lengths, not blocks.
type FixtureTreeStats struct{ Files, Dirs, LogicalBytes, SampleHooks uint64 }

// FixtureTreeBudget gives inclusive per-constructor limits.
type FixtureTreeBudget FixtureTreeStats

func (b FixtureTreeBudget) Verdict(s FixtureTreeStats) error {
	if s.Files > b.Files || s.Dirs > b.Dirs || s.LogicalBytes > b.LogicalBytes || s.SampleHooks > b.SampleHooks {
		return fmt.Errorf("fixture tree exceeds budget: inventory=%+v budget=%+v", s, b)
	}
	return nil
}

// Add combines independent retained inventories, rejecting wrapped counters.
func (s FixtureTreeStats) Add(other FixtureTreeStats) (FixtureTreeStats, error) {
	a := []uint64{s.Files, s.Dirs, s.LogicalBytes, s.SampleHooks}
	b := []uint64{other.Files, other.Dirs, other.LogicalBytes, other.SampleHooks}
	for i := range a {
		if b[i] > math.MaxUint64-a[i] {
			return FixtureTreeStats{}, fmt.Errorf("fixture inventory counter %d overflow", i)
		}
		a[i] += b[i]
	}
	return FixtureTreeStats{Files: a[0], Dirs: a[1], LogicalBytes: a[2], SampleHooks: a[3]}, nil
}

// AuditFixtureTree walks only the constructor-owned root. Symlinks and special
// files are errors; neither their targets nor sibling trees are inspected.
func AuditFixtureTree(root string) (FixtureTreeStats, error) {
	var total FixtureTreeStats
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if path == root && !info.IsDir() {
			return fmt.Errorf("fixture root is not a directory: %s", root)
		}
		var next FixtureTreeStats
		switch {
		case info.IsDir():
			next.Dirs = 1
		case info.Mode().IsRegular():
			if info.Size() < 0 {
				return fmt.Errorf("negative file length: %s", path)
			}
			next.Files = 1
			next.LogicalBytes = uint64(info.Size())
			if filepath.Base(filepath.Dir(path)) == "hooks" && strings.HasSuffix(d.Name(), ".sample") {
				next.SampleHooks = 1
			}
		default:
			return fmt.Errorf("unsupported fixture entry %s: %s", path, info.Mode())
		}
		total, err = total.Add(next)
		return err
	})
	if err != nil {
		return FixtureTreeStats{}, fmt.Errorf("audit fixture %s: %w", root, err)
	}
	return total, nil
}

// RunWithEmptyGitTemplate scopes Git's template selection around the entire test
// invocation (including testing cleanup). Call it from TestMain before tests
// start, after any helper dispatch; it changes process-global environment.
// A callback failure is retained even if cleanup fails. Infrastructure failures
// are printed and turn an otherwise successful invocation into exit status 1.
func RunWithEmptyGitTemplate(run func() int) (status int) {
	template, err := os.MkdirTemp("", "pair-git-template-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "empty Git template setup: %v\n", err)
		return 1
	}
	old, present := os.LookupEnv("GIT_TEMPLATE_DIR")
	defer func() {
		var restoreErr error
		if present {
			restoreErr = os.Setenv("GIT_TEMPLATE_DIR", old)
		} else {
			restoreErr = os.Unsetenv("GIT_TEMPLATE_DIR")
		}
		for _, err := range []error{restoreErr, os.RemoveAll(template)} {
			if err != nil {
				fmt.Fprintf(os.Stderr, "empty Git template cleanup: %v\n", err)
				if status == 0 {
					status = 1
				}
			}
		}
	}()
	if err := os.Setenv("GIT_TEMPLATE_DIR", template); err != nil {
		fmt.Fprintf(os.Stderr, "empty Git template environment: %v\n", err)
		return 1
	}
	return run()
}
