package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type samplerReader struct {
	mu                             sync.Mutex
	rows                           []procutil.Process
	tableErr, errorUsage, errorCPU error
	usages                         []int
	commands, environments         []int
	cwds                           []int
	cwdPath                        string
	cwdErr                         error
	env                            map[int]map[string]string
	unstableEnv                    int
	tables, clocks                 int
	tableHook                      func(context.Context)
	cpu                            uint64
}

func (f *samplerReader) Table(ctx context.Context) ([]procutil.Process, int, error) {
	f.mu.Lock()
	f.tables++
	hook := f.tableHook
	rows, err := f.rows, f.tableErr
	f.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return rows, 2, err
}

// setRows swaps the whole table the sampler will see next. The sampler runs on
// its own worker, so a sweep test writes the table concurrently with the pass
// that reads it; replacing the slice rather than appending to it keeps both
// sides of that race holding a complete, immutable view.
func (f *samplerReader) setRows(rows []procutil.Process) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = rows
}
func (f *samplerReader) Usage(ctx context.Context, p procutil.Process) (procutil.ResourceUsage, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usages = append(f.usages, p.PID)
	if f.errorUsage != nil {
		return procutil.ResourceUsage{}, 1, f.errorUsage
	}
	return procutil.ResourceUsage{CPUTimeNS: 100, RSSBytes: 1000, DiskWriteBytes: 100}, 2, nil
}
func (f *samplerReader) Command(ctx context.Context, p procutil.Process) (procutil.CommandInfo, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, p.PID)
	return procutil.CommandInfo{Args: []string{"nvim", "--embed"}}, 2, nil
}
func (f *samplerReader) Cwd(ctx context.Context, p procutil.Process) (string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cwds = append(f.cwds, p.PID)
	return f.cwdPath, 3, f.cwdErr
}
func (f *samplerReader) Environment(ctx context.Context, p procutil.Process, keys ...string) (map[string]string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.environments = append(f.environments, p.PID)
	if p.PID == f.unstableEnv {
		f.unstableEnv = 0
		return nil, 2, &procutil.IdentityUnavailableError{Err: errors.New("process parent changed during resource observation")}
	}
	return f.env[p.PID], 2, nil
}
func (f *samplerReader) SelfCPU(context.Context) (uint64, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clocks++
	return f.cpu, 1, f.errorCPU
}
func testResourceSampler(f *samplerReader) *resourceSampler {
	p, _ := couchcore.ParseResourcePolicy(nil)
	p.OrphanRSSBytes = 100
	return &resourceSampler{reader: f, policy: p, now: func() time.Time { return time.Unix(100, 0) }, slots: func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Roots: []couchcore.ResourceRoot{{Key: couchcore.ResourceProcessKey{PID: 10, Identity: "owned"}}}}}}
	}}
}

