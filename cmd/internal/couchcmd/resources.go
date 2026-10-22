package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type resourceReader interface {
	Table(context.Context) ([]procutil.Process, int, error)
	Usage(context.Context, procutil.Process) (procutil.ResourceUsage, int, error)
	Command(context.Context, procutil.Process) (procutil.CommandInfo, int, error)
	Cwd(context.Context, procutil.Process) (string, int, error)
	Environment(context.Context, procutil.Process, ...string) (map[string]string, int, error)
	SelfCPU(context.Context) (uint64, int, error)
}

// resourceSampler has one worker owner. Handlers read cached reports or request
// fresh ownership evidence from that worker; they never read mutable state.
// persist must honor ctx; its storage/alert work is included in the final cost.
type resourceSampler struct {
	history func() (time.Time, int)
	reader  resourceReader
	slots   func() couchcore.ResourceSlotInventory
	policy  couchcore.ResourcePolicy
	state   couchcore.ResourceState
	now     func() time.Time
	persist func(context.Context, couchcore.ResourceView) (couchcore.ResourceCost, error)
	alerts  func(context.Context, *couchcore.ResourceState) error
	cached  atomic.Pointer[couchcore.ResourceView]
	// sweeps carries a confirmed reap's request for a classification of its own
	// (#583). Only run reads it, so the plan a kill is decided from is built by
	// the one owner of state and provenance rather than by a copy of them.
	sweeps chan resourceRequest
	// threads caches each launchd child's inherited thread by process
	// generation (pair#606): an environment never changes after exec, so each
	// process costs one read in its lifetime. Only run touches it.
	threads map[couchcore.ResourceProcessKey]inheritedThread
	// leftRunning is the orphan registry teardown writes (pair#695); each
	// pass prunes it against its table and reports the rest.
	leftRunning *couchcore.LeftRunningRegistry
	// Constructed once for this Couch service lifetime; only run owns this
	// sticky flag. No later healthy scan can recover forgotten provenance.
	// Nothing persists it, and a fresh process is the only reset (#792).
	provenanceIncomplete bool
}

// resourceSweep is a reap's answer: the view one pass produced, and whether that
// pass ran to completion. A pass cut short leaves the cache holding an older
// view, and `fresh` is what stops that older view from being handed over as a
// classification to kill from.
type resourceRequest struct {
	ctx   context.Context
	proof bool
	reply chan resourceSweep
}

type resourceSweep struct {
	proof         couchcore.ProvenOrphanProof
	err           error
	view          couchcore.ResourceView
	fresh         bool
	legacyRefused bool
}

type inheritedThread struct {
	thread couchcore.ThreadAddress
	ok     bool
	known  bool
}

// inheritedThread answers DetachedSlotRoots from each process's environment,
// reading a generation once and forgetting generations that left the table.
func (s *resourceSampler) inheritedThread(ctx context.Context, r resourceReader, table []procutil.Process, sample *couchcore.ResourceSample, unknown map[couchcore.ResourceProcessKey]bool) func(procutil.Process) (couchcore.ThreadAddress, bool) {
	live := make(map[couchcore.ResourceProcessKey]bool, len(table))
	for _, p := range table {
		live[couchcore.ResourceProcessKey{PID: p.PID, Identity: p.Identity}] = true
	}
	for k := range s.threads {
		if !live[k] {
			delete(s.threads, k)
		}
	}
	if s.threads == nil {
		s.threads = map[couchcore.ResourceProcessKey]inheritedThread{}
	}
	return func(p procutil.Process) (couchcore.ThreadAddress, bool) {
		k := couchcore.ResourceProcessKey{PID: p.PID, Identity: p.Identity}
		if c, ok := s.threads[k]; ok {
			if !c.known {
				unknown[k] = true
			}
			return c.thread, c.ok
		}
		env, calls, err := r.Environment(ctx, p, "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PATH")
		sample.Cost.EnvironmentCalls += calls
		sample.Cost.NativeCalls += calls
		var unstable *procutil.IdentityUnavailableError
		if ctx.Err() != nil || errors.As(err, &unstable) {
			// Cancellation and a generation changing mid-read are transient:
			// ask again next pass.
			unknown[k] = true
			return couchcore.ThreadAddress{}, false
		}
		// Keep withheld/unreadable evidence distinct from a known absence. PATH
		// is only an observability witness and is never retained. An empty
		// environment (Darwin platform binaries) cannot disprove ownership.
		c := inheritedThread{thread: couchcore.ThreadAddress{RepoScope: env["COUCH_THREAD_SCOPE"], Tag: couchcore.ThreadTag(env["COUCH_THREAD_TAG"])}}
		c.known = err == nil && len(env) > 0 && ((c.thread.RepoScope == "") == (c.thread.Tag == ""))
		c.ok = c.known && c.thread.RepoScope != "" && c.thread.Tag != ""
		if !c.known {
			unknown[k] = true
		}
		s.threads[k] = c
		return c.thread, c.ok
	}
}

