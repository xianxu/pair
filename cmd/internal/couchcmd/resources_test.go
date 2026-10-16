package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type samplerReader struct {
	mu                             sync.Mutex
	rows                           []procutil.Process
	tableErr, errorUsage, errorCPU error
	usages                         []int
	commands                       []int
	tables, clocks                 int
	tableHook                      func(context.Context)
	cpu                            uint64
}

func (f *samplerReader) Table(ctx context.Context) ([]procutil.Process, int, error) {
	f.mu.Lock()
	f.tables++
	hook := f.tableHook
	f.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return f.rows, 2, f.tableErr
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
func (f *samplerReader) SelfCPU(context.Context) (uint64, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clocks++
	return f.cpu, 1, f.errorCPU
}
func testResourceSampler(f *samplerReader) *resourceSampler {
	p, _ := couchcore.ParseResourcePolicy(nil)
	p.OrphanRSSBytes = 100
	return &resourceSampler{reader: f, policy: p, now: func() time.Time { return time.Unix(100, 0) }, slots: func() []couchcore.ResourceSlot {
		return []couchcore.ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Roots: []couchcore.ResourceRoot{{Key: couchcore.ResourceProcessKey{PID: 10, Identity: "owned"}}}}}
	}}
}
func TestResourceSamplerSelectionCallsAndImmutableView(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}, {PID: 11, PPID: 10, Identity: "child"}, {PID: 12, PPID: 1, Identity: "vim", Name: "nvim"}, {PID: 13, PPID: 1, Identity: "other", Name: "unrelated"}}}
	s := testResourceSampler(f)
	s.persist = func(_ context.Context, state couchcore.ResourceState, v couchcore.ResourceView) (couchcore.ResourceState, couchcore.ResourceCost, error) {
		f.cpu += 500
		return state, couchcore.ResourceCost{StorageWrites: 1, StorageBytes: 200}, nil
	}
	s.pass(context.Background())
	v := s.view()
	if !reflect.DeepEqual(f.usages, []int{10, 11, 12}) || !reflect.DeepEqual(f.commands, []int{12}) {
		t.Fatal(f.usages, f.commands)
	}
	if v.Cost.NativeCalls != 12 || v.Cost.UsageCalls != 6 || v.Cost.CommandCalls != 2 || v.Cost.TableCalls != 2 || v.Cost.CPUTimeNS != 500 || !v.Cost.CPUKnown || v.Cost.StorageWrites != 1 {
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
	if len(v.Processes) != 1 || len(v.Unattributed) != couchcore.ResourceMaxProcesses-1 || !strings.Contains(strings.Join(v.Errors, " "), "limit exceeded") {
		t.Fatal("missing partial coverage", len(v.Processes), len(v.Unattributed), v.Errors)
	}
	if v.Cost.NativeCalls > 8+2*len(f.usages)+v.Cost.CommandCalls {
		t.Fatal("native call envelope", v.Cost)
	}
}
func TestResourceSamplerPersistenceFailureKeepsAdmittedState(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	s := testResourceSampler(f)
	s.persist = func(_ context.Context, state couchcore.ResourceState, v couchcore.ResourceView) (couchcore.ResourceState, couchcore.ResourceCost, error) {
		state.Episodes = []couchcore.ResourceEpisode{{Key: "admitted", Phase: "notified", Effect: &couchcore.ResourceEffect{ID: "stable"}}}
		return state, couchcore.ResourceCost{StorageWrites: 1}, errors.New("checkpoint failed")
	}
	s.pass(context.Background())
	v := s.view()
	if len(s.state.Episodes) != 1 || s.state.Episodes[0].Phase != "notified" || !strings.Contains(strings.Join(v.Errors, " "), "checkpoint failed") {
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

// TestResourceSamplerNativeCost measures the complete pass, including reducer,
// serialization, validation and snapshot publication at nearly full history.
// Native collection follows the real bounded selector; only the test root is
// owned, and other rows are eligible only for the ordinary PPID1 candidate path.
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
	persistence := newResourcePersistence(dir, resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return couchmessage.Binding{}, false }})
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
	if err = persistence.store.SaveHistory(raw); err != nil {
		t.Fatal(err)
	}
	persistence.history = seed
	if len(raw) < couchcore.ResourceHistoryLimit-65536 {
		t.Fatalf("fixture did not approach history cap: %d", len(raw))
	}
	policy, _ := couchcore.ParseResourcePolicy(nil)
	sampler := &resourceSampler{reader: reader, policy: policy, persist: persistence.persist, slots: func() []couchcore.ResourceSlot {
		return []couchcore.ResourceSlot{{Slot: address, Workspace: workspace, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{{Key: couchcore.ResourceProcessKey{PID: owned.PID, Identity: owned.Identity}}}}}
	}}
	var sum, maxCPU uint64
	for pass := 0; pass < 3; pass++ {
		sampler.now = func() time.Time { return base.Add(time.Duration(pass) * time.Minute) }
		sampler.pass(ctx)
		v := sampler.view()
		if !v.Cost.CPUKnown || v.Cost.StorageWrites != 2 || len(v.Processes) == 0 {
			t.Fatalf("incomplete measurement: %+v errors=%v", v.Cost, v.Errors)
		}
		for _, e := range v.Errors {
			if strings.Contains(e, "persistence") {
				t.Fatal(e)
			}
		}
		sum += v.Cost.CPUTimeNS
		if v.Cost.CPUTimeNS > maxCPU {
			maxCPU = v.Cost.CPUTimeNS
		}
		info, err := os.Stat(filepath.Join(dir, "resources", "history.json"))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("pass=%d table_rows=%d slots=%d owned_processes=%d unattributed=%d calls=%d table_calls=%d usage_calls=%d command_calls=%d CPU_ms=%.3f wall_ms=%.3f storage_bytes=%d storage_writes=%d history_bytes=%d", pass+1, len(rows), len(v.Slots), len(v.Processes), len(v.Unattributed), v.Cost.NativeCalls, v.Cost.TableCalls, v.Cost.UsageCalls, v.Cost.CommandCalls, float64(v.Cost.CPUTimeNS)/1e6, float64(v.Cost.WallTime)/1e6, v.Cost.StorageBytes, v.Cost.StorageWrites, info.Size())
	}
	mean := float64(sum) / 3
	t.Logf("CPU_mean_ms=%.3f CPU_max_ms=%.3f sixty_second_CPU_pct=%.6f seeded_frames=%d seeded_history_bytes=%d", mean/1e6, float64(maxCPU)/1e6, mean/float64(time.Minute)*100, len(seed.Frames), len(raw))
	// Revised measured budget: below 0.2% of one core at the default minute cadence.
	if mean >= float64(120*time.Millisecond) || maxCPU >= uint64(300*time.Millisecond) {
		t.Errorf("resource CPU budget exceeded: mean=%.3fms max=%.3fms", mean/1e6, float64(maxCPU)/1e6)
	}
}
