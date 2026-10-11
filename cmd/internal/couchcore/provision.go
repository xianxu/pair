package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WorkspaceProvisioner budgets each command by its class (pair#442): a
// read-only probe gets ProbeTimeout, a git step that writes a checkout
// (worktree add, worktree repair) WriteTimeout, a fetch FetchTimeout, and
// weave compile SetupTimeout.
type WorkspaceProvisioner struct {
	IO                                                     ProvisionIO
	Store                                                  ProvisionStorage
	ProbeTimeout, WriteTimeout, FetchTimeout, SetupTimeout time.Duration
}

func NewWorkspaceProvisioner(commandIO ProvisionIO) *WorkspaceProvisioner {
	return &WorkspaceProvisioner{IO: commandIO, Store: ProvisionStore{}, ProbeTimeout: 5 * time.Second, WriteTimeout: 2 * time.Minute, FetchTimeout: 120 * time.Second, SetupTimeout: 20 * time.Minute}
}

// ProvisionCommandError names the command that failed and its budget, so a
// failure report says which command ran out of time (pair#442). Err keeps
// the cause (with its stderr tail) for errors.Is/As.
type ProvisionCommandError struct {
	Command  string
	Timeout  time.Duration
	TimedOut bool // the command's own budget expired, not the caller's context
	Err      error
}

func (e *ProvisionCommandError) Error() string {
	if e.TimedOut {
		return fmt.Sprintf("%s: timed out after %s: %v", e.Command, e.Timeout, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Command, e.Err)
}
func (e *ProvisionCommandError) Unwrap() error { return e.Err }

type provisionHost struct {
	identity        WorkspaceIdentity
	admin, baseline string
}

func (p *WorkspaceProvisioner) command(ctx context.Context, dir, program string, args []string, lease *HostCreationLease, progress io.Writer, timeout time.Duration) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := ProvisionCommand{Dir: dir, Program: program, Args: args, Timeout: timeout, Progress: progress, StreamOutput: program == "weave"}
	if lease != nil && program == "git" {
		c.Lease = lease.File()
	}
	out, err := p.IO.Run(ctx, c)
	if err != nil {
		err = &ProvisionCommandError{Command: strings.Join(append([]string{program}, args...), " "), Timeout: timeout,
			TimedOut: timeout > 0 && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded), Err: err}
	}
	return out, err
}
func (p *WorkspaceProvisioner) git(ctx context.Context, dir string, lease *HostCreationLease, args ...string) (string, error) {
	return p.gitWithin(ctx, p.ProbeTimeout, dir, lease, args...)
}

// gitWrite runs a git step that writes a checkout: it gets the write budget.
func (p *WorkspaceProvisioner) gitWrite(ctx context.Context, dir string, lease *HostCreationLease, args ...string) (string, error) {
	return p.gitWithin(ctx, p.WriteTimeout, dir, lease, args...)
}
func (p *WorkspaceProvisioner) gitWithin(ctx context.Context, timeout time.Duration, dir string, lease *HostCreationLease, args ...string) (string, error) {
	raw, err := p.command(ctx, dir, "git", args, lease, nil, timeout)
	return strings.TrimSuffix(string(raw), "\n"), err
}
func (p *WorkspaceProvisioner) identity(ctx context.Context, path string) (WorkspaceIdentity, error) {
	raw, err := p.command(ctx, path, "sdlc", []string{"workspace", "--json"}, nil, nil, p.ProbeTimeout)
	if err != nil {
		return WorkspaceIdentity{}, fmt.Errorf("resolve workspace: %w", err)
	}
	return ParseWorkspaceIdentity(raw)
}

// Ensure converges slot N of the repository to a working slot (pair#387): its
// body is the slot reconciler, so creating a slot, repairing one after a
// crash or an interrupted setup, and re-adding a deleted one are the same
// operation. Repeating it after any interruption finishes what is provably
// missing. A blocking outcome is a *SlotReconcileError (its text is the
// operator advice); a degraded one returns the slot with Warning set.
func (p *WorkspaceProvisioner) Ensure(ctx context.Context, req ProvisionRequest) (ProvisionResult, error) {
	if ctx == nil {
		return ProvisionResult{}, errors.New("provision requires a context")
	}
	if p == nil || p.IO == nil || p.Store == nil {
		return ProvisionResult{}, errors.New("workspace provisioning is unavailable")
	}
	if _, err := ParseProvisionRequest(req.Path, strconv.Itoa(req.Slot), req.Remote); err != nil {
		return ProvisionResult{}, err
	}
	physical, err := filepath.EvalSymlinks(NormalizePath(req.Path))
	if err != nil {
		return ProvisionResult{}, err
	}
	primary, err := p.identity(ctx, physical)
	if err != nil {
		return ProvisionResult{}, err
	}
	if primary.Kind != "primary" || primary.WorktreeRoot != primary.PrimaryRoot {
		return ProvisionResult{}, errors.New("provision path must identify the primary checkout")
	}
	layout := NewSlotLayout(primary.PrimaryRoot, primary.RepoIdentity, req.Slot)
	if err := provisionSafePath(layout.Host()); err != nil {
		return ProvisionResult{}, err
	}
	_, hostErr := os.Lstat(layout.Host())
	existed := hostErr == nil
	result, runErr := p.Reconcile(ctx, ReconcileRequest{Layout: layout, Agent: req.Agent, Remote: req.Remote, Progress: req.Progress,
		AgentNow: req.AgentNow, IgnoreMemo: req.IgnoreMemo})
	address := WorkspaceReference{Repo: primary.Repo, Number: req.Slot}.String()
	blocking, warnings := SlotOutcome(address, primary.Repo, result, runErr)
	setAside := SavedWorkManifests(result.SetAside)
	if blocking != nil {
		blocking.SetAside = setAside
		if err := p.recordSlotFailure(layout, blocking); err != nil {
			blocking.RecordErr = err
		}
		return ProvisionResult{}, blocking
	}
	host, err := p.verifyHost(ctx, primary, req.Slot, nil)
	if err != nil {
		return ProvisionResult{}, err
	}
	var marker SetupSuccess
	if _, err := p.Store.Read(SetupMarkerPath(host.admin), &marker); err != nil {
		return ProvisionResult{}, err
	}
	host.baseline = marker.BaselineSHA
	if err := p.clearSlotFailure(layout); err != nil { // converged and verified
		warnings = append(warnings, fmt.Sprintf("slot %s: its last failure record could not be cleared: %v", address, err))
	}
	disposition := "reused"
	switch {
	case !existed:
		disposition = "created"
	case executedStep(result, StepCompile):
		disposition = "prepared"
	}
	out := provisionResult(host, disposition)
	if len(setAside) > 0 {
		out.SetAside = setAside
		for _, m := range out.SetAside {
			warnings = append(warnings, fmt.Sprintf("slot %s: set aside %s (it could not be read); restore with: %s", address, m.Path, m.Restore))
		}
	}
	out.Warning = strings.Join(warnings, "\n")
	out.Report = &SlotReport{Address: address, Observation: result.Observation, Plan: result.Plan}
	return out, nil
}

