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
	"strconv"
	"strings"
	"time"
)

type WorkspaceProvisioner struct {
	IO                                       ProvisionIO
	Store                                    ProvisionStorage
	ProbeTimeout, FetchTimeout, SetupTimeout time.Duration
}

func NewWorkspaceProvisioner(commandIO ProvisionIO) *WorkspaceProvisioner {
	return &WorkspaceProvisioner{IO: commandIO, Store: ProvisionStore{}, ProbeTimeout: 5 * time.Second, FetchTimeout: 120 * time.Second, SetupTimeout: 20 * time.Minute}
}

type provisionHost struct {
	identity                    WorkspaceIdentity
	admin, baseline, intentPath string
	created, ready              bool
}

func (p *WorkspaceProvisioner) command(ctx context.Context, dir, program string, args []string, lease *HostCreationLease, progress io.Writer, timeout time.Duration) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := ProvisionCommand{Dir: dir, Program: program, Args: args, Timeout: timeout, Progress: progress, StreamOutput: program == "weave"}
	if lease != nil && program == "git" {
		c.Lease = lease.File()
	}
	return p.IO.Run(ctx, c)
}
func (p *WorkspaceProvisioner) git(ctx context.Context, dir string, lease *HostCreationLease, args ...string) (string, error) {
	raw, err := p.command(ctx, dir, "git", args, lease, nil, p.ProbeTimeout)
	return strings.TrimSuffix(string(raw), "\n"), err
}
func (p *WorkspaceProvisioner) identity(ctx context.Context, path string) (WorkspaceIdentity, error) {
	raw, err := p.command(ctx, path, "sdlc", []string{"workspace", "--json"}, nil, nil, p.ProbeTimeout)
	if err != nil {
		return WorkspaceIdentity{}, fmt.Errorf("resolve workspace: %w", err)
	}
	return ParseWorkspaceIdentity(raw)
}

