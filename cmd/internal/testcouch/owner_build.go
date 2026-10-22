package testcouch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// BuildConfig names the physical checkout and an optional exact Couch candidate.
// ScratchParent is a caller-selected existing absolute directory.
type BuildConfig struct {
	Source, ScratchParent, Couch string
	Log                          io.Writer
}

// CandidateOwner owns only its private preparation directory. BuildOwned returns
// it on partial failure too, after all started command groups have been joined.
type CandidateOwner struct {
	Binaries Binaries
	mu       sync.Mutex
	closed   bool
}

func (o *CandidateOwner) Close(ctx context.Context) error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(o.Binaries.Dir); err != nil {
		return err
	}
	o.closed = true
	return nil
}

type buildDependencies struct {
	// pairCandidate is selected by the legacy testing wrapper, never ambient
	// policy in the standalone entrypoint. Its lifetime belongs to testpair.Run.
	pairCandidate string
	resolve       func(string) (string, error)
	run           testpair.Runner
}

func BuildOwned(ctx context.Context, cfg BuildConfig) (*CandidateOwner, error) {
	return buildOwned(ctx, cfg, buildDependencies{resolve: resolveTool, run: runCandidateBuild})
}

func runCandidateBuild(ctx context.Context, program string, args []string, dir string, env []string) (string, error) {
	var output buildOutput
	_, err := (couchcore.OSProvisionIO{Env: env}).Run(ctx, couchcore.ProvisionCommand{Dir: dir, Program: program, Args: args, Progress: &output, StreamOutput: true})
	return output.String(), err
}

func physicalDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", fmt.Errorf("requires clean absolute directory: %q", path)
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", physical)
	}
	return physical, nil
}

