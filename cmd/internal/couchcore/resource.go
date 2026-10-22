package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"path/filepath"
	"sort"
	"time"
)

const ResourceMaxProcesses = 8192
const ResourceMaxTableRows = 65536
const ResourceMaxSlots = 256
const ResourceMaxEpisodes = 2*ResourceMaxProcesses + 2*ResourceMaxSlots

type ResourceProcessKey struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}
type ResourceRoot struct {
	Key       ResourceProcessKey `json:"key"`
	Slot      string             `json:"slot"`
	Thread    ThreadAddress      `json:"thread"`
	Workspace WorkspaceIdentity  `json:"workspace"`
}
type ResourceSlot struct {
	Reason         string            `json:"reason,omitempty"`
	Slot           string            `json:"slot"`
	Thread         ThreadAddress     `json:"thread"`
	Workspace      WorkspaceIdentity `json:"workspace"`
	OwnershipKnown bool              `json:"ownership_known"`
	IdleKnown      bool              `json:"idle_known"`
	Idle           bool              `json:"idle"`
	Roots          []ResourceRoot    `json:"roots"`
}

// ResourceSlotInventory distinguishes a completed census from startup, cancellation
// or bounded omission. Unknown inventory must never prove slot disappearance.
type ResourceSlotInventory struct {
	Slots    []ResourceSlot
	Complete bool
	Omitted  int
}

type ResourceProvenance struct {
	ResourceRoot
	Root        ResourceProcessKey `json:"root"`
	Orphan      bool               `json:"orphan"`
	OrphanKnown bool               `json:"orphan_known"`
}
type ResourceUsageObservation struct {
	Key                                 ResourceProcessKey
	CPUTimeNS, RSSBytes, DiskWriteBytes uint64
	CPUKnown, RSSKnown, DiskKnown       bool
	Reason                              string
}
type ResourceCost struct {
	CPUKnown      bool   `json:"cpu_known"`
	StorageBytes  uint64 `json:"storage_bytes"`
	StorageWrites int    `json:"storage_writes"`
	TableCalls    int    `json:"table_calls"`
	UsageCalls    int    `json:"usage_calls"`
	CommandCalls  int    `json:"command_calls"`
	// EnvironmentCalls read launchd children's inherited thread (pair#606).
	EnvironmentCalls int `json:"environment_calls"`
	// CwdCalls read a candidate's working directory (pair#583).
	CwdCalls    int           `json:"cwd_calls"`
	NativeCalls int           `json:"native_calls"`
	CPUTimeNS   uint64        `json:"cpu_time_ns"`
	WallTime    time.Duration `json:"wall_time"`
}
type ResourceSample struct {
	At                time.Time
	Table             []procutil.Process
	TableKnown        bool
	InventoryComplete bool
	InventoryOmitted  int
	Slots             []ResourceSlot
	Observations      []ResourceUsageObservation
	Commands          map[ResourceProcessKey][]string
	// OrphanProbes carries the cwd/age evidence the sampler gathered for the
	// candidates worth probing; a row with no entry is unproven, not clean.
	OrphanProbes map[ResourceProcessKey]ResourceOrphanProbe
	Errors       []string
	Cost         ResourceCost
}
type ResourceProcessView struct {
	ResourceProvenance
	Name                string   `json:"name"`
	CPUPercent          *float64 `json:"cpu_percent"`
	RSSBytes            *uint64  `json:"rss_bytes"`
	WriteBytesPerSecond *float64 `json:"write_bytes_per_second"`
	Reason              string   `json:"reason,omitempty"`
}
type ResourceSlotView struct {
	ObservedCPUTimeNS      uint64        `json:"observed_cpu_time_ns"`
	ObservedDiskWriteBytes uint64        `json:"observed_disk_write_bytes"`
	Slot                   string        `json:"slot"`
	Thread                 ThreadAddress `json:"thread"`
	CPUPercent             *float64      `json:"cpu_percent"`
	RSSBytes               *uint64       `json:"rss_bytes"`
	WriteBytesPerSecond    *float64      `json:"write_bytes_per_second"`
	ProcessCount           int           `json:"process_count"`
	IdleKnown              bool          `json:"idle_known"`
	Idle                   bool          `json:"idle"`
	Reason                 string        `json:"reason,omitempty"`
}

