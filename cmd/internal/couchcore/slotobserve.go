package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// AgentRunning reads the recovery report's agent evidence for reconcile
// (pair#387): running is whether an agent process may be working in the slot,
// known whether that is proven. A detached session is a running agent; a
// parked or absent one is not; an unusable row proves nothing either way.
// Reconcile never removes a checkout unless running is false and known.
func AgentRunning(a EvidenceAgent) (running, known bool) {
	switch a {
	case AgentNone, AgentParked:
		return false, true
	case AgentLive, AgentDetached, AgentBusy, AgentOrphaned:
		// An orphan's server and agent are alive, only unreachable (#399).
		return true, true
	}
	return false, false
}

// StatePending is a resource not observed because a resource it depends on is
// not present yet: it is neither a step nor a stop, and the next pass observes
// it once its dependency converges.
const StatePending ObservedState = 0

// Sub-states refine an observation (SlotResourceSpec.SubStates names the
// setup ones the plan reads).
const (
	SubStale       = "stale"        // registration: its directory is gone
	SubLocked      = "locked"       // registration: locked by its owner
	SubMismatched  = "mismatched"   // host: a checkout that is not this slot's
	SubUnreadable  = "unreadable"   // a checkout git positively cannot read
	SubForeign     = "foreign"      // a non-directory or symlink at a slot path
	SubElsewhere   = "elsewhere"    // branch: checked out in another worktree
	SubConflict    = "conflict"     // upstream: configured differently
	SubNotLayer    = "not-a-layer"  // dep: git-readable but not a layer
	SubOutside     = "outside"      // dep: not in the environment; not slot state
	SubLockHeld    = "lock-held"    // setup: weave is running
	SubMarkerValid = "marker-valid" // setup: absent only because a dependency is
	SubWithWarning = "present-with-warning"
	SubFailedKnown = "failed-known"
)

// ResourceObservation is one resource's observed state.
type ResourceObservation struct {
	ID     SlotResourceID
	State  ObservedState
	Sub    string
	Reason string
	// Branch: for registration and host, the branch the registration records.
	Branch string
	// Dep: for a dep:* instance, its declaration.
	Dep *DeclaredDep
}

// SlotObservation is one observation of every resource of a slot, in
// converge order (dependency instances follow deps).
type SlotObservation struct {
	Primary   string
	Number    int
	Resources []ResourceObservation
	Agent     EvidenceAgent
}

// Get returns the observation of id.
func (o SlotObservation) Get(id SlotResourceID) (ResourceObservation, bool) {
	for _, r := range o.Resources {
		if r.ID == id {
			return r, true
		}
	}
	return ResourceObservation{}, false
}

// Equal compares two observations (the loop's no-progress guard).
func (o SlotObservation) Equal(other SlotObservation) bool {
	if o.Primary != other.Primary || o.Number != other.Number || o.Agent != other.Agent || len(o.Resources) != len(other.Resources) {
		return false
	}
	for i, r := range o.Resources {
		s := other.Resources[i]
		if r.ID != s.ID || r.State != s.State || r.Sub != s.Sub || r.Reason != s.Reason || r.Branch != s.Branch || (r.Dep == nil) != (s.Dep == nil) || (r.Dep != nil && *r.Dep != *s.Dep) {
			return false
		}
	}
	return true
}

// SlotObserveInput is what ObserveSlot needs.
type SlotObserveInput struct {
	IO     ProvisionIO
	Layout SlotLayout
	// Remote is the remote the resting branch should track ("" accepts any
	// configured remote whose merge is refs/heads/main).
	Remote string
	Agent  EvidenceAgent
	// IgnoreMemo reads setup without the remembered failure (R5): an
	// explicit couch --reconcile compiles again.
	IgnoreMemo bool
}

// worktreeEntry is one record of `git worktree list --porcelain -z`.
type worktreeEntry struct {
	Path, Head, Branch string
	Bare, Detached     bool
	Prunable, Locked   bool
}

