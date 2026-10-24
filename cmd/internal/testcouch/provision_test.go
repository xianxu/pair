package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
)

type provisionFixture struct {
	t               *testing.T
	b               Binaries
	p               RootsPlan
	env             []string
	commands        []string
	failSlot        int
	badIdentity     bool
	badMarker       bool
	cancel          context.CancelFunc
	adopted         bool
	adoptionFailure string
}

func newWorkspaceFixture(t *testing.T) *provisionFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git, err := resolveTool("git")
	if err != nil {
		t.Fatal(err)
	}
	return &provisionFixture{t: t, b: Binaries{Git: git, Couch: "/candidate/couch", SDLC: "/installed/sdlc"}, p: RootsPlan{Root: root, Mutable: map[string]string{"fleet": filepath.Join(root, "fleet"), "store": filepath.Join(root, "store"), "data": filepath.Join(root, "data"), "identity": filepath.Join(root, "identity")}}, env: []string{"PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid"}}
}
func (f *provisionFixture) git(ctx context.Context, dir string, args ...string) string {
	f.t.Helper()
	out, err := runProvisionCommand(ctx, dir, f.env, f.b.Git, args...)
	if err != nil {
		f.t.Fatal(err)
	}
	return strings.TrimSpace(out)
}
func (f *provisionFixture) run(ctx context.Context, dir string, env []string, program string, args ...string) (string, error) {
	if strings.Join(env, "\x00") != strings.Join(f.env, "\x00") {
		return "", errors.New("command lost explicit environment")
	}
	f.commands = append(f.commands, filepath.Base(program)+" "+strings.Join(args, " "))
	switch program {
	case f.b.Couch:
		if args[0] == "--adopt-store" {
			roots := couchsingleton.Roots{StoreDir: f.p.Mutable["store"], PairDataDir: filepath.Join(f.p.Mutable["data"], "pair"), IdentityDir: f.p.Mutable["identity"]}
			want := []string{"--adopt-store", roots.StoreDir, "--pair-data", roots.PairDataDir, "--identity-dir", roots.IdentityDir}
			if len(args) < 6 || strings.Join(args[:6], "\x00") != strings.Join(want, "\x00") {
				return "", errors.New("adoption roots mismatch")
			}
			digest := strings.Repeat("a", 64)
			if len(args) == 6 {
				report := couchsingleton.Report{Digest: digest, Selection: couchsingleton.Selection{Version: 1, Roots: roots}, Status: "ADOPTABLE"}
				if f.adoptionFailure == "blocker" {
					report.Blockers = []string{"fixture ambiguity"}
				}
				if f.adoptionFailure == "digest" {
					report.Digest = "invalid"
				}
				raw, _ := json.Marshal(report)
				if len(report.Blockers) > 0 {
					return string(raw), errors.New("UNMIGRATED")
				}
				return string(raw), nil
			}
			if len(args) != 8 || args[6] != "--apply" || args[7] != digest {
				return "", errors.New("wrong preview digest applied")
			}
			if f.adoptionFailure == "apply" {
				return "", errors.New("adoption failed")
			}
			selected := couchsingleton.Selection{Version: 1, Roots: roots}
			if f.adoptionFailure == "selection" {
				selected.Roots.StoreDir = filepath.Join(f.p.Root, "other")
			}
			raw, _ := json.Marshal(struct {
				Status    string                   `json:"status"`
				Selection couchsingleton.Selection `json:"selection"`
			}{"SELECTED", selected})
			f.adopted = true
			return string(raw), nil
		}
		if !f.adopted {
			return "", errors.New("Couch is UNMIGRATED: adoption required")
		}
		n, _ := strconv.Atoi(strings.TrimPrefix(args[3], "--slot="))
		if n == f.failSlot {
			return "", errors.New("fixture setup failed")
		}
		primary := args[2]
		host := filepath.Join(filepath.Dir(primary), "worktree", fmt.Sprintf("fixture-slot%d", n), "fixture")
		f.git(ctx, primary, "worktree", "add", "-b", fmt.Sprintf("main-slot%d", n), host, "upstream/main")
		admin := f.git(ctx, host, "rev-parse", "--absolute-git-dir")
		marker := couchcore.SetupSuccess{SchemaVersion: 1, Host: host, Common: filepath.Join(primary, ".git"), Admin: admin, Slot: n, BaselineSHA: f.git(ctx, host, "rev-parse", "HEAD")}
		if f.badMarker {
			marker.Slot++
		}
		raw, _ := json.Marshal(marker)
		if err := os.WriteFile(filepath.Join(admin, "couch-setup-success.json"), raw, 0600); err != nil {
			return "", err
		}
		result, _ := json.Marshal(couchcore.ProvisionResult{SchemaVersion: 1, Address: fmt.Sprintf("fixture:%d", n), Path: host, RestingBranch: fmt.Sprintf("main-slot%d", n), BaselineSHA: marker.BaselineSHA, Disposition: "created"})
		if f.cancel != nil {
			f.cancel()
		}
		return string(result), nil
	case f.b.SDLC:
		n, _ := strconv.Atoi(strings.TrimPrefix(filepath.Base(filepath.Dir(dir)), "fixture-slot"))
		primary := filepath.Join(f.p.Mutable["fleet"], "fixture")
		address, branch, head, rest := fmt.Sprintf("fixture:%d", n), fmt.Sprintf("main-slot%d", n), f.git(ctx, dir, "rev-parse", "HEAD"), fmt.Sprintf("main-slot%d", n)
		id := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "fixture", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: filepath.Dir(primary), EnvironmentRoot: filepath.Dir(dir), WorktreeRoot: dir, Kind: "slot", Address: &address, Slot: &n, Branch: &branch, Head: &head, RestingBranch: &rest, EnvironmentHost: &couchcore.EnvironmentHost{Repo: "fixture", Slot: n, RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, WorktreeRoot: dir}}
		if f.badIdentity {
			id.Kind = "primary"
		}
		raw, _ := json.Marshal(id)
		return string(raw), nil
	default:
		return runProvisionCommand(ctx, dir, env, program, args...)
	}
}
func TestPrepareWorkspacesRealGitTopologyAndValidatedEvidence(t *testing.T) {
	f := newWorkspaceFixture(t)
	w, err := prepareWorkspaces(context.Background(), f.b, f.p, f.env, 2, f.run)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Workers) != 2 || w.Primary != filepath.Join(f.p.Mutable["fleet"], "fixture") || w.Evidence["baseline"] == "" {
		t.Fatalf("workspaces: %+v", w)
	}
	for n, path := range w.Workers {
		if f.git(context.Background(), path, "rev-parse", "--abbrev-ref", "@{upstream}") != "upstream/main" {
			t.Fatalf("worker %d upstream", n)
		}
		if w.Evidence[fmt.Sprintf("slot-%d-identity", n)] == "" || w.Evidence[fmt.Sprintf("slot-%d-setup", n)] == "" {
			t.Fatalf("worker %d lacks observations", n)
		}
	}
	for _, name := range []string{"Makefile", "Brewfile"} {
		if _, err := os.Stat(filepath.Join(w.Primary, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected setup input %s: %v", name, err)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(w.Primary, "construct", "base.manifest"))
	if err != nil || string(manifest) != "# fixture\n" {
		t.Fatalf("manifest %q %v", manifest, err)
	}
	all := strings.Join(f.commands, "\n")
	if strings.Count(all, "couch --internal provision-workspace") != 2 || strings.Count(all, "sdlc workspace --json") != 2 || !strings.Contains(all, "clone --bare") || !strings.Contains(all, "push -u upstream main") {
		t.Fatalf("missing genuine adapter commands:\n%s", all)
	}
}
func TestPrepareWorkspacesRefusesBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"negative", "too-many", "cancelled", "escape", "missing-fleet", "nil-env", "relative-tool"} {
		t.Run(scenario, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			workers := 1
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "negative":
				workers = -1
			case "too-many":
				workers = 20
			case "cancelled":
				cancel()
			case "escape":
				f.p.Mutable["fleet"] = filepath.Join(filepath.Dir(f.p.Root), "foreign")
			case "missing-fleet":
				delete(f.p.Mutable, "fleet")
			case "nil-env":
				f.env = nil
			case "relative-tool":
				f.b.Couch = "couch"
			}
			if _, err := prepareWorkspaces(ctx, f.b, f.p, f.env, workers, f.run); err == nil {
				t.Fatal("invalid provisioning accepted")
			}
			entries, err := os.ReadDir(f.p.Root)
			if err != nil || len(entries) != 0 || len(f.commands) != 0 {
				t.Fatalf("refusal caused effects: files=%v commands=%v err=%v", entries, f.commands, err)
			}
		})
	}
}
func TestPrepareWorkspacesStopsOnFailureAndCancellation(t *testing.T) {
	for _, scenario := range []string{"setup", "identity", "marker", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "setup":
				f.failSlot = 2
			case "identity":
				f.badIdentity = true
			case "marker":
				f.badMarker = true
			case "cancel":
				f.cancel = cancel
			}
			w, err := prepareWorkspaces(ctx, f.b, f.p, f.env, 3, f.run)
			if err == nil {
				t.Fatal("failed observation accepted")
			}
			want := 0
			if scenario == "setup" {
				want = 1
			}
			if len(w.Workers) != want {
				t.Fatalf("partial success = %+v", w)
			}
			for _, command := range f.commands {
				if strings.Contains(command, "--slot=3") || (scenario != "setup" && strings.Contains(command, "--slot=2")) {
					t.Fatalf("started after failure: %v", f.commands)
				}
			}
		})
	}
}