// A slot's suite that lost its ancestry (pair#606) is the slot's, and each
// process generation's environment is read once, not every pass.
func TestResourceSamplerAttributesADetachedSuiteToItsSlot(t *testing.T) {
	thread := couchcore.ThreadAddress{RepoScope: "scope", Tag: "1-pair-6"}
	f := &samplerReader{
		rows:        []procutil.Process{{PID: 10, Identity: "owned"}, {PID: 20, PPID: 1, Identity: "zsh", Name: "zsh"}, {PID: 21, PPID: 20, Identity: "sdlc", Name: "sdlc"}, {PID: 22, PPID: 21, Identity: "test", Name: "couchcore.test"}, {PID: 30, PPID: 1, Identity: "stray", Name: "python3"}},
		env:         map[int]map[string]string{21: {"COUCH_THREAD_SCOPE": "scope", "COUCH_THREAD_TAG": "1-pair-6"}},
		unstableEnv: 21, // the first read races a reparent; the next pass asks again
	}
	s := testResourceSampler(f)
	s.slots = func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: "pair:6", Thread: thread, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{{Key: couchcore.ResourceProcessKey{PID: 10, Identity: "owned"}}}}}}
	}
	s.pass(context.Background())
	s.pass(context.Background())
	owned := map[int]string{}
	for _, p := range s.view().Processes {
		owned[p.Key.PID] = p.Slot
	}
	if owned[21] != "pair:6" || owned[22] != "pair:6" {
		t.Fatalf("processes %v, want the detached suite on pair:6", owned)
	}
	if _, ok := owned[30]; ok {
		t.Fatal("a process naming no thread was claimed")
	}
	if !reflect.DeepEqual(f.environments, []int{20, 21, 30, 21}) {
		t.Fatalf("environment reads %v, want each settled generation once across passes", f.environments)
	}
}
func TestResourceSamplerSelectionCallsAndImmutableView(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}, {PID: 11, PPID: 10, Identity: "child"}, {PID: 12, PPID: 1, Identity: "vim", Name: "nvim"}, {PID: 13, PPID: 1, Identity: "other", Name: "unrelated"}}}
	s := testResourceSampler(f)
	s.persist = func(_ context.Context, v couchcore.ResourceView) (couchcore.ResourceCost, error) {
		f.cpu += 500
		return couchcore.ResourceCost{StorageWrites: 1, StorageBytes: 200}, nil
	}
	s.pass(context.Background())
	v := s.view()
	if !reflect.DeepEqual(f.usages, []int{10, 11, 12}) || !reflect.DeepEqual(f.commands, []int{12}) {
		t.Fatal(f.usages, f.commands)
	}
	if v.Cost.NativeCalls != 14 || v.Cost.UsageCalls != 6 || v.Cost.CommandCalls != 2 || v.Cost.EnvironmentCalls != 2 || !reflect.DeepEqual(f.environments, []int{13}) || v.Cost.TableCalls != 2 || v.Cost.CPUTimeNS != 500 || !v.Cost.CPUKnown || v.Cost.StorageWrites != 1 {
		t.Fatalf("cost=%+v", v.Cost)
	}
	if len(v.Processes) != 2 || len(v.Unattributed) != 1 {
		t.Fatalf("view=%+v", v)
	}
	*v.Processes[0].RSSBytes = 0
	v.Processes[0].Name = "mutated"
	v.Errors = append(v.Errors, "mutated")
	again := s.view()
	if *again.Processes[0].RSSBytes != 1000 || again.Processes[0].Name == "mutated" {
		t.Fatal("caller mutated cache")
	}
	if f.tables != 1 {
		t.Fatal("view queried native reader")
	}
}

