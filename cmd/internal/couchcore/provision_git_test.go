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
	AfterGit              func(ProvisionCommand, []byte) error
}

func newProvisionFixture(t *testing.T) *ProvisionFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &ProvisionFixture{t: t, Primary: filepath.Join(root, "fleet space", "repo-name"), Remote: filepath.Join(root, "remote.git")}
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
	return filepath.Join(filepath.Dir(f.Primary), "worktree", fmt.Sprintf("repo-name-slot%d", n), "repo-name")
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
		common := filepath.Join(f.Primary, ".git")
		addr := fmt.Sprintf("repo-name:%d", slot)
		head := f.git(top, "rev-parse", "HEAD")
		branch := f.git(top, "branch", "--show-current")
		id := WorkspaceIdentity{SchemaVersion: 2, Repo: "repo-name", RepoIdentity: common, PrimaryRoot: f.Primary, FleetRoot: filepath.Dir(f.Primary), EnvironmentRoot: env, WorktreeRoot: top, Kind: kind, Address: &addr, Slot: &slot, Branch: &branch, Head: &head, RestingBranch: &rest}
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
	got, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.BaselineSHA != f.Base || f.WeaveCalls != 2 {
		t.Fatalf("result %+v calls=%d", got, f.WeaveCalls)
	}
}
func TestProvisionHostRefusesForeignPathOrBranch(t *testing.T) {
	for _, collision := range []string{"directory", "branch"} {
		t.Run(collision, func(t *testing.T) {
			f := newProvisionFixture(t)
			if collision == "directory" {
				os.MkdirAll(filepath.Dir(f.host(1)), 0700)
				os.WriteFile(filepath.Join(filepath.Dir(f.host(1)), "keep"), []byte("user"), 0600)
			} else {
				f.git(f.Primary, "branch", "main-slot1", f.Base)
			}
			if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err == nil {
				t.Fatal("foreign collision accepted")
			}
			if f.WeaveCalls != 0 {
				t.Fatal("compiled on collision")
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
	if _, err := p.Ensure(context.Background(), req); err == nil {
		t.Fatal("lost acknowledgment accepted")
	}
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
func TestProvisionHostInvalidSuccessRefusesWithoutCompile(t *testing.T) {
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
	if _, err := p.Ensure(context.Background(), req); err == nil {
		t.Fatal("invalid marker accepted")
	}
	if f.WeaveCalls != 1 {
		t.Fatal("compiled despite corrupt marker")
	}
}