func executedStep(r ReconcileResult, step ConvergeStep) bool {
	for _, s := range r.Executed {
		if s.Step == step {
			return true
		}
	}
	return false
}

func provisionResult(h provisionHost, disposition string) ProvisionResult {
	return ProvisionResult{SchemaVersion: 1, Address: *h.identity.Address, Path: h.identity.WorktreeRoot, RestingBranch: *h.identity.RestingBranch, BaselineSHA: h.baseline, Disposition: disposition}
}

func (p *WorkspaceProvisioner) verifyHost(ctx context.Context, primary WorkspaceIdentity, slot int, lease *HostCreationLease) (provisionHost, error) {
	path := NewSlotLayout(primary.PrimaryRoot, primary.RepoIdentity, slot).Host()
	if err := provisionSafePath(path); err != nil {
		return provisionHost{}, err
	}
	id, err := p.identity(ctx, path)
	if err != nil {
		return provisionHost{}, err
	}
	if id.Kind != "slot" || id.Slot == nil || *id.Slot != slot || id.WorktreeRoot != path || id.PrimaryRoot != primary.PrimaryRoot || id.RepoIdentity != primary.RepoIdentity {
		return provisionHost{}, errors.New("workspace does not match expected slot identity")
	}
	admin, err := p.git(ctx, path, lease, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return provisionHost{}, err
	}
	if filepath.Dir(admin) != NewSlotLayout(primary.PrimaryRoot, primary.RepoIdentity, slot).Registrations() {
		return provisionHost{}, errors.New("unexpected slot Git administrative directory")
	}
	if err := provisionSafePath(admin); err != nil {
		return provisionHost{}, err
	}
	common, err := p.git(ctx, path, lease, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return provisionHost{}, err
	}
	if filepath.Clean(common) != primary.RepoIdentity {
		return provisionHost{}, errors.New("slot Git common directory mismatch")
	}
	return provisionHost{identity: id, admin: admin}, nil
}
func (p *WorkspaceProvisioner) branchOID(ctx context.Context, dir, branch string, lease *HostCreationLease) (string, error) {
	ref := "refs/heads/" + branch
	out, err := p.git(ctx, dir, lease, "for-each-ref", "--format=%(refname) %(objectname)", ref)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", nil
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !validWorkspaceOID(fields[1]) {
			return "", errors.New("invalid branch ref output")
		}
		if fields[0] == ref {
			return fields[1], nil
		}
	}
	return "", nil
}
func (p *WorkspaceProvisioner) configValue(ctx context.Context, dir, key string, lease *HostCreationLease) (string, bool, error) {
	out, err := p.git(ctx, dir, lease, "config", "--get-all", key)
	if err != nil {
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() == 1 && out == "" {
			return "", false, nil
		}
		return "", false, err
	}
	if out == "" || strings.Contains(out, "\n") {
		return "", false, fmt.Errorf("empty or duplicate git configuration %s", key)
	}
	return out, true, nil
}
func (p *WorkspaceProvisioner) selectRemote(ctx context.Context, dir, requested string, lease *HostCreationLease) (string, error) {
	out, err := p.git(ctx, dir, lease, "remote")
	if err != nil {
		return "", err
	}
	var remotes []string
	if out != "" {
		remotes = strings.Split(out, "\n")
	}
	selected := requested
	if selected == "" {
		remote, hasRemote, err := p.configValue(ctx, dir, "branch.main.remote", lease)
		if err != nil {
			return "", err
		}
		merge, hasMerge, err := p.configValue(ctx, dir, "branch.main.merge", lease)
		if err != nil {
			return "", err
		}
		if hasRemote && hasMerge && merge == "refs/heads/main" {
			selected = remote
		} else if len(remotes) == 1 {
			selected = remotes[0]
		}
	}
	if selected == "" || selected == "." {
		return "", errors.New("no unique remote/main source; select a configured --remote")
	}
	if _, err := ParseProvisionRequest(dir, "1", selected); err != nil {
		return "", err
	}
	for _, name := range remotes {
		if name == selected {
			return name, nil
		}
	}
	return "", fmt.Errorf("configured remote %q does not exist", selected)
}