// The cwd probe is a cost, so only a row that already cleared the ppid/name/age
// gate pays for it: a live-parent nvim, a young one and an unparseable stamp are
// never probed, and only the proven orphan is reported (#583).
func TestResourceSamplerProbesCwdOnlyForAnOldOrphanNvim(t *testing.T) {
	const old = "1760000000.5"
	f := &samplerReader{
		rows: []procutil.Process{
			{PID: 40, PPID: 1, Identity: old, Name: "nvim"},
			{PID: 41, PPID: 1, Identity: "1760099999.5", Name: "nvim"},
			{PID: 42, PPID: 40, Identity: "1760000000.6", Name: "nvim"},
			{PID: 43, PPID: 1, Identity: "unparseable", Name: "nvim"},
			{PID: 44, PPID: 1, Identity: old, Name: "zellij"},
		},
		cwdPath: "/fleet/worktree/pair-slot1/pair/cmd",
	}
	s := testResourceSampler(f)
	at := time.Unix(1760100000, 0)
	s.now = func() time.Time { return at }
	s.slots = func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Workspace: couchcore.WorkspaceIdentity{WorktreeRoot: "/fleet/worktree/pair-slot1/pair"}}}}
	}
	s.pass(context.Background())
	v := s.view()
	if !reflect.DeepEqual(f.cwds, []int{40}) {
		t.Fatalf("cwd probes %v, want only the old orphan nvim", f.cwds)
	}
	if !reflect.DeepEqual(f.commands, []int{40, 41, 43}) {
		t.Fatalf("argv reads %v, want every nvim candidate", f.commands)
	}
	if len(v.OrphanNvim) != 1 || v.OrphanNvim[0].Key.PID != 40 || v.OrphanNvim[0].Root != "/fleet/worktree/pair-slot1/pair" || v.OrphanNvim[0].Age != at.Sub(time.Unix(1760000000, 5000)) {
		t.Fatalf("orphans %+v errors %v", v.OrphanNvim, v.Errors)
	}
	if len(v.Unattributed) != 3 {
		t.Fatalf("unattributed %+v", v.Unattributed)
	}
	for _, row := range v.Unattributed {
		if row.Key.PID == 40 {
			t.Fatalf("the specific sweep must subsume the generic row: %+v", v.Unattributed)
		}
	}
	if v.Cost.CwdCalls != 3 || v.Cost.NativeCalls < 3 {
		t.Fatalf("cost=%+v; the probe must be charged", v.Cost)
	}
	*v.OrphanNvim[0].RSSBytes = 0
	v.OrphanNvim[0].Cwd = "mutated"
	if again := s.view(); *again.OrphanNvim[0].RSSBytes != 1000 || again.OrphanNvim[0].Cwd == "mutated" {
		t.Fatal("caller mutated the cached orphan row")
	}
	// An unread cwd is an unproven orphan, and the failure is visible.
	f.cwds, f.cwdPath, f.cwdErr = nil, "", errors.New("cwd denied")
	s.pass(context.Background())
	v = s.view()
	if len(v.OrphanNvim) != 0 || !strings.Contains(strings.Join(v.Errors, " "), "candidate 40 cwd: cwd denied") {
		t.Fatalf("orphans %+v errors %v", v.OrphanNvim, v.Errors)
	}
}
func TestResourceSamplerFailuresRemainUnknown(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}, errorUsage: errors.New("usage denied"), errorCPU: errors.New("CPU denied")}
	s := testResourceSampler(f)
	s.pass(context.Background())
	v := s.view()
	if len(v.Processes) != 1 || v.Processes[0].RSSBytes != nil || v.Cost.CPUKnown || !strings.Contains(strings.Join(v.Errors, " "), "CPU denied") {
		t.Fatalf("view=%+v", v)
	}
	f.tableErr = errors.New("table denied")
	f.rows = nil
	s.pass(context.Background())
	v = s.view()
	if len(v.Processes) != 1 || v.Processes[0].OrphanKnown || !strings.Contains(strings.Join(v.Errors, " "), "table denied") {
		t.Fatalf("view=%+v", v)
	}
}
func TestResourceSamplerBlockedCollectionServesCacheAndJoins(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	s := testResourceSampler(f)
	s.pass(context.Background())
	s.policy.Cadence = time.Millisecond
	entered := make(chan struct{})
	f.tableHook = func(ctx context.Context) { close(entered); <-ctx.Done() }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.run(ctx); close(done) }()
	<-entered
	got := make(chan couchcore.ResourceView, 1)
	go func() { got <- s.view() }()
	select {
	case v := <-got:
		if len(v.Processes) != 1 {
			t.Fatal(v)
		}
	case <-time.After(time.Second):
		t.Fatal("cached view blocked")
	}
	// A full cadence elapses while the first pass remains blocked: no overlap.
	time.Sleep(5 * time.Millisecond)
	f.mu.Lock()
	tables := f.tables
	f.mu.Unlock()
	if tables != 2 {
		t.Fatal("overlapping sample", tables)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sampler failed to join")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tables != 2 {
		t.Fatal("sample launched after cancellation")
	}
}

