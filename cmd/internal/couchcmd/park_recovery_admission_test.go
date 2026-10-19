package couchcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// parkAdmissionCatalog retains a real stateful catalog. The barrier delays the
// external workspace probe, never the socket or the queue implementation.
type parkAdmissionCatalog struct {
	*couchcore.SlotCatalogFake
	discoveries atomic.Int32
	mu          sync.Mutex
	entered     chan struct{}
	release     chan struct{}
	once        sync.Once
	probes      atomic.Int32
}

func (f *parkAdmissionCatalog) ResolveWorkspace(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
	f.probes.Add(1)
	f.once.Do(func() { close(f.entered) })
	select {
	case <-f.release:
	case <-ctx.Done():
		return couchcore.WorkspaceIdentity{}, ctx.Err()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.SlotCatalogFake.ResolveWorkspace(ctx, path)
}
func (f *parkAdmissionCatalog) Discover(ctx context.Context, path string) (couchcore.SlotRepository, error) {
	f.discoveries.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.SlotCatalogFake.Discover(ctx, path)
}

type parkAdmissionRig struct {
	input           io.Writer
	child           *ptychild.Child
	done            <-chan struct{}
	service         *serviceRig
	caller          couchmessage.Binding
	console         *couchtty.Console
	couch           *couchcore.Couch
	before          couchcore.ThreadRecord
	catalog         *parkAdmissionCatalog
	effects         atomic.Int32
	callerProbes    atomic.Int32
	dispatched      chan struct{}
	dispatchRelease chan struct{}
	releaseOnce     sync.Once
}

func newParkAdmissionRig(t *testing.T, holdDispatch bool, rowCounts ...int) *parkAdmissionRig {
	return newParkAdmissionRigConfigured(t, holdDispatch, nil, rowCounts...)
}
func newParkAdmissionRigConfigured(t *testing.T, holdDispatch bool, configure func(*couchcore.Couch), rowCounts ...int) *parkAdmissionRig {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(root, "repo")
	if err := os.MkdirAll(primary, 0700); err != nil {
		t.Fatal(err)
	}
	rt := newRT(t, primary)
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	if configure != nil {
		configure(c)
	}
	scope, err := launcher.ResolveRepoScope(primary)
	if err != nil {
		t.Fatal(err)
	}
	profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	record, err := c.Threads.CreateThread(couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"}, StartingPath: primary, WorkingPath: primary, CreatedAt: time.Unix(1, 0).UTC(), Revision: 1, LatestLaunchProfile: &profile, Incarnations: []couchcore.ThreadIncarnation{{PID: 42, Identity: "park-owner", State: couchcore.IncarnationLive, RepoIdentity: filepath.Join(primary, ".git"), LaunchProfile: &profile}}})
	if err != nil {
		t.Fatal(err)
	}
	park := couchcore.ParkIdentity{Nonce: "park-admission-original", Address: record.Address, PID: 42, ProcessIdentity: "park-owner"}
	record, err = c.Threads.BeginPark(record.Address, record.Revision, park)
	if err != nil {
		t.Fatal(err)
	}
	record, err = c.Threads.AdvancePark(context.Background(), record.Address, record.Revision, couchcore.ParkEvent{Kind: couchcore.ParkRequestCommitted, Identity: park, Attempt: 1})
	if err != nil {
		t.Fatal(err)
	}
	rt.artifacts.SetPairSession(record.Address, "pair-exact", false)
	rt.artifacts.SetSessionPresence(record.Address, couchcore.SessionObservation{State: couchcore.SessionAbsent})
	address, slot, rest := "repo:0", 0, "main"
	identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: root, EnvironmentRoot: root, WorktreeRoot: primary, Kind: "primary", Address: &address, Slot: &slot, RestingBranch: &rest}
	catalog := &parkAdmissionCatalog{SlotCatalogFake: &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{".": identity, primary: identity}, Repositories: map[string]couchcore.SlotRepository{primary: {Identity: identity}}}, entered: make(chan struct{}), release: make(chan struct{})}
	c.Slots = catalog
	c.PairLifecycle = &couchcore.PairLifecycleController{Threads: c.Threads, DataDir: root, Lifecycle: couchcore.PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}, Sessions: rt.artifacts, Proc: rt.proc, Clock: c.Clock, Nonce: func() (string, error) { return "unused", nil }}
	if err := c.Threads.EnrollSlotRepository(context.Background(), couchcore.SlotRepository{Identity: identity}); err != nil {
		t.Fatal(err)
	}
	rows, err := c.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog.discoveries.Store(0)
	if len(rowCounts) > 0 {
		for len(rows) < rowCounts[0] {
			rows = append(rows, couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag("unrelated-" + strings.Repeat("x", len(rows)))}, StartingPath: filepath.Join(root, "other")})
		}
	}
	reader, input := io.Pipe()
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), reader)
	initial := ptychild.NewFakeChild(nil)
	console.Attach("initial-console-pane", "initial", initial)
	state, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{{Root: primary, Workspace: root, Slot: couchcore.HubSlot{Repository: primary, Number: 0}}})
	if err != nil {
		t.Fatal(err)
	}
	hubs := &hubRuntime{state: state}
	wireResolver(console, c, hubs)
	// Publish the retained observation through the same provider/run path as the
	// switcher; future current-world reads still use Couch's real inventory.
	console.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return rows, nil
	})
	rig := &parkAdmissionRig{input: input, child: initial, console: console, couch: c, before: record, catalog: catalog, dispatched: make(chan struct{}), dispatchRelease: make(chan struct{})}
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == couchcore.OpRecoverPark {
			close(rig.dispatched)
			if holdDispatch {
				select {
				case <-rig.dispatchRelease:
				case <-call.Context.Done():
					return nil, call.Context.Err()
				}
			}
		}
		value, err := actual(call)
		if call.Name == couchcore.OpRecoverPark && err == nil {
			rig.effects.Add(1)
		}
		return value, err
	})
	done := make(chan struct{})
	rig.done = done
	go func() { console.Run(); close(done) }()
	t.Cleanup(func() {
		rig.release()
		rig.releaseOnce.Do(func() { close(rig.dispatchRelease) })
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		initial.Exit(0)
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("console did not stop")
		}
	})
	testwait.Until(t, "cached interrupted-park inventory", func() bool {
		s := console.ResourceInventory()
		return s.InventoryKnown && len(s.Rows) == len(rows) && s.Rows[0].ParkNonce == park.Nonce
	})
	if catalog.probes.Load() != 0 {
		t.Fatal("cache publication invoked workspace resolution")
	}
	// Admit the fixture hub before installing the slow external-workspace
	// barrier. All subsequent service requests use production hub ownership and
	// target resolution, including the real workspace resolver for the caller.
	var registered atomic.Bool
	rig.service = &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}, slotOps: consoleSlotOperations(console, c, nil)}
	rig.service.decorateAuthority = func(a messageAuthority) messageAuthority {
		a.hub, a.hubs, a.hubTarget = primary, hubs, hubTargetResolver(c)
		a.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
			if !registered.Load() {
				return catalog.SlotCatalogFake.ResolveWorkspace(ctx, path)
			}
			rig.callerProbes.Add(1)
			return catalog.ResolveWorkspace(ctx, path)
		}
		return a
	}
	rig.service.init()
	rig.caller = rig.service.world.add(0)
	rig.caller.Slot, rig.caller.Repository = address, identity.RepoIdentity
	rig.service.world.set(rig.caller, func(s *worldSlot) {
		s.binding, s.root, s.workspace, s.branch = rig.caller, primary, identity, rest
	})
	rig.service.attach(rig.caller, couchmessage.PaneHandle("fixture-hub"))
	rig.service.wrapper(rig.caller)
	rig.service.waitConnected(rig.caller, true)
	registered.Store(true)
	return rig
}
func (r *parkAdmissionRig) release() {
	select {
	case <-r.catalog.release:
	default:
		close(r.catalog.release)
	}
}
func (r *parkAdmissionRig) request(id string) couchmessage.Request {
	req := slotRequest(r.caller, "recover-park", id, "repo:0")
	req.Confirmed = true
	return req
}
func (r *parkAdmissionRig) call(t *testing.T, req couchmessage.Request) couchmessage.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var response couchmessage.Response
	if err := couchmessage.Call(ctx, r.service.brokerSock, req, &response); err != nil {
		t.Fatalf("socket %s: %v", req.Op, err)
	}
	return response
}
func (r *parkAdmissionRig) terminal(t *testing.T, id string) couchmessage.OperationReceipt {
	t.Helper()
	var got couchmessage.Response
	testwait.Until(t, "terminal recover-park receipt", func() bool {
		got = r.call(t, statusRequest(r.caller, id))
		return got.Operation != nil && (got.Operation.Status == couchmessage.ReceiptSucceeded || got.Operation.Status == couchmessage.ReceiptFailed || got.Operation.Status == couchmessage.ReceiptRefused)
	})
	return *got.Operation
}
func awaitParkAdmission(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-testwait.After(t):
		t.Fatalf("waiting for %s", what)
	}
}