// run samples immediately, then serially on cadence. The service owns joining
// this goroutine; no child workers outlive run and ticks never overlap a pass.
// A reap's sweep is served in the same loop, so it is serial with the cadence
// too: two passes never read the host at once, and a sweep never waits behind
// more than the pass already in flight.
func (s *resourceSampler) run(ctx context.Context) {
	cadence := s.policy.Cadence
	if cadence <= 0 {
		cadence = time.Minute
	}
	ticker := time.NewTicker(cadence)
	defer ticker.Stop()
	if ctx.Err() == nil {
		s.pass(ctx)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-s.sweeps:
			if request.ctx.Err() != nil {
				request.reply <- resourceSweep{err: request.ctx.Err()}
				continue
			}
			// A request cannot lend the sampler a longer lifetime than its
			// owner. Both cancellation boundaries revoke the same collection.
			collection, cancel := context.WithCancel(request.ctx)
			unlink := context.AfterFunc(ctx, cancel)
			if ctx.Err() != nil {
				cancel()
			}
			if request.proof {
				sample, selection, unknown := s.collectOwnership(collection)
				// Monitor root-death evidence can remain useful while a current
				// owner is unknown. A destructive proof must additionally establish
				// complete ownership now; historical sticky refusal is independent.
				proof := couchcore.ProvenOrphanProof{Table: sample.Table, Selection: selection, TableKnown: sample.TableKnown, InventoryComplete: couchcore.ResourceOwnershipComplete(sample, selection, unknown), UnknownDetached: unknown}
				request.reply <- resourceSweep{proof: proof, err: collection.Err()}
			} else {
				fresh := s.pass(collection)
				request.reply <- resourceSweep{fresh: fresh, view: s.view(), legacyRefused: s.provenanceIncomplete, err: collection.Err()}
			}
			unlink()
			cancel()
			continue
		case <-ticker.C:
		}
		if ctx.Err() != nil {
			return
		}
		s.pass(ctx)
	}
}

// sweep asks the worker for a classification of its own and returns the view it
// produced. Both admission and collection are bounded by ctx. A cut-short
// pass never answers from the cache it did not refresh (#583).
func (s *resourceSampler) sweep(ctx context.Context) (resourceSweep, error) {
	if s == nil || s.sweeps == nil {
		return resourceSweep{}, errors.New("resource sampler has no worker")
	}
	reply := make(chan resourceSweep, 1)
	select {
	case s.sweeps <- resourceRequest{ctx: ctx, reply: reply}:
	case <-ctx.Done():
		return resourceSweep{}, ctx.Err()
	}
	select {
	case swept := <-reply:
		if swept.err != nil {
			return resourceSweep{}, swept.err
		}
		if !swept.fresh {
			return resourceSweep{}, errors.New("the sampler stopped before it could classify")
		}
		return swept, nil
	case <-ctx.Done():
		return resourceSweep{}, ctx.Err()
	}
}