// parseWorktreeList reads `git worktree list --porcelain -z` (records are
// separated by an empty field).
func parseWorktreeList(raw string) []worktreeEntry {
	var out []worktreeEntry
	var cur *worktreeEntry
	for _, field := range strings.Split(raw, "\x00") {
		if field == "" {
			if cur != nil {
				out = append(out, *cur)
				cur = nil
			}
			continue
		}
		key, value, _ := strings.Cut(field, " ")
		if key == "worktree" {
			if cur != nil {
				out = append(out, *cur)
			}
			cur = &worktreeEntry{Path: value}
			continue
		}
		if cur == nil {
			continue
		}
		switch key {
		case "HEAD":
			cur.Head = value
		case "branch":
			cur.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			cur.Bare = true
		case "detached":
			cur.Detached = true
		case "prunable":
			cur.Prunable = true
		case "locked":
			cur.Locked = true
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

func notAGitRepository(err error, out string) bool {
	return err != nil && strings.Contains(strings.ToLower(out+" "+err.Error()), "not a git repository")
}

// observer carries one ObserveSlot pass.
type observer struct {
	ctx context.Context
	in  SlotObserveInput
	obs SlotObservation
}

func (o *observer) run(dir string, args ...string) (string, error) {
	out, err := o.in.IO.Run(o.ctx, ProvisionCommand{Dir: dir, Program: "git", Args: args})
	return strings.TrimSuffix(string(out), "\n"), err
}

func (o *observer) add(r ResourceObservation) { o.obs.Resources = append(o.obs.Resources, r) }

func (o *observer) state(id SlotResourceID) ObservedState {
	r, _ := o.obs.Get(id)
	return r.State
}

// lstatState maps a filesystem probe to an observation of a directory.
func lstatState(path string) (ObservedState, string, string) {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return StateAbsent, "", ""
	case err != nil:
		return StateUnknown, "", err.Error()
	case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
		return StateBroken, SubForeign, path + " is not a directory"
	}
	return StatePresent, "", ""
}

// ObserveSlot observes every resource of a :1+ slot (pair#387). It changes
// nothing. A failed probe is unknown, never absent. The layout must be built
// from git's own resolved identity (Discover's SlotIdentity, or a physical
// primary with git's common directory): git reports resolved paths, so a
// layout from an unresolved symlink or the <primary>/.git guess misreads a
// healthy slot.
func ObserveSlot(ctx context.Context, in SlotObserveInput) SlotObservation {
	if ctx == nil {
		ctx = context.Background()
	}
	l := in.Layout
	o := &observer{ctx: ctx, in: in, obs: SlotObservation{Primary: l.primary, Number: l.n, Agent: in.Agent}}
	if in.Agent == "" {
		o.obs.Agent = AgentUnusableUnknown
	}
	// env, store
	state, sub, reason := lstatState(l.Env())
	o.add(ResourceObservation{ID: ResourceEnv, State: state, Sub: sub, Reason: reason})
	if o.state(ResourceEnv) == StatePresent {
		state, sub, reason = lstatState(l.Store())
		o.add(ResourceObservation{ID: ResourceStore, State: state, Sub: sub, Reason: reason})
	} else {
		o.add(ResourceObservation{ID: ResourceStore, State: StatePending, Reason: "waits for env"})
	}
	o.observeIntent()
	worktrees, listErr := o.run(l.primary, "worktree", "list", "--porcelain", "-z")
	entries := parseWorktreeList(worktrees)
	o.observeBranch(entries, listErr)
	o.observeUpstream()
	o.observeRegistration(entries, listErr)
	admin := o.observeHost()
	deps := o.observeDeps()
	o.observeSetup(admin, deps)
	o.add(ResourceObservation{ID: ResourceAgent, State: agentResourceState(o.obs.Agent), Reason: string(o.obs.Agent)})
	return o.obs
}

func agentResourceState(a EvidenceAgent) ObservedState {
	switch running, known := AgentRunning(a); {
	case !known:
		return StateUnknown
	case running:
		return StatePresent
	}
	return StateAbsent
}

func (o *observer) observeIntent() {
	path := o.in.Layout.Intent()
	if path == "" {
		o.add(ResourceObservation{ID: ResourceIntent, State: StateUnknown, Reason: "git common directory unknown"})
		return
	}
	_, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		o.add(ResourceObservation{ID: ResourceIntent, State: StateAbsent})
	case err != nil:
		o.add(ResourceObservation{ID: ResourceIntent, State: StateUnknown, Reason: err.Error()})
	default:
		o.add(ResourceObservation{ID: ResourceIntent, State: StatePresent})
	}
}

func (o *observer) observeBranch(entries []worktreeEntry, listErr error) {
	l := o.in.Layout
	out, err := o.run(l.primary, "for-each-ref", "--format=%(refname)", l.RestingRef())
	switch {
	case err != nil:
		o.add(ResourceObservation{ID: ResourceBranch, State: StateUnknown, Reason: "read " + l.RestingRef() + ": " + err.Error()})
		return
	case out == "":
		o.add(ResourceObservation{ID: ResourceBranch, State: StateAbsent})
		return
	case listErr != nil:
		o.add(ResourceObservation{ID: ResourceBranch, State: StateUnknown, Reason: "git worktree list: " + listErr.Error()})
		return
	}
	for _, e := range entries {
		if e.Branch == l.RestingBranch() && filepath.Clean(e.Path) != l.Host() {
			o.add(ResourceObservation{ID: ResourceBranch, State: StateBroken, Sub: SubElsewhere, Reason: l.RestingBranch() + " is checked out in " + e.Path})
			return
		}
	}
	o.add(ResourceObservation{ID: ResourceBranch, State: StatePresent})
}

// configValues reads a multi-valued git config key: none is not an error.
func (o *observer) configValues(key string) ([]string, error) {
	out, err := o.run(o.in.Layout.primary, "config", "--get-all", key)
	if err != nil {
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() == 1 && out == "" {
			return nil, nil
		}
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func (o *observer) observeUpstream() {
	if s := o.state(ResourceBranch); s != StatePresent {
		o.add(ResourceObservation{ID: ResourceUpstream, State: StatePending, Reason: "waits for branch"})
		return
	}
	rest := o.in.Layout.RestingBranch()
	remotes, err := o.configValues("branch." + rest + ".remote")
	if err == nil {
		var merges []string
		merges, err = o.configValues("branch." + rest + ".merge")
		if err == nil {
			o.add(upstreamObservation(remotes, merges, o.in.Remote))
			return
		}
	}
	o.add(ResourceObservation{ID: ResourceUpstream, State: StateUnknown, Reason: err.Error()})
}

// upstreamObservation judges the resting branch's upstream configuration.
func upstreamObservation(remotes, merges []string, want string) ResourceObservation {
	r := ResourceObservation{ID: ResourceUpstream}
	switch {
	case len(remotes) > 1 || len(merges) > 1:
		r.State, r.Sub, r.Reason = StateBroken, SubConflict, "duplicate upstream configuration"
	case len(merges) == 1 && merges[0] != "refs/heads/main":
		r.State, r.Sub, r.Reason = StateBroken, SubConflict, "merges "+merges[0]+", not refs/heads/main"
	case len(remotes) == 1 && want != "" && remotes[0] != want:
		r.State, r.Sub, r.Reason = StateBroken, SubConflict, "tracks remote "+remotes[0]+", not "+want
	case len(remotes) == 1 && len(merges) == 1:
		r.State = StatePresent
	default:
		r.State = StateAbsent
	}
	return r
}

func (o *observer) observeRegistration(entries []worktreeEntry, listErr error) {
	l := o.in.Layout
	if listErr != nil {
		o.add(ResourceObservation{ID: ResourceRegistration, State: StateUnknown, Reason: "git worktree list: " + listErr.Error()})
		return
	}
	for _, e := range entries {
		if filepath.Clean(e.Path) != l.Host() {
			continue
		}
		r := ResourceObservation{ID: ResourceRegistration, State: StatePresent, Branch: e.Branch}
		switch {
		case e.Locked:
			r.State, r.Sub, r.Reason = StateBroken, SubLocked, "the registration is locked"
		case e.Prunable:
			r.State, r.Sub, r.Reason = StateBroken, SubStale, "its directory is gone"
		}
		o.add(r)
		return
	}
	o.add(ResourceObservation{ID: ResourceRegistration, State: StateAbsent})
}

// observeHost observes the host checkout and returns its administrative
// directory when it is this slot's verified checkout.
func (o *observer) observeHost() string {
	l := o.in.Layout
	if o.state(ResourceEnv) != StatePresent {
		o.add(ResourceObservation{ID: ResourceHost, State: StatePending, Reason: "waits for env"})
		return ""
	}
	reg, _ := o.obs.Get(ResourceRegistration)
	r := ResourceObservation{ID: ResourceHost, Branch: reg.Branch}
	state, sub, reason := lstatState(l.Host())
	if state != StatePresent {
		r.State, r.Sub, r.Reason = state, sub, reason
		o.add(r)
		return ""
	}
	admin, err := o.run(l.Host(), "rev-parse", "--absolute-git-dir")
	if err != nil {
		r.State, r.Reason = StateUnknown, err.Error()
		if notAGitRepository(err, admin) {
			r.State, r.Sub, r.Reason = StateBroken, SubUnreadable, "git cannot read "+l.Host()
		}
		o.add(r)
		return ""
	}
	common, err := o.run(l.Host(), "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		r.State, r.Reason = StateUnknown, err.Error()
		o.add(r)
		return ""
	}
	if filepath.Clean(common) != filepath.Clean(l.common) || filepath.Dir(admin) != l.Registrations() {
		r.State, r.Sub, r.Reason = StateBroken, SubMismatched, l.Host()+" is not this slot's checkout of "+l.primary
		o.add(r)
		return ""
	}
	r.State = StatePresent
	o.add(r)
	return admin
}

// observeDeps observes the dependency declaration and each declared clone.
func (o *observer) observeDeps() []ResourceObservation {
	l := o.in.Layout
	if o.state(ResourceHost) != StatePresent {
		o.add(ResourceObservation{ID: ResourceDeps, State: StatePending, Reason: "waits for host"})
		return nil
	}
	declared, err := DeclaredDepsOf(l.Env(), l.Host())
	if err != nil {
		o.add(ResourceObservation{ID: ResourceDeps, State: StateUnknown, Reason: err.Error()})
		return nil
	}
	o.add(ResourceObservation{ID: ResourceDeps, State: StatePresent})
	var deps []ResourceObservation
	list := declared.Deps
	if declared.NotLayer != nil {
		list = []DeclaredDep{*declared.NotLayer}
	}
	for _, d := range list {
		d := d
		r := o.observeDep(d, declared.NotLayer != nil)
		o.add(r)
		deps = append(deps, r)
	}
	return deps
}

func (o *observer) observeDep(d DeclaredDep, notLayer bool) ResourceObservation {
	id := DepResource(d.Rel)
	if d.Outside {
		id = DepResource(d.Path)
	}
	r := ResourceObservation{ID: id, Dep: &d}
	if d.Outside {
		r.State, r.Sub = StateAbsent, SubOutside
		if d.Present {
			r.State = StatePresent
		}
		return r
	}
	state, sub, reason := lstatState(d.Path)
	if state != StatePresent {
		r.State, r.Sub, r.Reason = state, sub, reason
		return r
	}
	// Git's own answer about the work tree, not where its git directory lives:
	// a gitfile-backed clone is a readable checkout of its own (R2).
	r.State, r.Sub, r.Reason = checkoutEvidence(o.ctx, o.in.IO, d.Path)
	switch {
	case r.State != StatePresent:
	case notLayer:
		r.State, r.Sub, r.Reason = StateBroken, SubNotLayer, d.Path+" is a repository but not a layer (no construct/base.manifest)"
	}
	return r
}

// setupLockHeld probes weave's setup lock without creating it. A missing lock
// file means no setup has ever run, so none can be running.
func setupLockHeld(path string) (bool, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer unix.Close(fd)
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return true, nil
		}
		return false, err
	}
	return false, unix.Flock(fd, unix.LOCK_UN)
}

func (o *observer) observeSetup(admin string, deps []ResourceObservation) {
	l := o.in.Layout
	if o.state(ResourceHost) != StatePresent || admin == "" {
		o.add(ResourceObservation{ID: ResourceSetup, State: StatePending, Reason: "waits for host"})
		return
	}
	r := ResourceObservation{ID: ResourceSetup}
	var marker SetupSuccess
	exists, err := (ProvisionStore{}).Read(SetupMarkerPath(admin), &marker)
	switch {
	case err != nil:
		r.State, r.Reason = StateUnknown, "read setup marker: "+err.Error()
	case !exists:
		r.State = StateAbsent
	case !ValidSetupMarker(marker, l.Host(), l.common, admin, l.n):
		r.State, r.Reason = StateBroken, "the setup marker does not match this slot"
	default:
		r.State = StatePresent
		for _, d := range deps {
			if d.Sub == SubOutside || (d.State != StateAbsent && d.State != StateBroken) {
				continue
			}
			// The marker is valid, but the setup it records is incomplete.
			r.State, r.Sub, r.Reason = StateAbsent, SubMarkerValid, string(d.ID)+" is "+d.State.String()
			break
		}
	}
	// A remembered hand-off failure for unchanged inputs is not recompiled
	// (R5): under a valid marker the slot stays usable with the warning;
	// without one the failure is known.
	if (r.State == StateAbsent || r.State == StateBroken) && !o.in.IgnoreMemo {
		if failure, known := knownFailedSetup(o.ctx, o.in.IO, l, admin); known {
			if r.Sub == SubMarkerValid {
				r.State, r.Sub, r.Reason = StatePresent, SubWithWarning, failure
			} else {
				r.State, r.Sub, r.Reason = StateAbsent, SubFailedKnown, failure
			}
		}
	}
	// A running setup outranks every marker reading: nothing may compile now.
	held, err := setupLockHeld(l.SetupLock())
	switch {
	case err != nil:
		r.State, r.Sub, r.Reason = StateUnknown, "", "probe setup lock: "+err.Error()
	case held:
		r.Sub, r.Reason = SubLockHeld, "weave setup is running in "+l.Env()
	}
	o.add(r)
}

// ValidSetupMarker is the setup-success marker check readSuccess applies: the
// marker names exactly this slot's checkout, registration and baseline.
func ValidSetupMarker(s SetupSuccess, host, common, admin string, slot int) bool {
	return s.SchemaVersion == 1 && s.Host == host && s.Common == common && s.Admin == admin && s.Slot == slot && validWorkspaceOID(s.BaselineSHA)
}
