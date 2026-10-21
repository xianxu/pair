package testcouch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/internal/reporoot"
)

// Binaries names one qualification owner's candidates and installed read-only
// tools. Build registers cleanup before fleets, so later fleet cleanups run first.
type Binaries struct {
	Source, Dir, Couch, Pair, Helper, Fixture string
	Git, SDLC, Weave, Zellij, Nvim            string
	Evidence                                  map[string]string
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
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	b := Binaries{Source: source, Evidence: map[string]string{}}
	tools := []struct {
		name   string
		target *string
	}{
		{"git", &b.Git}, {"sdlc", &b.SDLC}, {"weave", &b.Weave}, {"zellij", &b.Zellij}, {"nvim", &b.Nvim},
	}
	for _, tool := range tools {
		*tool.target, err = resolveTool(tool.name)
		if err != nil {
			t.Fatalf("native Couch qualification requires installed %s: %v", tool.name, err)
		}
	}
	goTool, err := resolveTool("go")
	if err != nil {
		t.Fatal(err)
	}
	scratch := ""
	if info, e := os.Stat("/Volumes/TestScratch"); e == nil && info.IsDir() {
		scratch = "/Volumes/TestScratch"
	}
	b.Dir, err = os.MkdirTemp(scratch, "couch-fleet-binaries-")
	if err != nil {
		t.Fatal(err)
	}
	// Register before returning any resource to a fleet owner, including failure paths.
	owned := b.Dir
	t.Cleanup(func() {
		if err := os.RemoveAll(owned); err != nil {
			t.Errorf("remove candidate binaries: %v", err)
		}
	})
	b.Dir, err = filepath.EvalSymlinks(b.Dir)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := candidatePaths(b.Dir)
	if err != nil {
		t.Fatal(err)
	}
	b.Couch, b.Pair, b.Helper, b.Fixture = paths["couch"], paths["pair"], paths["pair-launch-helper"], paths["agent-fixture"]
	for _, target := range []struct{ path, pkg string }{
		{b.Couch, "./cmd/couch"}, {b.Helper, "./cmd/pair-launch-helper"}, {b.Fixture, "./cmd/internal/testcouch/agentfixture/cmd"},
	} {
		if output, err := buildCommand(ctx, b.Source, goTool, "build", "-o", target.path, target.pkg); err != nil {
			t.Fatalf("build %s: %v\n%s", target.pkg, err, output)
		}
		if _, err := resolveTool(target.path); err != nil {
			t.Fatal(err)
		}
	}
	b.Pair = testpair.AtContext(t, ctx, b.Pair)
	if _, err := resolveTool(b.Pair); err != nil {
		t.Fatal(err)
	}
	if head, err := buildCommand(ctx, b.Source, b.Git, "rev-parse", "HEAD"); err == nil {
		b.Evidence["source-head"] = strings.TrimSpace(head)
	}
	for _, tool := range tools {
		versionCtx, stop := context.WithTimeout(ctx, 2*time.Second)
		output, err := buildCommand(versionCtx, b.Source, *tool.target, "--version")
		stop()
		version := strings.TrimSpace(output)
		if line, _, ok := strings.Cut(version, "\n"); ok {
			version = line
		}
		if err != nil {
			version = fmt.Sprintf("%s (%v)", version, err)
		}
		b.Evidence[tool.name] = version
		t.Logf("native Couch tool %s: %s (%s)", tool.name, *tool.target, version)
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("native Couch candidate build deadline: %v", err)
	}
	t.Logf("native Couch candidates: source=%s head=%s dir=%s", b.Source, b.Evidence["source-head"], b.Dir)
	return b
}

func candidatePaths(dir string) (map[string]string, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || dir == string(filepath.Separator) {
		return nil, fmt.Errorf("candidate placement requires a clean absolute owned directory: %q", dir)
	}
	result := make(map[string]string, 4)
	for _, name := range []string{"pair", "couch", "pair-launch-helper", "agent-fixture"} {
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
	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 2 * time.Second
	var output buildOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}
