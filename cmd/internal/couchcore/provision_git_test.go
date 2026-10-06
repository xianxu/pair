package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ProvisionFixture retains real Git/filesystem state across invocations and
// supplies stateful SDLC/Weave outcomes through the production command seam.
// Live conformance separately verifies the fixture's external contract.
type ProvisionFixture struct {
	t                     *testing.T
	Primary, Remote, Base string
	IO                    OSProvisionIO
	WeaveCalls            int
	FailWeave             bool
	// DepSources names the dependencies whose clone source weave knows:
	// compile clones a missing declared dependency it knows (as a layer) and
	// fails, with weave's own line, on one it does not (pair#387).
	DepSources map[string]bool
	AfterGit   func(ProvisionCommand, []byte) error
	// WeaveIdentity is the installed weave's identity (ProgramIdentifier).
	WeaveIdentity string
}

func (f *ProvisionFixture) ProgramIdentity(program string) string {
	return program + " " + f.WeaveIdentity
}

func newProvisionFixture(t *testing.T, repositoryNames ...string) *ProvisionFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := "repo-name"
	if len(repositoryNames) > 0 {
		repo = repositoryNames[0]
	}
	f := &ProvisionFixture{t: t, Primary: filepath.Join(root, "fleet space", repo), Remote: filepath.Join(root, "remote.git")}
	f.IO.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.com", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.com", "GIT_TERMINAL_PROMPT=0")
	if err := os.MkdirAll(f.Primary, 0700); err != nil {
		t.Fatal(err)
	}
	f.git(f.Primary, "init", "-b", "main")
	f.git(f.Primary, "commit", "--allow-empty", "-m", "remote baseline")
	f.Base = f.git(f.Primary, "rev-parse", "HEAD")
	f.git(root, "clone", "--bare", f.Primary, f.Remote)
	f.git(f.Primary, "remote", "add", "upstream", f.Remote)
	return f
}
func (f *ProvisionFixture) git(dir string, args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = f.IO.Env
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
func (f *ProvisionFixture) host(n int) string {
	return filepath.Join(filepath.Dir(f.Primary), "worktree", fmt.Sprintf("%s-slot%d", filepath.Base(f.Primary), n), filepath.Base(f.Primary))
}

// cloneDeclaredDeps is the fixture weave's dependency acquisition.
func (f *ProvisionFixture) cloneDeclaredDeps(host string) error {
	declared, err := DeclaredDepsOf(filepath.Dir(host), host)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}
	for _, d := range declared.Deps {
		if d.Present || d.Outside {
			continue
		}
		if !f.DepSources[d.Rel] {
			return fmt.Errorf("Error: missing substrate %s declared in %s: record its source in construct/deps", d.Path, d.Owner)
		}
		os.MkdirAll(filepath.Join(d.Path, "construct"), 0o755)
		f.git(d.Path, "init", "-q", "-b", "main")
		os.WriteFile(filepath.Join(d.Path, "construct", "base.manifest"), []byte("# layer\n"), 0o644)
		f.git(d.Path, "add", ".")
		f.git(d.Path, "commit", "-q", "-m", "cloned")
	}
	return nil
}

