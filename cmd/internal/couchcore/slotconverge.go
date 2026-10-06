package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// slotConverger executes one converge step of a slot plan (pair#387). Every
// step re-checks its own precondition when it runs, so a stale plan or a
// repeated step is a no-op or a refusal, never a duplicate effect. It reuses
// the provisioner's git, fetch and remote helpers: there is one provisioner.
type slotConverger struct {
	p        *WorkspaceProvisioner
	layout   SlotLayout
	lease    *HostCreationLease
	remote   string
	progress io.Writer
	// agentNow re-reads the slot's agent right before a set-aside.
	agentNow func(context.Context) EvidenceAgent
	// rename is the set-aside move (the crash-injection seam; nil: os.Rename).
	rename func(oldpath, newpath string) error
}

func (cv *slotConverger) git(ctx context.Context, dir string, args ...string) (string, error) {
	return cv.p.git(ctx, dir, cv.lease, args...)
}

// converge runs one non-compile step. Compile runs outside the lease
// (compileSetup).
func (cv *slotConverger) converge(ctx context.Context, s PlannedStep) error {
	switch s.Step {
	case StepMkdirEnv:
		return cv.mkdirEnv()
	case StepRemoveIntent:
		return cv.removeIntent()
	case StepCreateBranch:
		return cv.createBranch(ctx)
	case StepSetUpstream:
		return cv.setUpstream(ctx)
	case StepRepairHost:
		// An attempt, judged by the next observation: git worktree repair
		// rewrites a broken .git file and still exits 1, reporting the
		// breakage it found (git 2.54, TestConvergeRepairHost). Its status is
		// evidence of nothing; a cancelled context still is.
		_, _ = cv.git(ctx, cv.layout.primary, "worktree", "repair", cv.layout.Host())
		return ctx.Err()
	case StepWorktreeAdd:
		return cv.worktreeAdd(ctx, s)
	case StepSetAside:
		return cv.setAside(ctx, s)
	}
	return fmt.Errorf("converge step %s is not executable here", s.Step)
}

func (cv *slotConverger) mkdirEnv() error {
	env := cv.layout.Env()
	if err := provisionSafePath(env); err != nil {
		return err
	}
	if err := provisionMkdirAll(filepath.Dir(env)); err != nil {
		return err
	}
	err := os.Mkdir(env, 0o700)
	if errors.Is(err, os.ErrExist) {
		if info, statErr := os.Lstat(env); statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil // already converged
		}
		return fmt.Errorf("%s exists and is not a directory", env)
	}
	return err
}