func TestRecoverParkAdmissionPrecedesSlowQueuedPreparation(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	request := r.request("slow-park")
	accepted := r.call(t, request)
	if accepted.Code != "accepted" || accepted.Operation == nil {
		t.Fatalf("admission: %+v", accepted)
	}
	awaitParkAdmission(t, r.catalog.entered, "queued full preparation")
	blocked := time.Now()
	// This interval is a protocol deadline assertion, not a readiness guess.
	<-time.After(time.Until(blocked.Add(couchmessage.AdmissionTimeout + 50*time.Millisecond)))
	status := r.call(t, statusRequest(r.caller, request.ID))
	if status.Operation == nil || status.Operation.Status != couchmessage.ReceiptRunning || r.effects.Load() != 0 {
		t.Fatalf("blocked status: %+v effects=%d", status, r.effects.Load())
	}
	probes, callerProbes := r.catalog.probes.Load(), r.callerProbes.Load()
	duplicate := r.call(t, request)
	if duplicate.Code != "accepted" || duplicate.Operation == nil || duplicate.Operation.ID != accepted.Operation.ID || r.catalog.probes.Load() != probes || r.callerProbes.Load() != callerProbes {
		t.Fatalf("duplicate: %+v probes=%d", duplicate, r.catalog.probes.Load())
	}
	r.release()
	terminal := r.terminal(t, request.ID)
	if terminal.Status != couchmessage.ReceiptSucceeded || terminal.Code != "park-recovered" || r.effects.Load() != 1 {
		t.Fatalf("terminal: %+v effects=%d", terminal, r.effects.Load())
	}
	after, err := r.couch.Threads.GetThread(r.before.Address)
	if err != nil || after.Park != nil || len(after.Incarnations) != 0 || len(after.ParkHistory) != 1 {
		t.Fatalf("recovered record: %+v %v", after, err)
	}
	revision := after.Revision
	if again := r.call(t, request); again.Operation == nil || again.Operation.Status != couchmessage.ReceiptSucceeded {
		t.Fatalf("repeat: %+v", again)
	}
	after, err = r.couch.Threads.GetThread(r.before.Address)
	if err != nil || after.Revision != revision || r.effects.Load() != 1 {
		t.Fatalf("duplicate effect: %+v %v", after, err)
	}
}