// collectOwnership is shared by metrics passes and destructive proof requests.
// Only the sampler worker calls it: provenance and environment-cache ownership
// stay serial. Withheld/empty environments are unknown, not a negative match.
func (s *resourceSampler) collectOwnership(ctx context.Context) (couchcore.ResourceSample, couchcore.ResourceSelection, []couchcore.ResourceProcessKey) {
	sample := couchcore.ResourceSample{}
	unknown := map[couchcore.ResourceProcessKey]bool{}
	r := s.reader
	if r == nil {
		r = procutil.NativeResourceReader{}
	}
	if s.slots != nil {
		inventory := s.slots()
		sample.Slots = inventory.Slots
		sample.InventoryComplete = inventory.Complete && inventory.Omitted == 0
		sample.InventoryOmitted = inventory.Omitted
		if !sample.InventoryComplete {
			sample.Errors = append(sample.Errors, "slot inventory incomplete")
		}
		if inventory.Omitted > 0 {
			sample.Errors = append(sample.Errors, fmt.Sprintf("slot inventory omitted %d rows at capacity", inventory.Omitted))
		}
	}
	table, n, err := r.Table(ctx)
	sample.Cost.TableCalls += n
	sample.Cost.NativeCalls += n
	if ctx.Err() != nil {
		s.provenanceIncomplete = true
		return sample, couchcore.ResourceSelection{}, nil
	}
	sample.TableKnown = err == nil
	if err != nil {
		sample.Errors = append(sample.Errors, "process table: "+err.Error())
	}
	if len(table) > couchcore.ResourceMaxTableRows {
		table = table[:couchcore.ResourceMaxTableRows]
		sample.TableKnown = false
		sample.Errors = append(sample.Errors, "process table limit exceeded")
	}
	sample.Table = table
	sample.Slots = couchcore.DetachedSlotRoots(table, sample.Slots, s.inheritedThread(ctx, r, table, &sample, unknown))
	selection := couchcore.SelectResourceProcesses(table, sample.Slots, s.state.Provenance, sample.TableKnown, sample.InventoryComplete)
	unknownKeys := make([]couchcore.ResourceProcessKey, 0, len(unknown))
	for k := range unknown {
		unknownKeys = append(unknownKeys, k)
	}
	if !couchcore.ResourceOwnershipComplete(sample, selection, unknownKeys) {
		s.provenanceIncomplete = true
	}
	return sample, selection, unknownKeys
}

// proveReap asks the existing worker for fresh complete ownership evidence.
func (s *resourceSampler) proveReap(ctx context.Context) (couchcore.ProvenOrphanProof, error) {
	if s == nil || s.sweeps == nil {
		return couchcore.ProvenOrphanProof{}, errors.New("resource sampler has no worker")
	}
	reply := make(chan resourceSweep, 1)
	select {
	case s.sweeps <- resourceRequest{ctx: ctx, proof: true, reply: reply}:
	case <-ctx.Done():
		return couchcore.ProvenOrphanProof{}, ctx.Err()
	}
	select {
	case r := <-reply:
		return r.proof, r.err
	case <-ctx.Done():
		return couchcore.ProvenOrphanProof{}, ctx.Err()
	}
}