func (f *ProvisionFixture) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	switch c.Program {
	case "sdlc":
		// Identity is assembled from the fixture's repository plus actual membership.
		out, err := f.IO.Run(ctx, ProvisionCommand{Dir: c.Dir, Program: "git", Args: []string{"rev-parse", "--show-toplevel"}})
		if err != nil {
			return nil, err
		}
		top := strings.TrimSpace(string(out))
		slot := 0
		kind := "primary"
		env := filepath.Dir(f.Primary)
		rest := "main"
		if top != f.Primary {
			for n := 1; n <= 20; n++ {
				if top == f.host(n) {
					slot = n
					kind = "slot"
					env = filepath.Dir(top)
					rest = fmt.Sprintf("main-slot%d", n)
					break
				}
			}
			if slot == 0 {
				return nil, errors.New("unknown fixture worktree")
			}
		}
		common := f.git(top, "rev-parse", "--path-format=absolute", "--git-common-dir")
		addr := fmt.Sprintf("%s:%d", filepath.Base(f.Primary), slot)
		head := f.git(top, "rev-parse", "HEAD")
		branch := f.git(top, "branch", "--show-current")
		id := WorkspaceIdentity{SchemaVersion: 2, Repo: filepath.Base(f.Primary), RepoIdentity: common, PrimaryRoot: f.Primary, FleetRoot: filepath.Dir(f.Primary), EnvironmentRoot: env, WorktreeRoot: top, Kind: kind, Address: &addr, Slot: &slot, Branch: &branch, Head: &head, RestingBranch: &rest}
		if slot > 0 {
			id.EnvironmentHost = &EnvironmentHost{Repo: id.Repo, Slot: slot, RepoIdentity: common, PrimaryRoot: f.Primary, WorktreeRoot: top}
		}
		return json.Marshal(id)
	case "weave":
		f.WeaveCalls++
		if c.Progress != nil {
			fmt.Fprintln(c.Progress, "fixture weave compile")
		}
		if f.FailWeave {
			return nil, errors.New("fixture setup failed")
		}
		if err := f.cloneDeclaredDeps(c.Dir); err != nil {
			return []byte(err.Error() + "\n"), fmt.Errorf("exit status 1: %w", err)
		}
		return nil, ctx.Err()
	default:
		out, err := f.IO.Run(ctx, c)
		if err == nil && f.AfterGit != nil {
			err = f.AfterGit(c, out)
		}
		return out, err
	}
}
func TestProvisionHostRemoteBaselineAndRepeatReadiness(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "commit", "--allow-empty", "-m", "local only")
	dirty := filepath.Join(f.Primary, "local.txt")
	os.WriteFile(dirty, []byte("uncommitted"), 0600)
	p := NewWorkspaceProvisioner(f)
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Path != f.host(1) || r.BaselineSHA != f.Base || r.Disposition != "created" {
		t.Fatalf("result %+v", r)
	}
	if got := f.git(r.Path, "rev-parse", "HEAD"); got != f.Base {
		t.Fatalf("baseline %s", got)
	}
	if got := f.git(r.Path, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "upstream/main" {
		t.Fatal(got)
	}
	f.git(r.Path, "switch", "-c", "issue-work")
	slotDirty := filepath.Join(r.Path, "work.txt")
	os.WriteFile(slotDirty, []byte("keep"), 0600)
	f.FailWeave = true
	r, err = p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Disposition != "reused" || f.WeaveCalls != 1 {
		t.Fatalf("result %+v compiles=%d", r, f.WeaveCalls)
	}
	if f.git(r.Path, "branch", "--show-current") != "issue-work" {
		t.Fatal("changed branch")
	}
	for _, path := range []string{dirty, slotDirty} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	f.FailWeave = false
	r2, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 2})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Path != f.host(2) || r2.BaselineSHA != f.Base {
		t.Fatalf("second %+v", r2)
	}
}
func TestProvisionHostMissingSuccessRepeatsSameOperation(t *testing.T) {
	f := newProvisionFixture(t)
	f.FailWeave = true
	p := NewWorkspaceProvisioner(f)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	if _, err := p.Ensure(context.Background(), req); err == nil {
		t.Fatal("failed setup accepted")
	}
	f.git(f.Primary, "commit", "--allow-empty", "-m", "new remote")
	f.git(f.Primary, "push", "upstream", "main")
	f.FailWeave = false
	// R5: the hand-off failure is remembered for unchanged inputs, so a plain
	// repeat refuses with it and does not recompile; the fix happened outside
	// the slot, so an explicit reconcile (IgnoreMemo) compiles again.
	if _, err := p.Ensure(context.Background(), req); err == nil || !strings.Contains(err.Error(), "fixture setup failed") || f.WeaveCalls != 1 {
		t.Fatalf("known failure: err=%v calls=%d", err, f.WeaveCalls)
	}
	req.IgnoreMemo = true
	got, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.BaselineSHA != f.Base || f.WeaveCalls != 2 {
		t.Fatalf("result %+v calls=%d", got, f.WeaveCalls)
	}
}

