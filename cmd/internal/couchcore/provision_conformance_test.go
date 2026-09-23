package couchcore

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This opt-in test runs the installed SDLC and Weave against disposable Git
// repositories. No Brewfile or Makefile exists, so compile cannot install
// packages or replace the operator's shared tool supplier.
func TestProvisionConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_WORKSPACE") != "1" {
		t.Skip("set PAIR_LIVE_WORKSPACE=1 for installed SDLC/Weave conformance")
	}
	for _, program := range []string{"sdlc", "weave"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Fatalf("live conformance requires %s: %v", program, err)
		}
	}
	f := newProvisionFixture(t)
	put := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Dir(f.Remote)
	seed := filepath.Join(root, "base seed")
	put(filepath.Join(seed, "construct/base.manifest"), "prose AGENTS.local.md\n")
	put(filepath.Join(seed, "AGENTS.local.md"), "Fixture dependency context.\n")
	put(filepath.Join(seed, "tracked.txt"), "baseline\n")
	f.git(seed, "init", "-b", "main")
	f.git(seed, "add", ".")
	f.git(seed, "commit", "-m", "dependency baseline")
	dependencyBaseline := f.git(seed, "rev-parse", "HEAD")
	origin := filepath.Join(root, "base remote.git")
	f.git(root, "clone", "--bare", seed, origin)
	localSource := (&url.URL{Scheme: "file", Path: origin}).String()
	// Numbered environments require remote source identities. Git's local
	// subprocess configuration rewrites transport only; the clone records the
	// HTTPS identity while all bytes come from this test's bare repository.
	source := "https://provision-fixture.invalid/base.git"
	f.IO.Env = append(f.IO.Env, "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=url."+localSource+".insteadOf", "GIT_CONFIG_VALUE_0="+source)
	put(filepath.Join(f.Primary, "construct/base.manifest"), "# fixture host\n")
	put(filepath.Join(f.Primary, "construct/deps"), "substrate ../base "+source+"\n")
	f.git(f.Primary, "add", "construct")
	f.git(f.Primary, "commit", "-m", "fixture setup declarations")
	f.git(f.Primary, "push", "upstream", "main")
	f.Base = f.git(f.Primary, "rev-parse", "HEAD")
	// An unpublished primary commit must not enter either numbered workspace.
	f.git(f.Primary, "commit", "--allow-empty", "-m", "primary-only work")
	put(filepath.Join(f.Primary, "primary-dirty.txt"), "keep primary\n")
	production := OSProvisionIO{Env: f.IO.Env}
	p := NewWorkspaceProvisioner(production)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var progress bytes.Buffer
	var firstMarker string
	for _, slot := range []int{1, 2} {
		result, err := p.Ensure(ctx, ProvisionRequest{Path: f.Primary, Slot: slot, Progress: &progress})
		if err != nil {
			t.Fatalf("slot %d: %v\n%s", slot, err, progress.String())
		}
		if result.Path != f.host(slot) || result.BaselineSHA != f.Base || result.Disposition != "created" {
			t.Fatalf("slot %d result: %+v", slot, result)
		}
		if got := f.git(result.Path, "rev-parse", "HEAD"); got != f.Base {
			t.Fatalf("slot baseline %s != %s", got, f.Base)
		}
		if got := f.git(result.Path, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "upstream/main" {
			t.Fatal(got)
		}
		raw, err := production.Run(ctx, ProvisionCommand{Program: "sdlc", Dir: result.Path, Args: []string{"workspace", "--json"}})
		if err != nil {
			t.Fatal(err)
		}
		id, err := ParseWorkspaceIdentity(raw)
		if err != nil {
			t.Fatalf("actual identity: %v: %s", err, raw)
		}
		if id.Kind != "slot" || id.Slot == nil || *id.Slot != slot || id.WorktreeRoot != result.Path || id.EnvironmentHost == nil {
			t.Fatalf("identity %+v", id)
		}
		dep := filepath.Join(filepath.Dir(result.Path), "base")
		info, err := os.Stat(filepath.Join(dep, ".git"))
		if err != nil || !info.IsDir() {
			t.Fatalf("dependency is not an ordinary clone: %v", err)
		}
		if got := f.git(dep, "rev-parse", "HEAD"); got != dependencyBaseline {
			t.Fatalf("dependency baseline %s", got)
		}
		if got := f.git(dep, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "origin/main" {
			t.Fatalf("dependency upstream %s", got)
		}
		admin := f.git(result.Path, "rev-parse", "--absolute-git-dir")
		marker := filepath.Join(admin, "couch-setup-success.json")
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("success marker: %v", err)
		}
		if slot == 1 {
			firstMarker = marker
		}
	}
	host := f.host(1)
	dep := filepath.Join(filepath.Dir(host), "base")
	f.git(host, "switch", "-c", "host-feature")
	put(filepath.Join(host, "host-dirty.txt"), "host dirty\n")
	f.git(dep, "switch", "-c", "dependency-feature")
	f.git(dep, "commit", "--allow-empty", "-m", "private dependency work")
	dependencyHead := f.git(dep, "rev-parse", "HEAD")
	put(filepath.Join(dep, "tracked.txt"), "dirty dependency\n")
	put(filepath.Join(dep, "untracked.txt"), "untracked dependency\n")
	// Delete only this test-created marker to exercise unconfirmed setup using
	// the same operation. Weave must preserve existing private dependency work.
	if err := os.Remove(firstMarker); err != nil {
		t.Fatal(err)
	}
	progress.Reset()
	result, err := p.Ensure(ctx, ProvisionRequest{Path: f.Primary, Slot: 1, Progress: &progress})
	if err != nil {
		t.Fatalf("reprepare: %v\n%s", err, progress.String())
	}
	if result.Disposition != "prepared" || result.BaselineSHA != f.Base {
		t.Fatalf("reprepare %+v", result)
	}
	if f.git(host, "branch", "--show-current") != "host-feature" || f.git(dep, "branch", "--show-current") != "dependency-feature" || f.git(dep, "rev-parse", "HEAD") != dependencyHead {
		t.Fatal("reprepare changed existing branches or dependency revision")
	}
	for path, want := range map[string]string{filepath.Join(f.Primary, "primary-dirty.txt"): "keep primary\n", filepath.Join(host, "host-dirty.txt"): "host dirty\n", filepath.Join(dep, "tracked.txt"): "dirty dependency\n", filepath.Join(dep, "untracked.txt"): "untracked dependency\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("preserved %s = %q, %v", path, got, err)
		}
	}
	before, err := os.ReadFile(firstMarker)
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := os.Stat(firstMarker)
	if err != nil {
		t.Fatal(err)
	}
	progress.Reset()
	result, err = p.Ensure(ctx, ProvisionRequest{Path: f.Primary, Slot: 1, Progress: &progress})
	if err != nil || result.Disposition != "reused" {
		t.Fatalf("ready reuse %+v: %v", result, err)
	}
	after, err := os.ReadFile(firstMarker)
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.Stat(firstMarker)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !stamp.ModTime().Equal(current.ModTime()) || progress.Len() != 0 {
		t.Fatalf("ready reuse rewrote marker or reran setup: %s", progress.String())
	}
}
