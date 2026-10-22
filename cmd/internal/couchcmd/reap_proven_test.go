package couchcmd

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestAuthorizeProvenOrphanRequiresOwningHub(t *testing.T) {
	state := guardHubState(t).snapshot()
	target := couchcore.HubSlot{Repository: "/repo", Number: 2}
	for _, tc := range []struct {
		root  string
		allow bool
	}{
		{"/repo1", true}, {"/repo0", false}, {"/repo2", false}, {"", false},
	} {
		err := authorizeProvenOrphan(couchcore.WorkspaceIdentity{WorktreeRoot: tc.root}, target, state)
		if (err == nil) != tc.allow {
			t.Fatalf("root %q allowed=%v: %v", tc.root, tc.allow, err)
		}
	}
}

func TestProvenOrphanSignalRejectsInvalidAndUnavailableProof(t *testing.T) {
	s := &messageService{}
	for _, request := range []couchmessage.Request{
		{Op: "reap-orphan-signal"},
		{Op: "reap-orphan-signal", Confirmed: true, OrphanTarget: effectTarget(), OrphanSignal: "TERM"},
	} {
		got := s.handleProvenOrphanSignal(context.Background(), request)
		if got.OrphanSignal != nil && got.OrphanSignal.Status == "sent" || got.Code == "ok" && got.OrphanSignal == nil {
			t.Fatalf("unproved request: %+v", got)
		}
	}
}

func effectTarget() *couchcore.ProvenOrphanTarget {
	n, address, rest := 0, "repo:0", "main"
	return &couchcore.ProvenOrphanTarget{
		Key:  couchcore.ResourceProcessKey{PID: 11, Identity: "old-child"},
		Root: couchcore.ResourceProcessKey{PID: 10, Identity: "old-root"},
		Slot: address, Thread: couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "1-repo-0"},
		Workspace: couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: "/repo/.git", PrimaryRoot: "/repo", FleetRoot: "/", EnvironmentRoot: "/", WorktreeRoot: "/repo", Kind: "primary", Slot: &n, Address: &address, RestingBranch: &rest},
	}
}

// This world keeps the old child's exact provenance while a new zellij root
// owns the same thread. The worker, rather than the cached view, proves effects.
func provenEffectService(t *testing.T) (*messageService, *samplerReader, *effectProcessTable) {
	t.Helper()
	target := effectTarget()
	f := &samplerReader{rows: []procutil.Process{{PID: 10, PPID: 1, Identity: "old-root", Name: "zellij"}, {PID: 11, PPID: 10, Identity: "old-child", Name: "codex"}, {PID: 20, PPID: 1, Identity: "new-root", Name: "zellij"}}}
	s := testResourceSampler(f)
	old := couchcore.ResourceRoot{Key: target.Root, Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace}
	first := couchcore.SelectResourceProcesses(f.rows[:2], []couchcore.ResourceSlot{{Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{old}}}, nil, true, true)
	s.state.Provenance = first.Processes
	current := old
	current.Key = couchcore.ResourceProcessKey{PID: 20, Identity: "new-root"}
	s.slots = func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{current}}}}
	}
	s.sweeps = make(chan resourceRequest)
	s.policy.Cadence = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	waitFor(t, func() bool { return len(s.view().Processes) == 3 })
	table := &effectProcessTable{identity: "old-child"}
	service := &messageService{provenOrphanProcesses: table}
	service.resources.Store(s)
	return service, f, table
}

type effectProcessTable struct {
	identity       string
	signalErr      error
	signals        []syscall.Signal
	beforeIdentity func()
	beforeSignal   func()
}

func (p *effectProcessTable) Snapshot(context.Context) ([]launcher.ProcessRow, error) {
	panic("effect expanded process table")
}
func (p *effectProcessTable) Command(int) ([]string, bool) { panic("effect read command") }
func (p *effectProcessTable) Identity(int) string {
	if p.beforeIdentity != nil {
		p.beforeIdentity()
	}
	return p.identity
}
func (p *effectProcessTable) Signal(pid int, sig syscall.Signal) error {
	if pid != 11 {
		panic("foreign process signalled")
	}
	if p.beforeSignal != nil {
		p.beforeSignal()
	}
	p.signals = append(p.signals, sig)
	return p.signalErr
}
func effectRequest() couchmessage.Request {
	return couchmessage.Request{Op: "reap-orphan-signal", Confirmed: true, OrphanTarget: effectTarget(), OrphanSignal: "TERM"}
}