func TestRecoverParkQueuedRejectsNewNonce(t *testing.T) {
	for _, atDispatch := range []bool{false, true} {
		name := "before-prepare"
		if atDispatch {
			name = "before-dispatch"
		}
		t.Run(name, func(t *testing.T) {
			r := newParkAdmissionRig(t, atDispatch)
			req := r.request("changed-park")
			if got := r.call(t, req); got.Code != "accepted" {
				t.Fatalf("admission: %+v", got)
			}
			awaitParkAdmission(t, r.catalog.entered, "queued preparation")
			if atDispatch {
				r.release()
				awaitParkAdmission(t, r.dispatched, "prepared dispatch")
			}
			current, err := r.couch.Threads.AbandonPark(r.before.Address, r.before.Revision, r.before.Park.Identity)
			if err != nil {
				t.Fatal(err)
			}
			replacement := r.before.Park.Identity
			replacement.Nonce = "park-admission-replacement"
			current, err = r.couch.Threads.BeginPark(current.Address, current.Revision, replacement)
			if err != nil {
				t.Fatal(err)
			}
			r.release()
			r.releaseOnce.Do(func() { close(r.dispatchRelease) })
			got := r.terminal(t, req.ID)
			if got.Status == couchmessage.ReceiptSucceeded || !strings.Contains(got.Detail, "target changed") || r.effects.Load() != 0 {
				t.Fatalf("stale target: %+v effects=%d", got, r.effects.Load())
			}
			after, err := r.couch.Threads.GetThread(current.Address)
			if err != nil || after.Revision != current.Revision || after.Park.Identity.Nonce != replacement.Nonce {
				t.Fatalf("replacement changed: %+v %v", after, err)
			}
			// Completion releases the canonical pending key, even after refusal.
			if retry := r.call(t, r.request("retry-changed")); retry.Code != "accepted" {
				t.Fatalf("pending key retained: %+v", retry)
			}
			_ = r.terminal(t, "retry-changed")
		})
	}
}