// pass reads the host once and publishes the view. It reports whether it
// classified anything: a ctx cut short returns without storing, and a caller
// that acts on the answer -- a reap -- must be able to tell that from a pass
// that saw an empty host (#583).
func (s *resourceSampler) pass(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	wall := time.Now()
	at := wall
	if s.now != nil {
		at = s.now()
	}
	r := s.reader
	if r == nil {
		r = procutil.NativeResourceReader{}
	}
	sample := couchcore.ResourceSample{At: at, Commands: map[couchcore.ResourceProcessKey][]string{}, OrphanProbes: map[couchcore.ResourceProcessKey]couchcore.ResourceOrphanProbe{}}
	start, n, cpuErr := r.SelfCPU(ctx)
	sample.Cost.NativeCalls += n
	if cpuErr != nil {
		sample.Errors = append(sample.Errors, "sampler CPU start: "+cpuErr.Error())
	}
	ownership, selection, _ := s.collectOwnership(ctx)
	if ctx.Err() != nil {
		return false
	}
	ownership.At, ownership.Commands, ownership.OrphanProbes = sample.At, sample.Commands, sample.OrphanProbes
	ownership.Cost.NativeCalls += sample.Cost.NativeCalls
	ownership.Errors = append(sample.Errors, ownership.Errors...)
	sample = ownership
	table := sample.Table
	rows := make(map[couchcore.ResourceProcessKey]procutil.Process, len(table))
	pidCounts := map[int]int{}
	for _, p := range table {
		rows[couchcore.ResourceProcessKey{PID: p.PID, Identity: p.Identity}] = p
		pidCounts[p.PID]++
	}
	selected := map[couchcore.ResourceProcessKey]bool{}
	observe := func(p procutil.Process) couchcore.ResourceUsageObservation {
		key := couchcore.ResourceProcessKey{PID: p.PID, Identity: p.Identity}
		u, calls, err := r.Usage(ctx, p)
		sample.Cost.UsageCalls += calls
		sample.Cost.NativeCalls += calls
		o := couchcore.ResourceUsageObservation{Key: key}
		if err != nil {
			o.Reason = "resource usage: " + err.Error()
		} else {
			o.CPUTimeNS = u.CPUTimeNS
			o.RSSBytes = u.RSSBytes
			o.DiskWriteBytes = u.DiskWriteBytes
			o.CPUKnown = true
			o.RSSKnown = true
			o.DiskKnown = true
		}
		sample.Observations = append(sample.Observations, o)
		return o
	}
	for _, owned := range selection.Processes {
		if ctx.Err() != nil {
			return false
		}
		selected[owned.Key] = true
		if p, ok := rows[owned.Key]; ok {
			observe(p)
		}
	}
	candidates := make([]procutil.Process, 0)
	for _, p := range table {
		// The row shape and the kind set are couchcore's rules, not the
		// sampler's: #583's orphan sweep classifies through the same predicate,
		// so a row this filter admits and the classifier refuses (or the
		// reverse) would be a disagreement between two copies of one rule.
		// UnattributedKind is the wide half — an nvim row is worth an argv read
		// before anything knows whether it is an embed (#583, ARCH-DRY).
		if pidCounts[p.PID] == 1 && couchcore.UnattributedShape(p, selected) && couchcore.UnattributedKind(filepath.Base(p.Name)) {
			candidates = append(candidates, p)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].PID < candidates[j].PID })
	remaining := couchcore.ResourceMaxProcesses - len(selected)
	if len(candidates) > remaining {
		candidates = candidates[:remaining]
		sample.Errors = append(sample.Errors, "unattributed candidate limit exceeded")
	}
	for _, p := range candidates {
		if ctx.Err() != nil {
			return false
		}
		o := observe(p)
		if filepath.Base(p.Name) != "nvim" {
			continue
		}
		// The age comes from the generation stamp the table already carries, so
		// the orphan gate costs nothing; only a row that clears it, or one big
		// enough for the unattributed report, pays for a probe (#583).
		start, ageKnown := procutil.StartTimeFromIdentity(p.Identity)
		age := at.Sub(start)
		old := ageKnown && age >= s.policy.OrphanNvimAge
		if !old && (!o.RSSKnown || o.RSSBytes <= s.policy.OrphanRSSBytes) {
			continue
		}
		command, calls, err := r.Command(ctx, p)
		sample.Cost.CommandCalls += calls
		sample.Cost.NativeCalls += calls
		if err != nil {
			sample.Errors = append(sample.Errors, fmt.Sprintf("candidate %d command: %v", p.PID, err))
			continue
		}
		sample.Commands[o.Key] = command.Args
		if !old {
			continue
		}
		cwd, calls, err := r.Cwd(ctx, p)
		sample.Cost.CwdCalls += calls
		sample.Cost.NativeCalls += calls
		probe := couchcore.ResourceOrphanProbe{Age: age, AgeKnown: ageKnown}
		if err != nil {
			sample.Errors = append(sample.Errors, fmt.Sprintf("candidate %d cwd: %v", p.PID, err))
		} else {
			probe.Cwd, probe.CwdKnown = cwd, true
		}
		sample.OrphanProbes[o.Key] = probe
	}
	if ctx.Err() != nil {
		return false
	}
	next, v, _ := couchcore.AdvanceResources(s.state, sample, s.policy)
	if s.leftRunning != nil {
		entries, err := s.registeredLeftRunning(ctx, sample.TableKnown, rows)
		if err != nil {
			v.Errors = append(v.Errors, "orphan registry: "+err.Error())
		}
		v.LeftRunning = entries
	}
	if s.alerts != nil {
		if err := s.alerts(ctx, &next); err != nil {
			v.Errors = append(v.Errors, "resource alerts: "+err.Error())
		}
	}
	s.state = next
	v.Episodes = next.Episodes
	if s.persist != nil {
		cost, err := s.persist(ctx, cloneResourceView(v))
		v.Cost.StorageBytes = cost.StorageBytes
		v.Cost.StorageWrites = cost.StorageWrites
		if err != nil {
			v.Errors = append(v.Errors, "resource persistence: "+err.Error())
		}
	}
	if s.history != nil {
		v.HistoryOldest, v.HistoryFrames = s.history()
	}
	v = cloneResourceView(projectResourceView(v)) // Include bounded wire projection and cache construction in sampler CPU.
	end, n, endErr := r.SelfCPU(ctx)
	v.Cost.NativeCalls += n
	if endErr != nil {
		v.Errors = append(v.Errors, "sampler CPU end: "+endErr.Error())
	}
	if cpuErr == nil && endErr == nil && end >= start {
		v.Cost.CPUTimeNS = end - start
		v.Cost.CPUKnown = true
	} else if cpuErr == nil && endErr == nil {
		v.Errors = append(v.Errors, "sampler CPU counter moved backwards")
	}
	v.Cost.WallTime = time.Since(wall)
	s.cached.Store(&v)
	return true
}