func TestProvisionCommandUsesOnlyExplicitEnvironmentAndSeparateDiagnostics(t *testing.T) {
	t.Setenv("PAIR_TESTCOUCH_AMBIENT_SENTINEL", "must-not-cross")
	out, err := runProvisionCommand(context.Background(), t.TempDir(), []string{"OWNED_VALUE=private"}, "/bin/sh", "-c", `printf '%s:%s' "$OWNED_VALUE" "${PAIR_TESTCOUCH_AMBIENT_SENTINEL-unset}"; printf 'diagnostic' >&2`)
	if err != nil || out != "private:unset" {
		t.Fatalf("command environment/output: %q %v", out, err)
	}
	out, err = runProvisionCommand(context.Background(), t.TempDir(), nil, "/bin/sh", "-c", `printf '%s' "${PAIR_TESTCOUCH_AMBIENT_SENTINEL-unset}"`)
	if err != nil || out != "unset" {
		t.Fatalf("nil command env inherited ambient state: %q %v", out, err)
	}
	_, err = runProvisionCommand(context.Background(), t.TempDir(), []string{}, "/bin/sh", "-c", "printf '%070000d' 0 >&2; exit 9")
	if err == nil || len(err.Error()) > buildOutputLimit+1000 || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("unbounded command error: %v", err)
	}
}