func TestRecoverParkAdmissionUsesConstantFullPasses(t *testing.T) {
	var firstCounts [3]int32
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			r := newParkAdmissionRig(t, false, n)
			if got := r.call(t, r.request("inventory-size")); got.Code != "accepted" {
				t.Fatalf("admission: %+v", got)
			}
			awaitParkAdmission(t, r.catalog.entered, "queued preparation")
			if r.catalog.discoveries.Load() != 0 || r.effects.Load() != 0 {
				t.Fatalf("blocked probe escaped: discoveries=%d effects=%d", r.catalog.discoveries.Load(), r.effects.Load())
			}
			r.release()
			got := r.terminal(t, "inventory-size")
			counts := [3]int32{r.catalog.probes.Load(), r.catalog.discoveries.Load(), r.callerProbes.Load()}
			if n == 8 {
				firstCounts = counts
			}
			if got.Status != couchmessage.ReceiptSucceeded || r.effects.Load() != 1 || counts != firstCounts || counts[0] == 0 || counts[1] == 0 {
				t.Fatalf("%d cached rows: receipt=%+v effects=%d probes=%d discoveries=%d", n, got, r.effects.Load(), r.catalog.probes.Load(), r.catalog.discoveries.Load())
			}
		})
	}
}

func TestRecoverParkQueuedCancellationHasNoLateEffect(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	if got := r.call(t, r.request("cancel-park")); got.Code != "accepted" {
		t.Fatalf("admission: %+v", got)
	}
	awaitParkAdmission(t, r.catalog.entered, "queued preparation")
	r.console.Stop()
	awaitParkAdmission(t, r.done, "console joining canceled preparation")
	r.release()
	if r.effects.Load() != 0 {
		t.Fatalf("canceled recovery effects=%d", r.effects.Load())
	}
	after, err := r.couch.Threads.GetThread(r.before.Address)
	if err != nil || after.Revision != r.before.Revision || after.Park == nil || after.Park.Identity != r.before.Park.Identity {
		t.Fatalf("canceled operation changed park: %+v %v", after, err)
	}
	select {
	case <-r.dispatched:
		t.Fatal("canceled preparation reached dispatch")
	default:
	}
}

func TestRecoverParkQueuedPreparationCancellationReleasesPendingKey(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	if got := r.call(t, r.request("prepare-canceled")); got.Code != "accepted" {
		t.Fatalf("admission: %+v", got)
	}
	awaitParkAdmission(t, r.catalog.entered, "queued preparation")
	// A canceled external observation is an operation failure. It must finish
	// normally while the console and receipt service remain available.
	r.catalog.mu.Lock()
	r.catalog.Errors = map[string]error{".": context.Canceled, r.before.StartingPath: context.Canceled}
	r.catalog.mu.Unlock()
	r.release()
	got := r.terminal(t, "prepare-canceled")
	if got.Status != couchmessage.ReceiptFailed && got.Status != couchmessage.ReceiptRefused {
		t.Fatalf("canceled preparation: %+v", got)
	}
	if !strings.Contains(got.Detail, "context canceled") || r.effects.Load() != 0 {
		t.Fatalf("canceled preparation: %+v effects=%d", got, r.effects.Load())
	}
	after, err := r.couch.Threads.GetThread(r.before.Address)
	if err != nil || after.Revision != r.before.Revision || after.Park == nil || after.Park.Identity != r.before.Park.Identity {
		t.Fatalf("canceled preparation changed park: %+v %v", after, err)
	}
	r.catalog.mu.Lock()
	r.catalog.Errors = nil
	r.catalog.mu.Unlock()
	if retry := r.call(t, r.request("after-prepare-canceled")); retry.Code != "accepted" {
		t.Fatalf("pending key retained: %+v", retry)
	}
	if retry := r.terminal(t, "after-prepare-canceled"); retry.Status != couchmessage.ReceiptSucceeded || r.effects.Load() != 1 {
		t.Fatalf("retry: %+v effects=%d", retry, r.effects.Load())
	}
}

func TestRecoverParkAdmissionCachedRefusals(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguous), func(t *testing.T) {
			r := newParkAdmissionRig(t, false)
			request := r.request("cached-refusal")
			if ambiguous {
				snapshot := r.console.ResourceInventory()
				rows := append(snapshot.Rows, snapshot.Rows[0])
				r.console.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
					return rows, nil
				})
				testwait.Until(t, "ambiguous cached rows published", func() bool { return len(r.console.ResourceInventory().Rows) == 2 })
			} else {
				request.Target = "repo:2"
			}
			got := r.call(t, request)
			if got.Code != "refused" || !strings.Contains(got.Error, "refresh") || !strings.Contains(got.Error, "retry") || got.Operation != nil || r.catalog.probes.Load() != 0 || r.effects.Load() != 0 {
				t.Fatalf("cached refusal %+v probes=%d effects=%d", got, r.catalog.probes.Load(), r.effects.Load())
			}
		})
	}
}
