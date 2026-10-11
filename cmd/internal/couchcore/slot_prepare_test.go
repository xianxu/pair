package couchcore

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// prepareWorld is a slot over real git: an origin a commit ahead of the slot's
// resting branch, and one declared substrate beside the host.
type prepareWorld struct {
	host, substrate string
}

func prepareGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writePrepareFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newPrepareWorld(t *testing.T) prepareWorld {
	t.Helper()
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t", "GIT_CONFIG_GLOBAL": os.DevNull} {
		t.Setenv(k, v)
	}
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	prepareGit(t, root, "init", "-q", "-b", "main", seed)
	writePrepareFile(t, filepath.Join(seed, "construct", "deps"), "substrate ../ariadne\n")
	writePrepareFile(t, filepath.Join(seed, ".gitattributes"), "*.raw binary\n")
	prepareGit(t, seed, "add", "-A")
	prepareGit(t, seed, "commit", "-q", "-m", "one")
	origin := filepath.Join(root, "pair.git")
	prepareGit(t, root, "clone", "-q", "--bare", seed, origin)
	env := filepath.Join(root, "pair-slot1")
	w := prepareWorld{host: filepath.Join(env, "pair"), substrate: filepath.Join(env, "ariadne")}
	prepareGit(t, root, "clone", "-q", origin, w.host)
	prepareGit(t, w.host, "checkout", "-q", "-b", "main-slot1", "--track", "origin/main")
	// origin moves on: the slot is stale.
	writePrepareFile(t, filepath.Join(seed, "two"), "2\n")
	prepareGit(t, seed, "add", "-A")
	prepareGit(t, seed, "commit", "-q", "-m", "two")
	prepareGit(t, seed, "push", "-q", origin, "main")
	prepareGit(t, root, "init", "-q", "-b", "main", w.substrate)
	writePrepareFile(t, filepath.Join(w.substrate, "construct", "base.manifest"), "")
	prepareGit(t, w.substrate, "add", "-A")
	prepareGit(t, w.substrate, "commit", "-q", "-m", "substrate old")
	return w
}

// fakeWeave is weave behind the ProvisionIO seam: `weave refresh`
// fast-forwards the host and advances the substrate, as the real refresh does;
// every other command runs for real.
type fakeWeave struct {
	t       *testing.T
	w       prepareWorld
	refresh []string // the directories refresh ran in
	dirty   bool     // compile rewrites the tracked .gitattributes
	fail    error
	realGit OSProvisionIO
}

func (f *fakeWeave) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	if c.Program != "weave" {
		return f.realGit.Run(ctx, c)
	}
	f.refresh = append(f.refresh, c.Dir)
	if f.fail != nil {
		return nil, f.fail
	}
	prepareGit(f.t, c.Dir, "fetch", "-q", "origin")
	prepareGit(f.t, c.Dir, "merge", "-q", "--ff-only", "origin/main")
	writePrepareFile(f.t, filepath.Join(f.w.substrate, "new"), "x\n")
	prepareGit(f.t, f.w.substrate, "add", "-A")
	prepareGit(f.t, f.w.substrate, "commit", "-q", "-m", "substrate fresh")
	if f.dirty {
		writePrepareFile(f.t, filepath.Join(c.Dir, ".gitattributes"), "*.raw binary\nworkshop/lessons.md merge=union\n")
	}
	return nil, nil
}

type prepareRig struct {
	w       prepareWorld
	weave   *fakeWeave
	reloads int
	run     PrepareRun
}

func newPrepareRig(t *testing.T) *prepareRig {
	w := newPrepareWorld(t)
	r := &prepareRig{w: w, weave: &fakeWeave{t: t, w: w}}
	r.run = PrepareRun{Target: "pair:1", IO: r.weave,
		Admit: func(context.Context) (PrepareAdmission, error) {
			return PrepareAdmission{Address: ThreadAddress{RepoScope: "s", Tag: "t1"}, Path: w.host}, nil
		},
		Reload: func(context.Context) error { r.reloads++; return nil },
	}
	return r
}

func stepStatuses(r PrepareReport) string {
	var out []string
	for _, s := range r.Steps {
		out = append(out, s.Name+"="+s.Status)
	}
	return strings.Join(out, " ")
}