func TestPrepareWorkspacesRefusesSymlinkEscapeAndExistingDestinations(t *testing.T) {
	for _, scenario := range []string{"symlink-fleet", "existing-primary", "existing-remote"} {
		t.Run(scenario, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			if scenario == "symlink-fleet" {
				if err := os.Symlink(t.TempDir(), f.p.Mutable["fleet"]); err != nil {
					t.Fatal(err)
				}
			} else {
				name := "fixture"
				if scenario == "existing-remote" {
					name = "upstream.git"
				}
				if err := os.MkdirAll(filepath.Join(f.p.Mutable["fleet"], name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := prepareWorkspaces(context.Background(), f.b, f.p, f.env, 1, f.run); err == nil {
				t.Fatal("unsafe destination accepted")
			}
			if len(f.commands) != 0 {
				t.Fatalf("unsafe destination started commands: %v", f.commands)
			}
		})
	}
}

func TestPrepareWorkspacesBootstrapsOnlyThroughPublicAdoption(t *testing.T) {
	f := newWorkspaceFixture(t)
	w, err := prepareWorkspaces(context.Background(), f.b, f.p, f.env, 1, f.run)
	if err != nil {
		t.Fatal(err)
	}
	if !f.adopted || w.Evidence["adoption-digest"] != strings.Repeat("a", 64) {
		t.Fatalf("missing adoption evidence: %+v", w)
	}
	all := strings.Join(f.commands, "\n")
	preview := strings.Index(all, "couch --adopt-store")
	apply := strings.Index(all, "--apply "+strings.Repeat("a", 64))
	provision := strings.Index(all, "couch --internal provision-workspace")
	if preview < 0 || apply < preview || provision < apply || strings.Count(all, "couch --adopt-store") != 2 || strings.Contains(all, "--json --apply") {
		t.Fatalf("bootstrap order: %s", all)
	}
}
func TestPrepareWorkspacesAdoptionFailureStartsNoSlots(t *testing.T) {
	for _, failure := range []string{"blocker", "digest", "apply", "selection"} {
		t.Run(failure, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			f.adoptionFailure = failure
			w, err := prepareWorkspaces(context.Background(), f.b, f.p, f.env, 2, f.run)
			if err == nil || len(w.Workers) != 0 {
				t.Fatalf("adoption failure accepted: %+v %v", w, err)
			}
			for _, command := range f.commands {
				if strings.Contains(command, "--internal provision-workspace") {
					t.Fatalf("slot started before adoption: %v", f.commands)
				}
				if (failure == "blocker" || failure == "digest") && strings.Contains(command, "--apply") {
					t.Fatalf("invalid preview applied: %v", f.commands)
				}
			}
		})
	}
}

func TestPrepareWorkspacesStewardOnly(t *testing.T) {
	f := newWorkspaceFixture(t)
	w, err := prepareWorkspaces(t.Context(), f.b, f.p, f.env, 0, f.run)
	if err != nil || w.Primary == "" || len(w.Workers) != 0 {
		t.Fatalf("workspaces=%+v err=%v", w, err)
	}
}