// HostedIngestClient counts bytes admitted by one hosted Child lifetime, not
// retained output, acknowledged rendering, or the agent wrapper's scrollback.
type HostedIngestClient struct {
	EndpointID  string             `json:"endpoint_id"`
	Thread      ThreadAddress      `json:"thread"`
	Process     ResourceProcessKey `json:"process"`
	IngestBytes uint64             `json:"ingest_bytes"`
}

// HostedIngestSnapshot is fresh request-time diagnostics. Match both server and
// endpoint incarnations before computing deltas; missing rows are not zeros.
type HostedIngestSnapshot struct {
	SampledAt       time.Time            `json:"sampled_at"`
	ServerPID       int                  `json:"server_pid"`
	ServerStartedAt time.Time            `json:"server_started_at"`
	Clients         []HostedIngestClient `json:"clients"`
	Omitted         int                  `json:"omitted"`
}

// SupervisionLogHealth describes acknowledgments by the optional fact sink,
// not fsync durability. LastError remains historical after a later recovery.
type SupervisionLogHealth struct {
	State          string    `json:"state"`
	Offered        uint64    `json:"offered"`
	Written        uint64    `json:"written"`
	Dropped        uint64    `json:"dropped"`
	LastWriteAt    time.Time `json:"last_write_at"`
	ErrorOperation string    `json:"error_operation,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	LastErrorAt    time.Time `json:"last_error_at"`
}

type ResourceView struct {
	SupervisionLog *SupervisionLogHealth `json:"supervision_log,omitempty"`
	HostedIngest   *HostedIngestSnapshot `json:"hosted_ingest,omitempty"`
	HistoryOldest  time.Time             `json:"history_oldest"`
	HistoryFrames  int                   `json:"history_frames"`
	SampledAt      time.Time             `json:"sampled_at"`
	Slots          []ResourceSlotView    `json:"slots"`
	Processes      []ResourceProcessView `json:"processes"`
	Orphans        []ResourceProcessView `json:"orphans"`
	Unattributed   []ResourceProcessView `json:"unattributed"`
	OrphanNvim     []OrphanNvim          `json:"orphan_nvim"`
	// LeftRunning is the orphan registry (pair#695): what teardown left
	// running, pruned to the identities this sample still saw.
	LeftRunning            []LeftRunningOrphan `json:"left_running"`
	Errors                 []string            `json:"errors"`
	Cost                   ResourceCost        `json:"cost"`
	ObservedCPUTimeNS      uint64              `json:"observed_cpu_time_ns"`
	ObservedDiskWriteBytes uint64              `json:"observed_disk_write_bytes"`
	Episodes               []ResourceEpisode   `json:"episodes"`
}
type ResourceSelection struct {
	Processes []ResourceProvenance
	Errors    []string
}

func resourceKey(p procutil.Process) ResourceProcessKey { return ResourceProcessKey{p.PID, p.Identity} }
func resourceSort(ps []ResourceProvenance) {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Key.PID != ps[j].Key.PID {
			return ps[i].Key.PID < ps[j].Key.PID
		}
		return ps[i].Key.Identity < ps[j].Key.Identity
	})
}

// SelectResourceProcesses follows only unique, generation-identified ancestry.
// Remembered ownership survives reparenting; missing evidence never proves orphanhood.
func SelectResourceProcesses(table []procutil.Process, slots []ResourceSlot, prior []ResourceProvenance, known, inventoryComplete bool) ResourceSelection {
	out := ResourceSelection{}
	rows := map[int]procutil.Process{}
	duplicate := map[int]bool{}
	if len(table) > ResourceMaxTableRows {
		out.Errors = append(out.Errors, "process limit exceeded")
		known = false
		table = table[:ResourceMaxTableRows]
	}
	for _, p := range table {
		if _, ok := rows[p.PID]; ok {
			duplicate[p.PID] = true
		}
		rows[p.PID] = p
	}
	if len(slots) > ResourceMaxSlots {
		out.Errors = append(out.Errors, "slot limit exceeded")
		slots = slots[:ResourceMaxSlots]
		inventoryComplete = false
	}
	roots := map[ResourceProcessKey][]ResourceRoot{}
	own := map[string]ResourceSlot{}
	for _, s := range slots {
		own[s.Slot] = s
		for _, r := range s.Roots {
			if r.Slot == "" {
				r.Slot = s.Slot
			}
			if r.Thread == (ThreadAddress{}) {
				r.Thread = s.Thread
			}
			if r.Workspace.RepoIdentity == "" {
				r.Workspace = s.Workspace
			}
			if s.OwnershipKnown && r.Key.Identity != "" {
				roots[r.Key] = append(roots[r.Key], r)
			}
		}
	}
	old := map[ResourceProcessKey]ResourceProvenance{}
	for _, p := range prior {
		old[p.Key] = p
	}
	ancestry := map[int][]ResourceRoot{}
	invalid := map[int]bool{}
	for _, p := range table {
		k := resourceKey(p)
		if p.Zombie || p.Identity == "" || duplicate[p.PID] {
			continue
		}
		var candidates []ResourceRoot
		seen := map[int]bool{}
		path := []procutil.Process{}
		cur := p
		cycle := false
		for {
			if cached, ok := ancestry[cur.PID]; ok {
				candidates = cached
				cycle = invalid[cur.PID]
				break
			}
			if seen[cur.PID] || duplicate[cur.PID] {
				cycle = true
				break
			}
			seen[cur.PID] = true
			path = append(path, cur)
			if cur.Identity == "" {
				break
			}
			parent, ok := rows[cur.PPID]
			if !ok || cur.PPID <= 0 {
				break
			}
			cur = parent
		}
		for i := len(path) - 1; i >= 0; i-- {
			row := path[i]
			if !cycle {
				joined := append([]ResourceRoot{}, roots[resourceKey(row)]...)
				joined = append(joined, candidates...)
				candidates = joined
			}
			ancestry[row.PID] = candidates
			invalid[row.PID] = cycle
		}

		if cycle {
			out.Errors = append(out.Errors, "cyclic or ambiguous process ancestry")
			continue
		}
		previous, had := old[k]
		conflict := false
		var chosen ResourceRoot
		if had {
			chosen = previous.ResourceRoot
		} else if len(candidates) > 0 {
			chosen = candidates[0]
		}
		for _, r := range candidates {
			if r.Slot != chosen.Slot || r.Thread != chosen.Thread || r.Workspace.RepoIdentity != chosen.Workspace.RepoIdentity || r.Workspace.WorktreeRoot != chosen.Workspace.WorktreeRoot {
				conflict = true
			}
		}
		if conflict {
			out.Errors = append(out.Errors, "conflicting process ownership")
			continue
		}
		if chosen.Slot == "" {
			continue
		}
		pv := ResourceProvenance{ResourceRoot: chosen, Root: chosen.Key, OrphanKnown: known}
		pv.Key = k
		if had {
			pv = previous
			pv.Orphan = false
			// A remembered root that died while a live root of the same thread
			// still reaches this process (an agent relaunched under its detached
			// suite, #606) hands ownership over instead of orphaning it.
			if rp, alive := rows[pv.Root.PID]; known && (!alive || rp.Zombie || rp.Identity != pv.Root.Identity) {
				for _, c := range candidates {
					if c.Slot == pv.Slot && c.Thread == pv.Thread {
						pv.ResourceRoot, pv.Root = c, c.Key
						pv.Key = k
						break
					}
				}
			}
			_, rootAlive := rows[pv.Root.PID]
			rp := rows[pv.Root.PID]
			s, exists := own[pv.Slot]
			// Exact root death/reuse is positive process evidence, independent of
			// the slot census. A live root plus an omitted slot is only unknown.
			rootDead := known && !duplicate[pv.Root.PID] && (!rootAlive || rp.Zombie || rp.Identity != "" && rp.Identity != pv.Root.Identity)
			ownerKnown := exists && s.OwnershipKnown || !exists && inventoryComplete
			pv.OrphanKnown = rootDead || known && ownerKnown && rootAlive && rp.Identity != "" && !duplicate[pv.Root.PID]
			pv.Orphan = rootDead || pv.OrphanKnown && (len(candidates) == 0 || exists && s.Thread != pv.Thread)
		}
		out.Processes = append(out.Processes, pv)
	}
	for _, old := range prior {
		if row, ok := rows[old.Key.PID]; ok && row.Identity == "" && !row.Zombie {
			old.Orphan = false
			old.OrphanKnown = false
			out.Processes = append(out.Processes, old)
			out.Errors = append(out.Errors, "process identity unavailable")
		}
	}
	// On incomplete tables keep absent generations for a later authoritative sample.
	if !known {
		present := map[ResourceProcessKey]bool{}
		for _, p := range out.Processes {
			present[p.Key] = true
		}
		for _, p := range prior {
			if _, observedPID := rows[p.Key.PID]; !present[p.Key] && !observedPID {
				p.Orphan = false
				p.OrphanKnown = false
				out.Processes = append(out.Processes, p)
			}
		}
	}
	resourceSort(out.Processes)
	sort.SliceStable(out.Processes, func(i, j int) bool {
		a, b := out.Processes[i], out.Processes[j]
		ar, br := len(roots[a.Key]) > 0, len(roots[b.Key]) > 0
		if ar != br {
			return ar
		}
		return !a.Orphan && b.Orphan
	})
	if len(out.Processes) > ResourceMaxProcesses {
		out.Errors = append(out.Errors, "selected process limit exceeded")
		out.Processes = out.Processes[:ResourceMaxProcesses]
	}
	resourceSort(out.Processes)
	return out
}

// unattributedKinds is the closed set of process kinds this supervisor knows the
// shape of, built once from the shared agent inventory instead of per pass: the
// classifier walks the whole process table, and rebuilding the set for every row
// would be the sampler's most pointless allocation. `nvim` is deliberately absent
// — its kind is only known once argv has been read, which is UnattributedKind's
// job to say and UnattributedCandidate's to prove.
var unattributedKinds = func() map[string]bool {
	kinds := map[string]bool{"zellij": true}
	for _, name := range launcher.AgentInventory() {
		kinds[name] = true
	}
	return kinds
}()

// UnattributedKind reports whether a process name is a kind this supervisor
// knows the shape of, before its argv has been read. The sampler uses it to
// decide which rows are worth a Command read; UnattributedCandidate then applies
// the argv rule the kind needs. Two callers, one kind set (#583, ARCH-DRY).
func UnattributedKind(name string) bool { return name == "nvim" || unattributedKinds[name] }

// UnattributedShape reports the row-level half of the candidate rule: at launchd,
// not a zombie, a generation worth fencing on, and unclaimed by any slot. It says
// nothing about kind, so the sampler can use it to decide which rows are worth an
// argv read before any argv exists.
func UnattributedShape(p procutil.Process, claimed map[ResourceProcessKey]bool) bool {
	return p.PPID == 1 && !p.Zombie && p.PID > 0 && p.Identity != "" && !claimed[resourceKey(p)]
}

// UnattributedCandidate reports whether p is a process of a known kind, sitting
// at launchd with no owner this supervisor can name: UnattributedShape, plus the
// kind rule — and for nvim, whose kind argv alone reveals, an exact `--embed`
// field. It grants no ownership and decides nothing on its own; it is the shared
// rule behind #587's unattributed report and #583's orphan sweep, so the two
// cannot drift apart.
//
// A pid that is not positive cannot arrive from either platform's table, and the
// predicate now also gates a kill path, so it refuses rather than trusting the
// caller's filter.
func UnattributedCandidate(p procutil.Process, claimed map[ResourceProcessKey]bool, args []string) bool {
	if !UnattributedShape(p, claimed) {
		return false
	}
	name := filepath.Base(p.Name)
	if name != "nvim" {
		return unattributedKinds[name]
	}
	for _, arg := range args {
		if arg == "--embed" {
			return true
		}
	}
	return false
}

// ClassifyUnattributed reports known launchd children only; it grants no ownership.
// commands must have been generation-fenced by the caller around command IO.
func ClassifyUnattributed(table []procutil.Process, owned []ResourceProvenance, observations []ResourceUsageObservation, commands map[ResourceProcessKey][]string, floor uint64) []ResourceProcessView {
	claimed := map[ResourceProcessKey]bool{}
	for _, p := range owned {
		claimed[p.Key] = true
	}
	obs := map[ResourceProcessKey]ResourceUsageObservation{}
	for _, o := range observations {
		obs[o.Key] = o
	}
	var out []ResourceProcessView
	for i, p := range table {
		if i >= ResourceMaxTableRows || len(out) >= ResourceMaxProcesses-len(owned) {
			break
		}
		k := resourceKey(p)
		o := obs[k]
		if !o.RSSKnown || o.RSSBytes <= floor {
			continue
		}
		if !UnattributedCandidate(p, claimed, commands[k]) {
			continue
		}
		rss := o.RSSBytes
		out = append(out, ResourceProcessView{ResourceProvenance: ResourceProvenance{ResourceRoot: ResourceRoot{Key: k}}, Name: filepath.Base(p.Name), RSSBytes: &rss, Reason: "unattributed (unproven)"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.PID < out[j].Key.PID })
	return out
}