func TestProvenOrphanSignalUsesFreshWorkerAndFinalGeneration(t *testing.T) {
	for _, tc := range []struct {
		name, identity, want string
		signalErr            error
		cancel               bool
		gone                 bool
	}{
		{name: "exact TERM", identity: "old-child", want: "sent"},
		{name: "unknown final identity", want: "refused"},
		{name: "reused final PID", identity: "replacement", want: "skipped"},
		{name: "signal failure", identity: "old-child", want: "failed", signalErr: errors.New("denied")},
		{name: "cancel at final gate", identity: "old-child", want: "failed", cancel: true},
		{name: "fresh absence", identity: "old-child", want: "skipped", gone: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, reader, table := provenEffectService(t)
			table.identity, table.signalErr = tc.identity, tc.signalErr
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				table.beforeIdentity = cancel
			}
			if tc.gone {
				reader.setRows([]procutil.Process{{PID: 20, PPID: 1, Identity: "new-root", Name: "zellij"}})
			}
			got := s.handleProvenOrphanSignal(ctx, effectRequest())
			if got.OrphanSignal == nil || got.OrphanSignal.Status != tc.want {
				t.Fatalf("response %+v", got)
			}
			wantSignals := 0
			if tc.want == "sent" || tc.signalErr != nil {
				wantSignals = 1
			}
			if len(table.signals) != wantSignals {
				t.Fatalf("signals %v", table.signals)
			}
		})
	}
}

func TestProvenOrphanSignalHoldsHubOwnershipThroughSingleEffect(t *testing.T) {
	s, _, table := provenEffectService(t)
	s.authority.hubs = guardHubState(t)
	checked := false
	table.beforeSignal = func() {
		checked = true
		if s.authority.hubs.mu.TryLock() {
			s.authority.hubs.mu.Unlock()
			t.Fatal("ownership move could commit during signal")
		}
	}
	got := s.handleProvenOrphanSignal(context.Background(), effectRequest())
	if got.OrphanSignal == nil || got.OrphanSignal.Status != "sent" || !checked {
		t.Fatalf("response %+v checked=%v", got, checked)
	}
	if !s.authority.hubs.mu.TryLock() {
		t.Fatal("hub lock leaked after signal")
	}
	s.authority.hubs.mu.Unlock()
}

func TestProvenOrphanSignalAuthenticatesCurrentOwningHub(t *testing.T) {
	r, bindings := hubMessageRig(t)
	resources, reader, table := provenEffectService(t)
	r.s.resources.Store(resources.resources.Load())
	r.s.provenOrphanProcesses = table
	for i, b := range bindings {
		req := effectRequest()
		req.Scope, req.Tag, req.Session, req.Nonce = b.Scope, b.Tag, b.Session, b.Nonce
		got := r.s.handleProvenOrphanSignal(context.Background(), req)
		want := "refused"
		if i == 0 {
			want = "sent"
		}
		if got.OrphanSignal == nil || got.OrphanSignal.Status != want {
			t.Fatalf("caller %d response %+v", i, got)
		}
	}
	before := len(table.signals)
	// A valid connection whose recorded incarnation changes while the worker is
	// proving ownership cannot retain admission's former authority.
	b := bindings[0]
	req := effectRequest()
	req.Scope, req.Tag, req.Session, req.Nonce = b.Scope, b.Tag, b.Session, b.Nonce
	reader.mu.Lock()
	reader.tableHook = func(context.Context) { r.world.set(b, func(w *worldSlot) { w.recorded = false }) }
	reader.mu.Unlock()
	got := r.s.handleProvenOrphanSignal(context.Background(), req)
	if got.OrphanSignal == nil || got.OrphanSignal.Status != "refused" || len(table.signals) != before {
		t.Fatalf("stale caller response %+v signals %v", got, table.signals)
	}
	req = effectRequest()
	req.Scope = "scope"
	got = r.s.handleProvenOrphanSignal(context.Background(), req)
	if got.Code != "invalid-request" || len(table.signals) != before {
		t.Fatalf("partial credentials response %+v", got)
	}
}

func TestProvenOrphanSignalRepeatsOwnershipVetoBeforeKill(t *testing.T) {
	for _, kind := range []string{"current", "detached-unknown", "incomplete"} {
		t.Run(kind, func(t *testing.T) {
			s, reader, table := provenEffectService(t)
			request := effectRequest()
			first := s.handleProvenOrphanSignal(context.Background(), request)
			if first.OrphanSignal == nil || first.OrphanSignal.Status != "sent" {
				t.Fatalf("TERM %+v", first)
			}
			rows := []procutil.Process{{PID: 10, PPID: 1, Identity: "old-root", Name: "zellij"}, {PID: 11, PPID: 20, Identity: "old-child", Name: "codex"}, {PID: 20, PPID: 1, Identity: "new-root", Name: "zellij"}}
			if kind == "detached-unknown" {
				rows[1].PPID = 1
				rows[1].Name = "sdlc"
			}
			reader.setRows(rows)
			if kind == "incomplete" {
				reader.mu.Lock()
				reader.tableErr = errors.New("partial census")
				reader.mu.Unlock()
			}
			request.OrphanSignal = "KILL"
			second := s.handleProvenOrphanSignal(context.Background(), request)
			if second.OrphanSignal == nil || second.OrphanSignal.Status != "refused" || len(table.signals) != 1 {
				t.Fatalf("KILL %+v signals %v", second, table.signals)
			}
		})
	}
}