// TestRememberedSetupFailureRetriesWithAnUpgradedWeave: weave is a setup
// input, so a plain open after a weave upgrade compiles again rather than
// repeating the remembered failure.
func TestRememberedSetupFailureRetriesWithAnUpgradedWeave(t *testing.T) {
	f := newProvisionFixture(t)
	f.WeaveIdentity = "v1"
	f.FailWeave = true
	p := NewWorkspaceProvisioner(f)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	if _, err := p.Ensure(context.Background(), req); err == nil {
		t.Fatal("failed setup accepted")
	}
	if _, err := p.Ensure(context.Background(), req); err == nil || f.WeaveCalls != 1 {
		t.Fatalf("unchanged weave: err=%v calls=%d, want the remembered failure", err, f.WeaveCalls)
	}
	f.FailWeave = false
	f.WeaveIdentity = "v2"
	if _, err := p.Ensure(context.Background(), req); err != nil || f.WeaveCalls != 2 {
		t.Fatalf("upgraded weave: err=%v calls=%d, want a fresh compile", err, f.WeaveCalls)
	}
}

func TestOSProgramIdentityChangesWhenTheProgramIsReplaced(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	io := OSProvisionIO{}
	if got := io.ProgramIdentity("weave"); got != "unresolved" {
		t.Fatalf("missing program: %q", got)
	}
	bin := filepath.Join(dir, "weave")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	first := io.ProgramIdentity("weave")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho v2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if second := io.ProgramIdentity("weave"); second == first || !strings.HasPrefix(second, bin) {
		t.Fatalf("identity %q then %q", first, second)
	}
}