func TestResourceSamplerCombinedCapKeepsOwnedFirst(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	for i := 0; i < couchcore.ResourceMaxProcesses; i++ {
		f.rows = append(f.rows, procutil.Process{PID: 100 + i, PPID: 1, Identity: "candidate", Name: "zellij"})
	}
	s := testResourceSampler(f)
	s.pass(context.Background())
	v := s.view()
	if len(f.usages) != couchcore.ResourceMaxProcesses || f.usages[0] != 10 || len(f.commands) != 0 {
		t.Fatal("combined cap or owned priority", len(f.usages), f.usages[0], len(f.commands))
	}
	if len(v.Processes) != 1 || len(v.Unattributed) == 0 || len(v.Unattributed) >= couchcore.ResourceMaxProcesses-1 || !strings.Contains(strings.Join(v.Errors, " "), "limit exceeded") {
		t.Fatal("missing partial coverage", len(v.Processes), len(v.Unattributed), v.Errors)
	}
	if !strings.Contains(strings.Join(v.Errors, " "), fmt.Sprintf("%d unattributed processes omitted", couchcore.ResourceMaxProcesses-1-len(v.Unattributed))) {
		t.Fatal("missing exact wire omission count", v.Errors)
	}
	if v.Cost.NativeCalls > 8+2*len(f.usages)+v.Cost.CommandCalls {
		t.Fatal("native call envelope", v.Cost)
	}
}
func TestResourceSamplerPersistenceFailureKeepsAdmittedState(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	s := testResourceSampler(f)
	s.alerts = func(_ context.Context, state *couchcore.ResourceState) error {
		state.Episodes = []couchcore.ResourceEpisode{{Key: "admitted", Phase: "notified", Effect: &couchcore.ResourceEffect{ID: "stable"}}}
		return nil
	}
	s.persist = func(_ context.Context, v couchcore.ResourceView) (couchcore.ResourceCost, error) {
		return couchcore.ResourceCost{}, errors.New("history failed")
	}
	s.pass(context.Background())
	v := s.view()
	if len(s.state.Episodes) != 1 || s.state.Episodes[0].Phase != "notified" || !strings.Contains(strings.Join(v.Errors, " "), "history failed") {
		t.Fatal(s.state.Episodes, v.Errors)
	}
	v.Episodes[0].Effect.ID = "mutated"
	if s.view().Episodes[0].Effect.ID != "stable" || s.state.Episodes[0].Effect.ID != "stable" {
		t.Fatal("episode alias escaped")
	}
}
func TestResourceSamplerCandidateFloorAvoidsCommand(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}, {PID: 12, PPID: 1, Identity: "vim", Name: "nvim"}, {PID: 13, PPID: 2, Identity: "vim2", Name: "nvim"}, {PID: 14, PPID: 1, Identity: "zombie", Name: "nvim", Zombie: true}, {PID: 15, PPID: 1, Name: "nvim"}}}
	s := testResourceSampler(f)
	s.policy.OrphanRSSBytes = 1000
	s.pass(context.Background())
	if !reflect.DeepEqual(f.usages, []int{10, 12}) || len(f.commands) != 0 {
		t.Fatal(f.usages, f.commands)
	}
	if len(s.view().Unattributed) != 0 {
		t.Fatal("below-floor candidate included")
	}
}

// costSteadyRatioK is the steady pass's CPU budget in units of the same-run
// history-write baseline: a steady pass may not cost more than twice the write it
// performs. Derived in pair#644 from runs across the fleet's load range, not tuned
// per run: K is headroom over the worst steady ratio measured anywhere in that
// range, and those runs are recorded in the issue's Log. It is also the native
// backstop for #606's environment memo — a pass shaped like the cold one, which
// reads its candidates' environments, costs several times a write and fails here too.
const costSteadyRatioK = 2.0