// Done-when 1 and 3: a stale, clean, idle slot ends at origin/main with a
// refreshed substrate and a reloaded agent, and the report names the
// substrate's new commit (the one the slot's sdlc builds from).
func TestRunPrepareBringsStaleSlotUp(t *testing.T) {
	r := newPrepareRig(t)
	report := RunPrepare(context.Background(), r.run)
	if !report.OK() || stepStatuses(report) != "admit=ok refresh=ok recheck=ok reload=ok" {
		t.Fatalf("%+v", report)
	}
	if len(r.weave.refresh) != 1 || r.weave.refresh[0] != r.w.host || r.reloads != 1 {
		t.Fatalf("refresh %v reloads %d", r.weave.refresh, r.reloads)
	}
	origin := prepareGit(t, r.w.host, "rev-parse", "origin/main")
	if head := prepareGit(t, r.w.host, "rev-parse", "HEAD"); head != origin || report.Head != head || report.OriginMain != origin {
		t.Fatalf("head %s origin %s report %+v", head, origin, report)
	}
	fresh := prepareGit(t, r.w.substrate, "rev-parse", "HEAD")
	if len(report.Substrates) != 1 || report.Substrates[0].Commit != fresh || report.Substrates[0].Subject != "substrate fresh" {
		t.Fatalf("substrates %+v, want %s", report.Substrates, fresh)
	}
}

// Done-when 2: a refused admission (busy, dirty, off its resting branch)
// changes nothing: no refresh, no reload, the checkout where it was.
func TestRunPrepareRefusedAdmissionChangesNothing(t *testing.T) {
	for _, code := range []string{LiveRestartBusy, LiveRestartDirty, LiveRestartNotResting} {
		r := newPrepareRig(t)
		before := prepareGit(t, r.w.host, "rev-parse", "HEAD")
		r.run.Admit = func(context.Context) (PrepareAdmission, error) {
			return PrepareAdmission{}, &SlotOperationError{Code: code, Detail: "no"}
		}
		report := RunPrepare(context.Background(), r.run)
		if report.OK() || report.Steps[0].Code != code || stepStatuses(report) != "admit=refused refresh=skipped recheck=skipped reload=skipped" {
			t.Fatalf("%s: %+v", code, report)
		}
		if len(r.weave.refresh) != 0 || r.reloads != 0 || prepareGit(t, r.w.host, "rev-parse", "HEAD") != before {
			t.Fatalf("%s: changed the slot: refresh %v reloads %d", code, r.weave.refresh, r.reloads)
		}
	}
}

// The ariadne#320 trap: compile writes its block into the tracked
// .gitattributes, so the refresh dirties its own checkout. Prepare stops before
// the reload and names the file; it never commits it.
func TestRunPrepareRefreshDirtyStopsBeforeReload(t *testing.T) {
	r := newPrepareRig(t)
	r.weave.dirty = true
	report := RunPrepare(context.Background(), r.run)
	recheck := report.Steps[2]
	if recheck.Code != PrepareRefreshDirty || !strings.Contains(recheck.Detail, ".gitattributes") || stepStatuses(report) != "admit=ok refresh=ok recheck=refused reload=skipped" {
		t.Fatalf("%+v", report)
	}
	if r.reloads != 0 {
		t.Fatal("reloaded onto a dirty checkout")
	}
	if got := prepareGit(t, r.w.host, "status", "--porcelain"); !strings.Contains(got, ".gitattributes") {
		t.Fatalf("prepare touched the refresh's output: %q", got)
	}
}

// The recheck's dirty predicate is the admission's: an untracked file the
// refresh leaves behind is refresh-dirty here, never a reload refused dirty.
func TestRunPrepareUntrackedOutputIsRefreshDirty(t *testing.T) {
	r := newPrepareRig(t)
	r.run.IO = untrackedWeave{r.weave}
	report := RunPrepare(context.Background(), r.run)
	if report.Steps[2].Code != PrepareRefreshDirty || !strings.Contains(report.Steps[2].Detail, "generated.txt") || r.reloads != 0 {
		t.Fatalf("%+v reloads %d", report, r.reloads)
	}
}

type untrackedWeave struct{ *fakeWeave }

func (u untrackedWeave) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	out, err := u.fakeWeave.Run(ctx, c)
	if c.Program == "weave" && err == nil {
		writePrepareFile(u.t, filepath.Join(c.Dir, "generated.txt"), "x\n")
	}
	return out, err
}

func TestRunPrepareRefreshFailureSkipsReload(t *testing.T) {
	r := newPrepareRig(t)
	r.weave.fail = errors.New("weave: checkouts must be clean")
	report := RunPrepare(context.Background(), r.run)
	if stepStatuses(report) != "admit=ok refresh=failed recheck=skipped reload=skipped" || !strings.Contains(report.Steps[1].Detail, "must be clean") || r.reloads != 0 {
		t.Fatalf("%+v reloads %d", report, r.reloads)
	}
}

// A turn that starts during the refresh: the reload's own admission refuses,
// and the report says the slot was refreshed but not reloaded.
func TestRunPrepareReloadRefusalIsReported(t *testing.T) {
	r := newPrepareRig(t)
	r.run.Reload = func(context.Context) error {
		return &SlotOperationError{Code: LiveRestartBusy, Detail: "pair:1: a turn is open"}
	}
	report := RunPrepare(context.Background(), r.run)
	if stepStatuses(report) != "admit=ok refresh=ok recheck=ok reload=refused" || report.Steps[3].Code != LiveRestartBusy {
		t.Fatalf("%+v", report)
	}
}