// TestProvisionHostAdoptsForeignDirectoryAndBranch: Couch no longer refuses a
// slot it did not itself create (pair#387 R1). A directory and a resting
// branch found at the slot's conventional place are adopted: the user's file
// survives and the branch is reused at its own commit.
func TestProvisionHostAdoptsForeignDirectoryAndBranch(t *testing.T) {
	for _, collision := range []string{"directory", "branch"} {
		t.Run(collision, func(t *testing.T) {
			f := newProvisionFixture(t)
			keep := filepath.Join(filepath.Dir(f.host(1)), "keep")
			var at string
			if collision == "directory" {
				os.MkdirAll(filepath.Dir(f.host(1)), 0700)
				os.WriteFile(keep, []byte("user"), 0600)
			} else {
				at = f.git(f.Primary, "commit-tree", "-m", "foreign", f.git(f.Primary, "rev-parse", "HEAD^{tree}"))
				f.git(f.Primary, "branch", "main-slot1", at)
			}
			if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
				t.Fatal(err)
			}
			if collision == "directory" {
				if raw, err := os.ReadFile(keep); err != nil || string(raw) != "user" {
					t.Fatalf("user file not preserved: %q %v", raw, err)
				}
			} else if got := f.git(f.Primary, "rev-parse", "main-slot1"); got != at {
				t.Fatalf("adopted branch moved to %s, want %s", got, at)
			}
		})
	}
}
func TestProvisionHostLostBranchAcknowledgment(t *testing.T) {
	f := newProvisionFixture(t)
	once := true
	f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if once && len(c.Args) > 0 && c.Args[0] == "update-ref" {
			once = false
			return errors.New("lost acknowledgment")
		}
		return nil
	}
	p := NewWorkspaceProvisioner(f)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	// The update-ref happened; only its acknowledgment was lost. The compare
	// and swap's probe finds the branch and adopts it in the same run.
	r, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if r.BaselineSHA != f.Base {
		t.Fatal(r)
	}
}
func TestParseFetchBaseline(t *testing.T) {
	sha := strings.Repeat("a", 40)
	other := strings.Repeat("b", 40)
	ref := "refs/remotes/upstream/main"
	good := "= " + sha + " " + other + " " + ref + "\n"
	got, err := ParseFetchBaseline([]byte(good), ref)
	if err != nil || got != other {
		t.Fatalf("%s %v", got, err)
	}
	for _, raw := range []string{"", good + good, "! " + sha + " " + other + " " + ref, "= bad " + other + " " + ref, strings.ReplaceAll(good, ref, ref+"-other")} {
		if _, err := ParseFetchBaseline([]byte(raw), ref); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestProvisionGitCaptureSurvivesExternalFetch(t *testing.T) {
	f := newProvisionFixture(t)
	once := true
	later := ""
	f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if once && len(c.Args) > 0 && c.Args[0] == "fetch" {
			once = false
			f.git(f.Primary, "commit", "--allow-empty", "-m", "remote advanced")
			later = f.git(f.Primary, "rev-parse", "HEAD")
			f.git(f.Primary, "push", "upstream", "main")
			f.git(f.Primary, "fetch", "upstream")
		}
		return nil
	}
	p := NewWorkspaceProvisioner(f)
	first, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.BaselineSHA != f.Base || f.git(first.Path, "rev-parse", "HEAD") != f.Base {
		t.Fatal("captured mutable tracking ref")
	}
	second, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 2})
	if err != nil {
		t.Fatal(err)
	}
	if second.BaselineSHA != later || later == f.Base {
		t.Fatalf("second=%+v later=%s", second, later)
	}
}
func TestProvisionGitRemoteSelection(t *testing.T) {
	for _, mode := range []string{"missing", "ambiguous", "configured", "explicit", "local-dot", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			f := newProvisionFixture(t)
			request := ProvisionRequest{Path: f.Primary, Slot: 1}
			wantSuccess := false
			switch mode {
			case "missing":
				f.git(f.Primary, "remote", "remove", "upstream")
			case "ambiguous":
				f.git(f.Primary, "remote", "add", "other", f.Remote)
			case "configured":
				f.git(f.Primary, "remote", "add", "other", f.Remote)
				f.git(f.Primary, "config", "branch.main.remote", "upstream")
				f.git(f.Primary, "config", "branch.main.merge", "refs/heads/main")
				wantSuccess = true
			case "explicit":
				f.git(f.Primary, "remote", "add", "other", f.Remote)
				request.Remote = "other"
				wantSuccess = true
			case "local-dot":
				f.git(f.Primary, "config", "branch.main.remote", ".")
				f.git(f.Primary, "config", "branch.main.merge", "refs/heads/main")
			case "duplicate":
				f.git(f.Primary, "config", "--add", "branch.main.remote", "upstream")
				f.git(f.Primary, "config", "--add", "branch.main.remote", "upstream")
			}
			_, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), request)
			if (err == nil) != wantSuccess {
				t.Fatalf("success=%v error=%v", wantSuccess, err)
			}
			if !wantSuccess && f.WeaveCalls != 0 {
				t.Fatal("setup after failed remote selection")
			}
		})
	}
}

// TestProvisionHostInvalidMarkerIsRecompiled: an invalid setup marker is a
// broken derived file (pair#387 R3): setup runs again and rewrites it.
func TestProvisionHostInvalidMarkerIsRecompiled(t *testing.T) {
	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	r, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	admin := f.git(r.Path, "rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(admin, "couch-setup-success.json"), []byte(`{"schema_version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Ensure(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if f.WeaveCalls != 2 {
		t.Fatalf("weave calls %d, want a recompile", f.WeaveCalls)
	}
	var marker SetupSuccess
	if exists, err := (ProvisionStore{}).Read(filepath.Join(admin, "couch-setup-success.json"), &marker); err != nil || !exists || marker.SchemaVersion != 1 {
		t.Fatalf("marker not rewritten: %+v %v", marker, err)
	}
}