// registeredLeftRunning drops the registry entries whose exact identity this
// pass's table no longer holds (a table the pass could not read prunes
// nothing) and returns the rest.
func (s *resourceSampler) registeredLeftRunning(ctx context.Context, tableKnown bool, rows map[couchcore.ResourceProcessKey]procutil.Process) ([]couchcore.LeftRunningOrphan, error) {
	if tableKnown {
		gone := func(p couchcore.ProcessIdentity) bool {
			_, ok := rows[couchcore.ResourceProcessKey{PID: p.PID, Identity: p.Identity}]
			return !ok
		}
		if err := s.leftRunning.Prune(ctx, gone); err != nil {
			return nil, err
		}
	}
	return s.leftRunning.Read()
}

func (s *resourceSampler) view() couchcore.ResourceView {
	if s == nil {
		return couchcore.ResourceView{Errors: []string{"resource sampler unavailable"}}
	}
	if v := s.cached.Load(); v != nil {
		return cloneResourceView(*v)
	}
	return couchcore.ResourceView{Errors: []string{"resource sampler warming up"}}
}
func resourceCopyPointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func cloneResourceWorkspace(w couchcore.WorkspaceIdentity) couchcore.WorkspaceIdentity {
	w.EnvironmentHost = resourceCopyPointer(w.EnvironmentHost)
	w.Address = resourceCopyPointer(w.Address)
	w.Slot = resourceCopyPointer(w.Slot)
	w.Branch = resourceCopyPointer(w.Branch)
	w.Head = resourceCopyPointer(w.Head)
	w.RestingBranch = resourceCopyPointer(w.RestingBranch)
	return w
}
func cloneResourceView(v couchcore.ResourceView) couchcore.ResourceView {
	if v.SupervisionLog != nil {
		health := *v.SupervisionLog
		v.SupervisionLog = &health
	}
	if v.HostedIngest != nil {
		hosted := *v.HostedIngest
		hosted.Clients = append([]couchcore.HostedIngestClient(nil), hosted.Clients...)
		v.HostedIngest = &hosted
	}
	v.Errors = append([]string(nil), v.Errors...)
	v.LeftRunning = append([]couchcore.LeftRunningOrphan(nil), v.LeftRunning...)
	v.Slots = append([]couchcore.ResourceSlotView(nil), v.Slots...)
	for i := range v.Slots {
		p := &v.Slots[i]
		p.CPUPercent = resourceCopyPointer(p.CPUPercent)
		p.RSSBytes = resourceCopyPointer(p.RSSBytes)
		p.WriteBytesPerSecond = resourceCopyPointer(p.WriteBytesPerSecond)
	}
	cloneProcesses := func(in []couchcore.ResourceProcessView) []couchcore.ResourceProcessView {
		out := append([]couchcore.ResourceProcessView(nil), in...)
		for i := range out {
			p := &out[i]
			p.CPUPercent = resourceCopyPointer(p.CPUPercent)
			p.RSSBytes = resourceCopyPointer(p.RSSBytes)
			p.WriteBytesPerSecond = resourceCopyPointer(p.WriteBytesPerSecond)
			p.Workspace = cloneResourceWorkspace(p.Workspace)
		}
		return out
	}
	v.Processes = cloneProcesses(v.Processes)
	v.Orphans = cloneProcesses(v.Orphans)
	v.Unattributed = cloneProcesses(v.Unattributed)
	v.OrphanNvim = append([]couchcore.OrphanNvim(nil), v.OrphanNvim...)
	for i := range v.OrphanNvim {
		v.OrphanNvim[i].RSSBytes = resourceCopyPointer(v.OrphanNvim[i].RSSBytes)
	}
	v.Episodes = append([]couchcore.ResourceEpisode(nil), v.Episodes...)
	for i := range v.Episodes {
		e := &v.Episodes[i]
		e.Effect = resourceCopyPointer(e.Effect)
		if e.Effect != nil {
			e.Effect.Workspace = cloneResourceWorkspace(e.Effect.Workspace)
		}
	}
	return v
}