// Ensure is the sole provisioning operation. Repeating it finishes provable
// missing steps; no retry mode, thread reservation, or agent launch is involved.
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
	lease, err := AcquireHostCreationLease(primary.RepoIdentity)
	if err != nil {
		return ProvisionResult{}, err
	}
	host, err := p.ensureHost(ctx, primary, req, lease)
	closeErr := lease.Close()
	if err != nil {
		return ProvisionResult{}, err
	}
	if closeErr != nil {
		return ProvisionResult{}, closeErr
	}
	if host.ready {
		return provisionResult(host, "reused"), nil
	}
	if req.Progress != nil {
		fmt.Fprintf(req.Progress, "Preparing %s in %s\n", *host.identity.Address, host.identity.WorktreeRoot)
	}
	if _, err := p.command(ctx, host.identity.WorktreeRoot, "weave", []string{"compile"}, nil, req.Progress, p.SetupTimeout); err != nil {
		return ProvisionResult{}, fmt.Errorf("workspace setup unconfirmed; open again to retry: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return ProvisionResult{}, err
	}
	lease, err = AcquireHostCreationLease(primary.RepoIdentity)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("setup completed but success is unconfirmed: %w", err)
	}
	defer lease.Close()
	current, err := p.verifyHost(ctx, primary, req.Slot, lease)
	if err != nil {
		return ProvisionResult{}, err
	}
	if current.admin != host.admin {
		return ProvisionResult{}, errors.New("workspace administrative directory changed during setup")
	}
	success, exists, err := p.readSuccess(current)
	if err != nil {
		return ProvisionResult{}, err
	}
	if exists {
		host.baseline = success.BaselineSHA
	} else {
		if err := ctx.Err(); err != nil {
			return ProvisionResult{}, err
		}
		success = SetupSuccess{SchemaVersion: 1, Host: host.identity.WorktreeRoot, Common: primary.RepoIdentity, Admin: host.admin, Slot: req.Slot, BaselineSHA: host.baseline}
		if err := p.Store.Write(filepath.Join(host.admin, "couch-setup-success.json"), success); err != nil {
			return ProvisionResult{}, fmt.Errorf("record setup success: %w", err)
		}
	}
	p.cleanupIntent(host, req.Progress)
	disposition := "prepared"
	if host.created {
		disposition = "created"
	}
	return provisionResult(host, disposition), nil
}
func provisionResult(h provisionHost, disposition string) ProvisionResult {
	return ProvisionResult{SchemaVersion: 1, Address: *h.identity.Address, Path: h.identity.WorktreeRoot, RestingBranch: *h.identity.RestingBranch, BaselineSHA: h.baseline, Disposition: disposition}
}
func slotHostPath(primary WorkspaceIdentity, n int) string {
	return filepath.Join(primary.FleetRoot, "worktree", fmt.Sprintf("%s-slot%d", primary.Repo, n), primary.Repo)
}
func (p *WorkspaceProvisioner) ensureHost(ctx context.Context, primary WorkspaceIdentity, req ProvisionRequest, lease *HostCreationLease) (provisionHost, error) {
	current, err := p.identity(ctx, primary.PrimaryRoot)
	if err != nil {
		return provisionHost{}, err
	}
	if current.Kind != "primary" || current.RepoIdentity != primary.RepoIdentity || current.PrimaryRoot != primary.PrimaryRoot || current.FleetRoot != primary.FleetRoot {
		return provisionHost{}, errors.New("primary workspace identity changed")
	}
	hostPath := slotHostPath(primary, req.Slot)
	env := filepath.Dir(hostPath)
	rest := fmt.Sprintf("main-slot%d", req.Slot)
	if err := provisionSafePath(hostPath); err != nil {
		return provisionHost{}, err
	}
	intentPath := filepath.Join(primary.RepoIdentity, "couch-workspaces", strconv.Itoa(req.Slot), "creation.json")
	var intent CreationIntent
	owned, err := p.Store.Read(intentPath, &intent)
	if err != nil {
		return provisionHost{}, err
	}
	if owned {
		if err := validateCreationIntent(intent, primary, req, hostPath); err != nil {
			return provisionHost{}, err
		}
	}
	info, pathErr := os.Lstat(hostPath)
	if pathErr == nil {
		if !info.IsDir() {
			return provisionHost{}, fmt.Errorf("foreign workspace path %s", hostPath)
		}
		host, err := p.verifyHost(ctx, primary, req.Slot, lease)
		if err != nil {
			return provisionHost{}, err
		}
		host.intentPath = intentPath
		success, exists, err := p.readSuccess(host)
		if err != nil {
			return provisionHost{}, err
		}
		observation := HostObservation{Kind: HostVerified}
		if exists {
			observation.Setup = SetupConfirmed
		}
		switch NextHostAction(observation) {
		case ReuseHost:
			host.baseline = success.BaselineSHA
			host.ready = true
			p.cleanupIntent(host, req.Progress)
			return host, nil
		case CompileHost:
			if owned {
				host.baseline = intent.BaselineSHA
			} else {
				host.baseline, err = p.git(ctx, hostPath, lease, "rev-parse", "--verify", "refs/heads/"+rest+"^{commit}")
				if err != nil {
					return provisionHost{}, err
				}
				if !validWorkspaceOID(host.baseline) {
					return provisionHost{}, errors.New("invalid resting branch baseline")
				}
			}
			return host, nil
		default:
			return provisionHost{}, errors.New("workspace setup evidence conflicts")
		}
	}
	if !errors.Is(pathErr, os.ErrNotExist) {
		return provisionHost{}, pathErr
	}
	observation := HostObservation{Kind: HostAbsent}
	if owned {
		observation.Kind = HostOwnedPartial
	}
	action := NextHostAction(observation)
	if action != CreateHost && action != CompleteHost {
		return provisionHost{}, errors.New("cannot create workspace from conflicting evidence")
	}
	if !owned {
		if _, err := os.Lstat(env); err == nil {
			return provisionHost{}, fmt.Errorf("workspace environment already exists without ownership evidence: %s", env)
		} else if !errors.Is(err, os.ErrNotExist) {
			return provisionHost{}, err
		}
		ref, err := p.branchOID(ctx, primary.PrimaryRoot, rest, lease)
		if err != nil {
			return provisionHost{}, err
		}
		if ref != "" {
			return provisionHost{}, fmt.Errorf("resting branch %s already exists without ownership evidence", rest)
		}
		remote, err := p.selectRemote(ctx, primary.PrimaryRoot, req.Remote, lease)
		if err != nil {
			return provisionHost{}, err
		}
		tracking := "refs/remotes/" + remote + "/main"
		raw, err := p.command(ctx, primary.PrimaryRoot, "git", []string{"fetch", "--verbose", "--porcelain", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--refmap=", remote, "+refs/heads/main:" + tracking}, lease, req.Progress, p.FetchTimeout)
		if err != nil {
			return provisionHost{}, fmt.Errorf("fetch remote main: %w", err)
		}
		baseline, err := ParseFetchBaseline(raw, tracking)
		if err != nil {
			return provisionHost{}, err
		}
		token := make([]byte, 16)
		if _, err := rand.Read(token); err != nil {
			return provisionHost{}, err
		}
		intent = CreationIntent{SchemaVersion: 1, Primary: primary.PrimaryRoot, Common: primary.RepoIdentity, Host: hostPath, Slot: req.Slot, Remote: remote, BaselineSHA: baseline, Token: hex.EncodeToString(token)}
		if err := p.Store.Write(intentPath, intent); err != nil {
			return provisionHost{}, err
		}
	}
	if intent.DirInode == 0 {
		if err := provisionMkdirAll(filepath.Dir(env)); err != nil {
			return provisionHost{}, err
		}
		if err := os.Mkdir(env, 0700); err != nil {
			return provisionHost{}, fmt.Errorf("cannot prove environment ownership; inspect %s: %w", env, err)
		}
		intent.DirDevice, intent.DirInode, err = provisionDirIdentity(env)
		if err != nil {
			return provisionHost{}, err
		}
		if err := p.Store.Write(intentPath, intent); err != nil {
			return provisionHost{}, err
		}
	} else {
		dev, ino, err := provisionDirIdentity(env)
		if err != nil {
			return provisionHost{}, err
		}
		if dev != intent.DirDevice || ino != intent.DirInode {
			return provisionHost{}, errors.New("workspace environment was replaced; inspect before proceeding")
		}
	}
	oid, err := p.branchOID(ctx, primary.PrimaryRoot, rest, lease)
	if err != nil {
		return provisionHost{}, err
	}
	message := "couch-slot-create:" + intent.Token
	if oid == "" {
		if _, err := p.git(ctx, primary.PrimaryRoot, lease, "update-ref", "--create-reflog", "-m", message, "refs/heads/"+rest, intent.BaselineSHA, strings.Repeat("0", len(intent.BaselineSHA))); err != nil {
			return provisionHost{}, err
		}
	} else {
		evidence, err := p.git(ctx, primary.PrimaryRoot, lease, "reflog", "show", "-1", "--format=%H%x00%gs", "refs/heads/"+rest)
		if err != nil {
			return provisionHost{}, err
		}
		if oid != intent.BaselineSHA || evidence != intent.BaselineSHA+"\x00"+message {
			return provisionHost{}, fmt.Errorf("cannot prove ownership of resting branch %s", rest)
		}
	}
	for _, entry := range []struct{ key, value string }{{"branch." + rest + ".remote", intent.Remote}, {"branch." + rest + ".merge", "refs/heads/main"}} {
		value, found, err := p.configValue(ctx, primary.PrimaryRoot, entry.key, lease)
		if err != nil {
			return provisionHost{}, err
		}
		if found && value != entry.value {
			return provisionHost{}, fmt.Errorf("conflicting upstream configuration for %s", rest)
		}
		if !found {
			if _, err := p.git(ctx, primary.PrimaryRoot, lease, "config", "--add", entry.key, entry.value); err != nil {
				return provisionHost{}, err
			}
		}
		value, found, err = p.configValue(ctx, primary.PrimaryRoot, entry.key, lease)
		if err != nil {
			return provisionHost{}, err
		}
		if !found || value != entry.value {
			return provisionHost{}, errors.New("upstream configuration changed")
		}
	}
	if _, err := p.git(ctx, primary.PrimaryRoot, lease, "worktree", "add", hostPath, rest); err != nil {
		return provisionHost{}, fmt.Errorf("create host worktree; retained owned progress for next invocation: %w", err)
	}
	host, err := p.verifyHost(ctx, primary, req.Slot, lease)
	if err != nil {
		return provisionHost{}, err
	}
	host.baseline = intent.BaselineSHA
	host.created = true
	host.intentPath = intentPath
	return host, nil
}
func validateCreationIntent(i CreationIntent, primary WorkspaceIdentity, req ProvisionRequest, host string) error {
	token, err := hex.DecodeString(i.Token)
	if i.SchemaVersion != 1 || i.Primary != primary.PrimaryRoot || i.Common != primary.RepoIdentity || i.Host != host || i.Slot != req.Slot || !validWorkspaceOID(i.BaselineSHA) || err != nil || len(token) != 16 || hex.EncodeToString(token) != i.Token {
		return errors.New("invalid workspace creation intent")
	}
	if _, err := ParseProvisionRequest(req.Path, strconv.Itoa(req.Slot), i.Remote); err != nil || i.Remote == "" {
		return errors.New("invalid intent remote")
	}
	if req.Remote != "" && req.Remote != i.Remote {
		return errors.New("requested remote conflicts with captured creation intent")
	}
	return nil
}
func (p *WorkspaceProvisioner) verifyHost(ctx context.Context, primary WorkspaceIdentity, slot int, lease *HostCreationLease) (provisionHost, error) {
	path := slotHostPath(primary, slot)
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
	if filepath.Dir(admin) != filepath.Join(primary.RepoIdentity, "worktrees") {
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
func (p *WorkspaceProvisioner) readSuccess(host provisionHost) (SetupSuccess, bool, error) {
	var s SetupSuccess
	exists, err := p.Store.Read(filepath.Join(host.admin, "couch-setup-success.json"), &s)
	if err != nil || !exists {
		return s, exists, err
	}
	if s.SchemaVersion != 1 || s.Host != host.identity.WorktreeRoot || s.Common != host.identity.RepoIdentity || s.Admin != host.admin || s.Slot != *host.identity.Slot || !validWorkspaceOID(s.BaselineSHA) {
		return s, false, errors.New("invalid or mismatched workspace setup-success marker")
	}
	return s, true, nil
}
func (p *WorkspaceProvisioner) cleanupIntent(host provisionHost, progress io.Writer) {
	if host.intentPath == "" {
		return
	}
	if err := p.Store.Remove(host.intentPath); err != nil && progress != nil {
		fmt.Fprintf(progress, "Workspace ready; creation-intent cleanup deferred: %v\n", err)
	}
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