// TestResourceSamplerNativeCost measures the complete pass, including reducer,
// serialization, validation and snapshot publication at nearly full history.
// Native collection follows the real bounded selector; only the test root is
// owned, and other rows are eligible only for the ordinary PPID1 candidate path.
//
// The verdict is load-invariant (#644). Absolute CPU cannot give one answer on an
// idle host and on one running the rest of the fleet: measured across a wide load
// range on unchanged code, a pass's native call counts barely move while its CPU
// moves by tens of percent, which flipped the old fixed three-pass mean budget. So
// the work a pass does is checked against bounds the code's own structure sets, and
// a steady pass's CPU is checked against the history write it performs, measured in
// the same run. The first pass is #606's one-time environment scan: work-bounded,
// CPU reported and not asserted, per #587's ruling that the binding budget governs
// steady sampling rather than a rare maximum-cardinality admission burst. The runs
// behind every constant here are recorded in #644's Log.
func TestResourceSamplerNativeCost(t *testing.T) {
	if os.Getenv("PAIR_RESOURCE_COST") != "1" {
		t.Skip("set PAIR_RESOURCE_COST=1 for native cost conformance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reader := procutil.NativeResourceReader{}
	rows, _, err := reader.Table(ctx)
	if errors.Is(err, procutil.ErrUnsupported) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	var owned procutil.Process
	for _, p := range rows {
		if p.PID == os.Getpid() {
			owned = p
		}
	}
	if owned.Identity == "" {
		t.Fatal("owned test process absent")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(dir, "pair")
	slot := 0
	address := "pair:0"
	branch := "main"
	workspace := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "pair", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: dir, EnvironmentRoot: dir, WorktreeRoot: primary, Kind: "primary", Slot: &slot, Address: &address, RestingBranch: &branch}
	persistence := newResourceHistoryPersistence(dir)
	if persistence.loadError != nil {
		t.Fatal(persistence.loadError)
	}
	base := time.Now()
	seed := couchcore.ResourceHistory{Version: 1}
	frameSlots := make([]couchcore.ResourceSlotView, 16)
	for i := range frameSlots {
		frameSlots[i] = couchcore.ResourceSlotView{Slot: fmt.Sprintf("seed:%d", i), Reason: "partial or warming-up", ProcessCount: 3}
	}
	size := len(`{"version":1,"frames":[]}`)
	for i := 0; i < 1440; i++ {
		f := couchcore.ResourceFrame{At: base.Add(time.Duration(i-1439) * time.Minute), Slots: frameSlots}
		raw, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		if size+len(raw)+1 > couchcore.ResourceHistoryLimit-4096 {
			break
		}
		size += len(raw) + 1
		seed.Frames = append(seed.Frames, f)
	}
	raw, err := json.Marshal(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err = persistence.store.Save(raw); err != nil {
		t.Fatal(err)
	}
	persistence.history = seed
	if len(raw) < couchcore.ResourceHistoryLimit-65536 {
		t.Fatalf("fixture did not approach history cap: %d", len(raw))
	}
	baselineDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	baseline := couchcore.ResourceHistoryStore{Dir: filepath.Join(baselineDir, "resources")}
	writeCPU := historyWriteCPU(t, ctx, reader, baseline, seed)
	policy, _ := couchcore.ParseResourcePolicy(nil)
	// Maximum cardinality: the widest gates this pass has, so nothing is filtered out
	// by a threshold this host happens to sit under. The age floor admits every PPID-1
	// nvim to the argv read and the cwd probe, and the RSS floor -- which is also
	// #587's unattributed report floor -- admits every known-kind launchd child to the
	// report and its serialization. The bounds below are asserted against that worst
	// case rather than against the defaults, because the defaults are a host-shaped
	// accident and the probe's cost is not (#583).
	policy.OrphanNvimAge, policy.OrphanRSSBytes = time.Second, 0
	sampler := &resourceSampler{reader: reader, policy: policy, persist: persistence.persist, slots: func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: address, Workspace: workspace, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{{Key: couchcore.ResourceProcessKey{PID: owned.PID, Identity: owned.Identity}}}}}}
	}}
	const passes = 3
	// workBound is one load-invariant assertion about what a pass did. `got`
	// carries the counters behind it so a failure names the evidence, and every
	// failure names the host state too.
	type workBound struct {
		name string
		ok   bool
		got  string
	}
	var coldCPU, steadySum, steadyMax uint64
	steady := 0
	for pass := 0; pass < passes; pass++ {
		// The bounds' basis: the table as this test sees it immediately before the
		// pass, which reads its own a few milliseconds later. The row-scaled bounds
		// below are multiples of that count, and the slack is what absorbs the churn
		// in the gap — the point is to catch a multiplied read, not a birth. How much
		// slack a run actually had is measured in #644's Log, not guaranteed here.
		table, _, err := reader.Table(ctx)
		if err != nil {
			t.Fatal(err)
		}
		basis, load := len(table), hostLoadAverage()
		host := fmt.Sprintf("rows=%d %s", basis, load)
		sampler.now = func() time.Time { return base.Add(time.Duration(pass) * time.Minute) }
		sampler.pass(ctx)
		v := sampler.view()
		if !v.Cost.CPUKnown || len(v.Processes) == 0 {
			t.Fatalf("incomplete measurement: %+v errors=%v (%s)", v.Cost, v.Errors, host)
		}
		for _, e := range v.Errors {
			if strings.Contains(e, "persistence") {
				t.Fatalf("%s (%s)", e, host)
			}
		}
		c := v.Cost
		// A native read costs at most two calls — procutil charges the read plus the
		// generation fence that follows it — and no category reads a row more than
		// once per pass: usage observes selected rows and candidates, which are
		// disjoint; #606's memo allows one environment read per generation; argv is
		// read once per nvim candidate and cwd only after an argv read succeeded. A
		// failed read costs one call, so these are bounds rather than equalities.
		accounted := c.TableCalls + c.UsageCalls + c.CommandCalls + c.CwdCalls + c.EnvironmentCalls + 2
		bounds := []workBound{
			{"native accounting", c.NativeCalls == accounted, fmt.Sprintf("native=%d accounted=%d (table=%d usage=%d command=%d cwd=%d environment=%d self_cpu=2)", c.NativeCalls, accounted, c.TableCalls, c.UsageCalls, c.CommandCalls, c.CwdCalls, c.EnvironmentCalls)},
			{"table attempts", c.TableCalls <= 6, fmt.Sprintf("table_calls=%d limit=6 (3 attempts x size+read)", c.TableCalls)},
			{"usage reads", c.UsageCalls <= 2*basis, fmt.Sprintf("usage_calls=%d limit=%d (2 x rows)", c.UsageCalls, 2*basis)},
			{"environment reads", c.EnvironmentCalls <= 2*basis, fmt.Sprintf("environment_calls=%d limit=%d (2 x rows)", c.EnvironmentCalls, 2*basis)},
			{"argv reads", c.CommandCalls <= 2*basis, fmt.Sprintf("command_calls=%d limit=%d (2 x rows)", c.CommandCalls, 2*basis)},
			{"cwd reads", c.CwdCalls <= c.CommandCalls, fmt.Sprintf("cwd_calls=%d command_calls=%d", c.CwdCalls, c.CommandCalls)},
			{"history writes", c.StorageWrites == 1, fmt.Sprintf("storage_writes=%d", c.StorageWrites)},
			{"history size", c.StorageBytes >= couchcore.ResourceHistoryLimit-65536 && c.StorageBytes <= couchcore.ResourceHistoryLimit, fmt.Sprintf("storage_bytes=%d limit=%d", c.StorageBytes, couchcore.ResourceHistoryLimit)},
		}
		if pass == 0 {
			// Without this the cold bound passes vacuously if the scan is ever removed.
			bounds = append(bounds, workBound{"cold environment scan", c.EnvironmentCalls >= 2, fmt.Sprintf("environment_calls=%d", c.EnvironmentCalls)})
		}
		for _, b := range bounds {
			if !b.ok {
				t.Errorf("pass %d work bound %q exceeded: %s (%s)", pass+1, b.name, b.got, host)
			}
		}
		ratio := float64(c.CPUTimeNS) / float64(writeCPU)
		info, err := os.Stat(filepath.Join(dir, "resources", "history.json"))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("pass=%d %s slots=%d owned_processes=%d unattributed=%d orphan_nvim=%d calls=%d table_calls=%d usage_calls=%d command_calls=%d cwd_calls=%d environment_calls=%d CPU_ms=%.3f wall_ms=%.3f CPU_per_history_write=%.2f storage_bytes=%d storage_writes=%d history_bytes=%d", pass+1, host, len(v.Slots), len(v.Processes), len(v.Unattributed), len(v.OrphanNvim), c.NativeCalls, c.TableCalls, c.UsageCalls, c.CommandCalls, c.CwdCalls, c.EnvironmentCalls, float64(c.CPUTimeNS)/1e6, float64(c.WallTime)/1e6, ratio, c.StorageBytes, c.StorageWrites, info.Size())
		if pass == 0 {
			coldCPU = c.CPUTimeNS
			continue
		}
		steadySum += c.CPUTimeNS
		steady++
		if c.CPUTimeNS > steadyMax {
			steadyMax = c.CPUTimeNS
		}
		if ratio > costSteadyRatioK {
			t.Errorf("steady pass %d CPU %.3fms exceeds %.2fx the same-run history-write baseline %.3fms (%s)", pass+1, float64(c.CPUTimeNS)/1e6, costSteadyRatioK, float64(writeCPU)/1e6, host)
		}
	}
	mean := float64(steadySum) / float64(steady)
	t.Logf("history_write_baseline_ms=%.3f steady_budget_ms=%.3f cold_pass_CPU_ms=%.3f cold_CPU_per_history_write=%.2f steady_mean_CPU_ms=%.3f steady_max_CPU_ms=%.3f steady_sixty_second_CPU_pct=%.6f %s seeded_frames=%d seeded_history_bytes=%d", float64(writeCPU)/1e6, float64(writeCPU)*costSteadyRatioK/1e6, float64(coldCPU)/1e6, float64(coldCPU)/float64(writeCPU), mean/1e6, float64(steadyMax)/1e6, mean/float64(time.Minute)*100, hostLoadAverage(), len(seed.Frames), len(raw))
}

