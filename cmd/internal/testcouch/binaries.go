package testcouch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/internal/reporoot"
)

// Binaries names one qualification owner's candidates and installed read-only
// tools. Build registers cleanup before fleets, so later fleet cleanups run first.
type Binaries struct {
	Source, Dir, Couch, Pair, Helper, Fixture     string
	Git, GitUploadPack, SDLC, Weave, Zellij, Nvim string
	Evidence                                      map[string]string
}

const buildOutputLimit = 64 * 1024

// Build creates candidates once for the calling qualification owner. It has no
// global cache; Pair alone uses the existing testpair candidate resolver.
func Build(t testing.TB) Binaries {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	source, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	info, scratchErr := os.Stat("/Volumes/TestScratch")
	cfg := defaultBuildConfig(source, testBuildLog{t: t}, os.TempDir(), scratchErr == nil && info.IsDir())
	// Preserve the suite/process-owned resolver only in this compatibility
	// wrapper. Standalone BuildOwned never observes PAIR_TEST_BINARY.
	selected := testpair.BinaryContext(t, ctx)
	owner, err := buildOwned(ctx, cfg, buildDependencies{resolve: resolveTool, run: runCandidateBuild, pairCandidate: selected})
	if owner != nil {
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if e := owner.Close(ctx); e != nil {
				t.Errorf("remove candidates: %v", e)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	return owner.Binaries
}

type testBuildLog struct{ t testing.TB }

func (l testBuildLog) Write(p []byte) (int, error) { l.t.Log(string(p)); return len(p), nil }

func candidatePaths(dir string) (map[string]string, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || dir == string(filepath.Separator) {
		return nil, fmt.Errorf("candidate placement requires a clean absolute owned directory: %q", dir)
	}
	result := make(map[string]string, 4)
	for _, name := range []string{"pair", "couch", "pair-launch-helper", "fixture-agent"} {
		result[name] = filepath.Join(dir, name)
	}
	return result, nil
}

func validateToolFacts(path string, mode os.FileMode) error {
	if !filepath.IsAbs(path) || !mode.IsRegular() || mode.Perm()&0111 == 0 {
		return fmt.Errorf("tool must be an absolute regular executable: %q", path)
	}
	return nil
}

func resolveTool(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("tool lookup returned relative path %q", path)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if err := validateToolFacts(path, info.Mode()); err != nil {
		return "", err
	}
	return path, nil
}

// buildOutput accepts all child writes while retaining bounded failure evidence.
type buildOutput struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
}

func (b *buildOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := min(len(p), buildOutputLimit-len(b.data))
	b.data = append(b.data, p[:n]...)
	b.truncated = b.truncated || n < len(p)
	return len(p), nil
}
func (b *buildOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := string(b.data)
	if b.truncated {
		result += "\n[diagnostics truncated]"
	}
	return result
}
func buildCommand(ctx context.Context, dir, tool string, args ...string) (string, error) {
	var output buildOutput
	_, err := (couchcore.OSProvisionIO{}).Run(ctx, couchcore.ProvisionCommand{Dir: dir, Program: tool, Args: append([]string(nil), args...), Progress: &output, StreamOutput: true})
	return output.String(), err
}

func defaultBuildConfig(source string, log io.Writer, temp string, scratchPresent bool) BuildConfig {
	return BuildConfig{Source: source, ScratchParent: legacyScratchParentFrom(temp, scratchPresent), Log: log}
}