func (cv *slotConverger) removeIntent() error {
	path := cv.layout.Intent()
	if path == "" {
		return errors.New("git common directory unknown")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// createBranch creates main-slotN at remote main with a zero-OID compare and
// swap. A branch that appeared meanwhile is left alone: the next observation
// adopts it.
func (cv *slotConverger) createBranch(ctx context.Context) error {
	l := cv.layout
	if oid, err := cv.p.branchOID(ctx, l.primary, l.RestingBranch(), cv.lease); err != nil {
		return err
	} else if oid != "" {
		return nil
	}
	remote, err := cv.p.selectRemote(ctx, l.primary, cv.remote, cv.lease)
	if err != nil {
		return err
	}
	tracking := "refs/remotes/" + remote + "/main"
	raw, err := cv.p.command(ctx, l.primary, "git", []string{"fetch", "--verbose", "--porcelain", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--refmap=", remote, "+refs/heads/main:" + tracking}, cv.lease, cv.progress, cv.p.FetchTimeout)
	if err != nil {
		return fmt.Errorf("fetch remote main: %w", err)
	}
	baseline, err := ParseFetchBaseline(raw, tracking)
	if err != nil {
		return err
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	_, err = cv.git(ctx, l.primary, "update-ref", "--create-reflog", "-m", "couch-slot-create:"+hex.EncodeToString(token), l.RestingRef(), baseline, strings.Repeat("0", len(baseline)))
	if err != nil {
		if oid, probe := cv.p.branchOID(ctx, l.primary, l.RestingBranch(), cv.lease); probe == nil && oid != "" {
			return nil // created concurrently: adopted on the next observation
		}
	}
	return err
}

// setUpstream fills the missing half (or both) of the resting branch's
// upstream; a present value is never overwritten (a conflict is a hand-off,
// planned before this step can run).
func (cv *slotConverger) setUpstream(ctx context.Context) error {
	l := cv.layout
	rest := l.RestingBranch()
	remote, found, err := cv.p.configValue(ctx, l.primary, "branch."+rest+".remote", cv.lease)
	if err != nil {
		return err
	}
	if !found {
		if remote, err = cv.p.selectRemote(ctx, l.primary, cv.remote, cv.lease); err != nil {
			return err
		}
		if _, err := cv.git(ctx, l.primary, "config", "--add", "branch."+rest+".remote", remote); err != nil {
			return err
		}
	}
	if _, found, err = cv.p.configValue(ctx, l.primary, "branch."+rest+".merge", cv.lease); err != nil {
		return err
	} else if !found {
		if _, err := cv.git(ctx, l.primary, "config", "--add", "branch."+rest+".merge", "refs/heads/main"); err != nil {
			return err
		}
	}
	return nil
}

// worktreeAdd re-adds the host. --force overrides a stale registration of the
// same path (and the branch that registration still names). The planned
// branch is used only while it exists and no live worktree has it; otherwise
// the resting branch is, and the caller reports which.
func (cv *slotConverger) worktreeAdd(ctx context.Context, s PlannedStep) error {
	l := cv.layout
	if _, err := os.Lstat(l.Host()); err == nil {
		return nil // appeared since the observation: re-observed next pass
	}
	branch, err := cv.addBranch(ctx, s.Branch)
	if err != nil {
		return err
	}
	args := []string{"worktree", "add"}
	if s.Force {
		args = append(args, "--force")
	}
	args = append(args, l.Host(), branch)
	_, err = cv.git(ctx, l.primary, args...)
	return err
}

// addBranch decides which branch the re-added host checks out.
func (cv *slotConverger) addBranch(ctx context.Context, planned string) (string, error) {
	rest := cv.layout.RestingBranch()
	if planned == "" || planned == rest {
		return rest, nil
	}
	oid, err := cv.p.branchOID(ctx, cv.layout.primary, planned, cv.lease)
	if err != nil {
		return "", err
	}
	if oid == "" {
		return rest, nil
	}
	raw, err := cv.git(ctx, cv.layout.primary, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return "", err
	}
	for _, e := range parseWorktreeList(raw) {
		if e.Branch == planned && !e.Prunable && filepath.Clean(e.Path) != cv.layout.Host() {
			return rest, nil // checked out in a live worktree elsewhere
		}
	}
	return planned, nil
}

// compileSetup runs weave compile in the host without the lease, then retakes
// the lease and records setup success only if the host's registration is the
// one compiled. relock returns the retaken lease, which the caller owns.
func (cv *slotConverger) compileSetup(ctx context.Context, relock func() (*HostCreationLease, error)) error {
	l := cv.layout
	admin, err := cv.git(ctx, l.Host(), "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	if cv.progress != nil {
		fmt.Fprintf(cv.progress, "Preparing %s in %s\n", WorkspaceReference{Repo: l.repo(), Number: l.n}, l.Host())
	}
	if _, err := cv.p.command(ctx, l.Host(), "weave", []string{"compile"}, nil, cv.progress, cv.p.SetupTimeout); err != nil {
		// A hand-off failure is remembered (R5), written under the lease like
		// every slot record. Best effort: without it the next open recompiles.
		if f := ClassifyConvergeError(PlannedStep{Step: StepCompile, Resource: ResourceSetup}, err); f.Class == FailureHandoff && ctx.Err() == nil {
			if lease, lerr := relock(); lerr == nil {
				cv.lease = lease
				_ = rememberFailedSetup(ctx, cv.p, l, admin, f)
			}
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	lease, err := relock()
	if err != nil {
		return fmt.Errorf("setup completed but success is unconfirmed: %w", err)
	}
	cv.lease = lease
	after, err := cv.git(ctx, l.Host(), "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	if after != admin {
		return errors.New("workspace administrative directory changed during setup")
	}
	baseline, err := cv.git(ctx, l.primary, "rev-parse", "--verify", l.RestingRef()+"^{commit}")
	if err != nil {
		return err
	}
	if !validWorkspaceOID(baseline) {
		return errors.New("invalid resting branch baseline")
	}
	// A marker a concurrent caller already published for this registration
	// wins: it is never rewritten.
	var published SetupSuccess
	if exists, err := cv.p.Store.Read(SetupMarkerPath(admin), &published); err == nil && exists && ValidSetupMarker(published, l.Host(), l.common, admin, l.n) {
		return cv.clearSetupMemo(admin)
	}
	marker := SetupSuccess{SchemaVersion: 1, Host: l.Host(), Common: l.common, Admin: admin, Slot: l.n, BaselineSHA: baseline}
	if err := cv.p.Store.Write(SetupMarkerPath(admin), marker); err != nil {
		return fmt.Errorf("record setup success: %w", err)
	}
	return cv.clearSetupMemo(admin)
}

func (cv *slotConverger) clearSetupMemo(admin string) error {
	if _, err := os.Lstat(SetupAttemptPath(admin)); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := cv.p.Store.Remove(SetupAttemptPath(admin)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear the remembered setup failure: %w", err)
	}
	return nil
}