// historyWriteCPU is the mean thread CPU of marshaling and saving one near-cap
// history, measured on this host under this load. It is the steady verdict's
// denominator: a steady pass performs exactly one such write, so the ratio answers
// "how much more than the write does a pass cost", which is the one CPU question
// whose answer does not move with fleet load (#644). Its own store keeps the
// baseline out of the fixture's history.
func historyWriteCPU(t *testing.T, ctx context.Context, reader procutil.NativeResourceReader, store couchcore.ResourceHistoryStore, seed couchcore.ResourceHistory) uint64 {
	t.Helper()
	const reps = 3
	var total uint64
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for range reps {
		start, _, err := reader.SelfCPU(ctx)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(seed)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(raw); err != nil {
			t.Fatal(err)
		}
		end, _, err := reader.SelfCPU(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if end < start {
			t.Fatalf("self CPU counter moved backwards: %d < %d", end, start)
		}
		total += end - start
	}
	if total/reps == 0 {
		t.Fatalf("history-write baseline measured no CPU over %d reps; the steady ratio would be vacuous", reps)
	}
	return total / reps
}

func TestResourceSamplerHistoryFailureDoesNotGateAdvisoryAlerts(t *testing.T) {
	reader := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	sampler := testResourceSampler(reader)
	sampler.policy.ProcessRSSBytes = 500
	sends := 0
	sampler.alerts = func(_ context.Context, state *couchcore.ResourceState) error {
		for i := range state.Episodes {
			e := &state.Episodes[i]
			if e.Phase == "pending" && e.Effect != nil {
				sends++
				*e, _ = couchcore.AdvanceResourceEpisode(*e, couchcore.ResourceEpisodeEvent{Delivery: couchcore.ResourceDeliveryAdmitted, EffectID: e.Effect.ID})
			}
		}
		return nil
	}
	sampler.persist = func(_ context.Context, v couchcore.ResourceView) (couchcore.ResourceCost, error) {
		if sends != 1 {
			t.Fatalf("history attempted before alert admission: %d", sends)
		}
		return couchcore.ResourceCost{}, errors.New("history permission denied")
	}
	sampler.pass(context.Background())
	sampler.now = func() time.Time { return time.Unix(160, 0) }
	sampler.pass(context.Background())
	if sends != 1 {
		t.Fatalf("history failure repeated high episode: %d", sends)
	}
	if !strings.Contains(strings.Join(sampler.view().Errors, " "), "history permission denied") {
		t.Fatal("history failure hidden")
	}
	restarted := testResourceSampler(reader)
	restarted.policy = sampler.policy
	restarted.alerts = sampler.alerts
	restarted.now = func() time.Time { return time.Unix(220, 0) }
	restarted.pass(context.Background())
	if sends != 2 {
		t.Fatalf("fresh supervisor must re-arm advisory episode: %d", sends)
	}
}

// Each pass prunes the orphan registry against its own table and reports what
// is left; a table the pass could not read prunes nothing (pair#695).
func TestResourceSamplerPrunesAndReportsLeftRunning(t *testing.T) {
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	registry := couchcore.NewLeftRunningRegistry(ns)
	at := time.Unix(90, 0)
	if err := registry.Add(context.Background(), []couchcore.LeftRunningOrphan{
		{Origin: "slot:/w/pair", Process: couchcore.ProcessIdentity{PID: 40, Identity: "runner"}, Command: "sdlc", LeftAt: at},
		{Origin: "slot:/w/pair", Process: couchcore.ProcessIdentity{PID: 41, Identity: "gone"}, Command: "npm", LeftAt: at},
	}); err != nil {
		t.Fatal(err)
	}
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}, {PID: 40, PPID: 1, Identity: "runner", Name: "sdlc"}, {PID: 41, PPID: 1, Identity: "recycled", Name: "npm"}}}
	s := testResourceSampler(f)
	s.leftRunning = registry
	f.tableErr = errors.New("table unreadable")
	if !s.pass(context.Background()) {
		t.Fatal("pass did not run")
	}
	if got := s.view().LeftRunning; len(got) != 2 {
		t.Fatalf("an unread table pruned the registry: %+v", got)
	}
	f.tableErr = nil
	if !s.pass(context.Background()) {
		t.Fatal("pass did not run")
	}
	got := s.view().LeftRunning
	if len(got) != 1 || got[0].Process.PID != 40 {
		t.Fatalf("left running = %+v, want only the identity the table still holds", got)
	}
	if kept, _ := registry.Read(); len(kept) != 1 {
		t.Fatalf("registry after prune = %+v", kept)
	}
}