func buildOwned(parent context.Context, cfg BuildConfig, deps buildDependencies) (*CandidateOwner, error) {
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Couch != "" && (!filepath.IsAbs(cfg.Couch) || filepath.Clean(cfg.Couch) != cfg.Couch) {
		return nil, fmt.Errorf("Couch override requires clean absolute executable: %q", cfg.Couch)
	}
	source, err := physicalDirectory(cfg.Source)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		path := filepath.Join(source, name)
		physical, e := filepath.EvalSymlinks(path)
		if e != nil {
			return nil, e
		}
		rel, e := filepath.Rel(source, physical)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("source file escapes checkout: %s", path)
		}
		info, e := os.Stat(physical)
		if e != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("missing regular source file: %s", path)
		}
	}
	scratch, err := physicalDirectory(cfg.ScratchParent)
	if err != nil {
		return nil, err
	}
	if scratch != cfg.ScratchParent {
		return nil, fmt.Errorf("physical scratch parent required, not alias %q", cfg.ScratchParent)
	}
	// A freshly allocated child cannot contain an already-existing live domain.
	// Reject parents inside one before any mkdir, metadata query or tool lookup;
	// ancestors such as the system temporary directory remain safe to allocate in.
	for _, denied := range append(ownerDenied(nil), source) {
		if !filepath.IsAbs(denied) {
			return nil, fmt.Errorf("invalid live authority path %q", denied)
		}
		physical, e := pathscope.CanonicalFuture(denied)
		if e != nil {
			return nil, e
		}
		if pathscope.Within(physical, scratch) {
			return nil, fmt.Errorf("candidate scratch overlaps readonly/live authority %q", physical)
		}
	}

	b := Binaries{Source: source, Evidence: map[string]string{}}
	selected := map[string]string{}
	toolNames := []string{"go", "bash", "dirname", "git", "sdlc", "weave", "zellij", "nvim", "cc", "c++"}
	if deps.pairCandidate != "" {
		// The unchanged central builder uses Python for candidate alias placement.
		toolNames = append(toolNames, "python3")
	}
	if runtime.GOOS == "darwin" {
		toolNames = append(toolNames, "clang", "clang++", "xcrun", "ld", "ar", "ranlib", "dsymutil")
	}
	for _, name := range toolNames {
		selected[name], err = deps.resolve(name)
		if err != nil {
			return nil, fmt.Errorf("select build tool %s: %w", name, err)
		}
	}
	override := ""
	if cfg.Couch != "" {
		override, err = deps.resolve(cfg.Couch)
		if err != nil {
			return nil, fmt.Errorf("select Couch candidate: %w", err)
		}
	}
	b.Git, b.SDLC, b.Weave, b.Zellij, b.Nvim = selected["git"], selected["sdlc"], selected["weave"], selected["zellij"], selected["nvim"]
	b.Dir, err = os.MkdirTemp(scratch, "couch-candidate-")
	if err != nil {
		return nil, err
	}
	owner := &CandidateOwner{Binaries: b}
	fail := func(err error) (*CandidateOwner, error) { return owner, err }
	tools := filepath.Join(b.Dir, "build-tools")
	if err = os.Mkdir(tools, 0700); err != nil {
		return fail(err)
	}
	for name, path := range selected {
		if err = os.Symlink(path, filepath.Join(tools, name)); err != nil {
			return fail(err)
		}
	}
	// Query only cache locations, never native settings. GOENV disables persistent
	// configuration; no build or toolchain download is allowed during this query.
	queryConfig := filepath.Join(b.Dir, "metadata-config")
	if err = os.Mkdir(queryConfig, 0700); err != nil {
		return fail(err)
	}
	queryEnv := []string{"XDG_CONFIG_HOME=" + queryConfig, "HOME=" + os.Getenv("HOME"), "PATH=" + tools, "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off"}
	for _, key := range []string{"GOCACHE", "GOMODCACHE"} {
		if value := os.Getenv(key); value != "" {
			queryEnv = append(queryEnv, key+"="+value)
		}
	}
	queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
	raw, err := deps.run(queryCtx, selected["go"], []string{"env", "-json", "GOCACHE", "GOMODCACHE"}, source, queryEnv)
	queryCancel()
	if err != nil {
		return fail(fmt.Errorf("resolve shared Go caches: %w: %s", err, raw))
	}
	var caches goCaches
	if err = json.Unmarshal([]byte(raw), &caches); err != nil {
		return fail(fmt.Errorf("decode Go cache locations: %w", err))
	}
	env, err := buildEnvironment(b.Dir, tools, caches)
	if err != nil {
		return fail(err)
	}
	env = append(env, "CC="+selected["cc"], "CXX="+selected["c++"])
	paths, err := candidatePaths(b.Dir)
	if err != nil {
		return fail(err)
	}
	b.Couch, b.Pair, b.Helper, b.Fixture = paths["couch"], paths["pair"], paths["pair-launch-helper"], paths["fixture-agent"]
	owner.Binaries = b
	run := func(tool string, args ...string) (string, error) {
		out, e := deps.run(ctx, tool, args, source, env)
		if cfg.Log != nil && out != "" {
			_, _ = io.WriteString(cfg.Log, out)
		}
		return out, e
	}
	for _, target := range []struct{ path, pkg string }{{b.Couch, "./cmd/couch"}, {b.Helper, "./cmd/pair-launch-helper"}, {b.Fixture, "./cmd/internal/testcouch/agentfixture/cmd"}} {
		if target.path == b.Couch && override != "" {
			b.Couch = override
			owner.Binaries.Couch = override
			continue
		}
		out, e := run(selected["go"], "build", "-o", target.path, target.pkg)
		if e != nil {
			return fail(fmt.Errorf("build %s: %w: %s", target.pkg, e, out))
		}
		if _, e := resolveTool(target.path); e != nil {
			return fail(e)
		}
	}
	err = testpair.PlaceCandidate(ctx, testpair.Placement{Source: source, Destination: b.Pair, Candidate: deps.pairCandidate, Bash: selected["bash"], Env: env}, deps.run)
	if err != nil {
		return fail(err)
	}
	if _, err = resolveTool(b.Pair); err != nil {
		return fail(err)
	}
	if head, e := run(b.Git, "rev-parse", "HEAD"); e == nil {
		b.Evidence["source-head"] = strings.TrimSpace(head)
	} else {
		return fail(fmt.Errorf("source HEAD: %w", e))
	}
	for _, name := range []string{"git", "sdlc", "weave", "zellij", "nvim"} {
		out, e := run(selected[name], "--version")
		version := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
		if e != nil {
			version = fmt.Sprintf("%s (%v)", version, e)
		}
		b.Evidence[name] = version
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	owner.Binaries = b
	return owner, nil
}
