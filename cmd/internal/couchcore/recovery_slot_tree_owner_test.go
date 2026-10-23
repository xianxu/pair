package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// slotTreeOwnerWorld couples actual pane-command visibility to the very process
// retirement stops. The held pane and its exact server survive process exit.
// Replies use the shared native serializer and the production strict decoder.
type slotTreeOwnerWorld struct {
	proc           *FakeProcOps
	server         launcher.SessionServerIdentity
	live, stopped  []byte
	events         []string
	replaceOnPanes bool
	absent         bool
	launched       map[string][]byte
}

func (w *slotTreeOwnerWorld) SessionServers(_ context.Context, name string) ([]launcher.SessionServerIdentity, error) {
	if _, ok := w.launched[name]; ok {
		return []launcher.SessionServerIdentity{{PID: 798, Identity: "replacement-server", Session: name, Socket: "/fixture/798/socket"}}, nil
	}
	if w.absent {
		return nil, nil
	}
	return []launcher.SessionServerIdentity{w.server}, nil
}
func (w *slotTreeOwnerWorld) SessionPresent(context.Context, string) (bool, error) {
	return !w.absent, nil
}
func (*slotTreeOwnerWorld) Socket(string) launcher.SocketState { return launcher.SocketPresent }
func (w *slotTreeOwnerWorld) SessionPanes(_ context.Context, name string) ([]byte, error) {
	if raw, ok := w.launched[name]; ok {
		w.events = append(w.events, "panes:replacement")
		return raw, nil
	}
	if w.replaceOnPanes {
		w.server.Identity = "replacement"
	}
	if w.proc.Exists(501) == Live {
		w.events = append(w.events, "panes:live")
		return w.live, nil
	}
	w.events = append(w.events, "panes:stopped")
	return w.stopped, nil
}
func newSlotTreeOwnerWorld(t *testing.T, proc *FakeProcOps, root string, address ThreadAddress, name string) *slotTreeOwnerWorld {
	t.Helper()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	command := fmt.Sprintf("pair wrap --scrollback-log %q codex", paths.ScrollbackRaw("codex"))
	reply := func(actual any) []byte {
		return []byte(zellijtest.Reply(t, "panes", []map[string]any{{"id": 1, "pane_command": actual, "terminal_command": command}}).Stdout)
	}
	return &slotTreeOwnerWorld{proc: proc,
		server: launcher.SessionServerIdentity{PID: 797, Identity: "server-original", Session: name, Socket: "/fixture/797/socket"},
		live:   reply(command), stopped: reply("")}
}

// slotTreeOwnedDeleter accepts only the generation positively observed by the
// real owner probe. A name-only delete cannot silently satisfy this fixture.
type slotTreeOwnedDeleter struct {
	world    *slotTreeOwnerWorld
	expected launcher.SessionServerIdentity
	deleted  []launcher.SessionOwnerObservation
	err      error
}

func (*slotTreeOwnedDeleter) DeleteSession(string) error {
	return errors.New("name-only deletion forbidden")
}
func (d *slotTreeOwnedDeleter) DeleteOwnedSessionContext(_ context.Context, owner launcher.SessionOwnerObservation) error {
	if owner.State != launcher.SessionOwnerOwned || owner.Server != d.expected || owner.Server != d.world.server || owner.Name != d.expected.Session {
		return errors.New("owner generation does not match deletion target")
	}
	d.world.events = append(d.world.events, "quiesce:owned")
	if d.err != nil {
		return d.err
	}
	d.deleted = append(d.deleted, owner)
	d.world.absent = true
	return nil
}
func slotTreeOwnerFixture(t *testing.T) (*recoveryExecutionFixture, *slotTreeOwnerWorld, *slotTreeOwnedDeleter) {
	t.Helper()
	f := slotTreeFixture(t)
	root := t.TempDir()
	name := recordSessionName(f.record)
	world := newSlotTreeOwnerWorld(t, f.Proc, root, f.record.Address, name)
	checker := NewScopedThreadArtifactCollisionChecker(root)
	deleter := &slotTreeOwnedDeleter{world: world, expected: world.server}
	checker.Sessions = deleter
	checker.OwnerProbe = launcher.SessionOwnerProbe{IO: world}
	checker.Servers = &ownerPassSource{states: map[string]launcher.ServerState{name: {Verdict: launcher.ServerReachable, Server: world.server}}}
	f.Couch.Artifacts = checker
	acknowledge := f.Runner.AfterAcknowledge
	f.Runner.AfterAcknowledge = func(id string) error {
		if err := acknowledge(id); err != nil {
			return err
		}
		current, err := f.Couch.Threads.observeSlotCurrent()
		if err != nil || current.Record == nil {
			return fmt.Errorf("observe replacement registration: %w", err)
		}
		address := current.Record.Address
		name := continuationChildSession(t, f.Runner, id)
		indexSession(t, root, address, name)
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: address.RepoScope, Tag: string(address.Tag)})
		if err != nil {
			return err
		}
		world.launched = map[string][]byte{name: []byte(zellijtest.Reply(t, "panes", []map[string]any{{"id": 1, "pane_command": fmt.Sprintf("pair wrap --scrollback-log %q codex", paths.ScrollbackRaw("codex"))}}).Stdout)}
		return nil
	}
	// Recovery may conservatively retain unproved source remnants; provide the
	// unchanged verified checkout that its ordinary readiness guard requires.
	slot := *f.Couch.Threads.slot
	f.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{slot.PrimaryRoot: {
		Identity: WorkspaceIdentity{RepoIdentity: slot.RepoIdentity, PrimaryRoot: slot.PrimaryRoot},
		Slots:    []SlotCandidate{{Identity: slot, Verified: true}},
	}}}
	f.Proc.OnSignal = func(pid int, _ os.Signal) { world.events = append(world.events, fmt.Sprintf("stop:%d", pid)) }
	return f, world, deleter
}

func TestSlotTreeOwnerQuiescesBeforeStoppingTree(t *testing.T) {
	f, world, deleter := slotTreeOwnerFixture(t)
	result, stops, err := f.rebootFresh(t)
	if err != nil || result.Start.Handle == nil {
		t.Fatalf("recovery: result=%+v err=%v events=%v", result, err, world.events)
	}
	roles := []string{}
	for _, stop := range stops {
		roles = append(roles, stop.Role)
	}
	if !strings.Contains(strings.Join(roles, ","), slotSessionRole+world.server.Session) {
		t.Fatalf("confirmation omitted session: %v", roles)
	}
	want := []string{"panes:live", "panes:live", "quiesce:owned", "stop:500", "stop:501", "stop:502"}
	if len(world.events) < len(want) || !reflect.DeepEqual(world.events[:len(want)], want) {
		t.Fatalf("retirement order=%v want prefix=%v", world.events, want)
	}
	for _, pid := range []int{500, 501, 502} {
		if f.Proc.Exists(pid) != Dead {
			t.Fatalf("pid %d survived", pid)
		}
	}
	if f.Proc.Exists(600) != Live || len(f.Proc.Signals[600]) != 0 {
		t.Fatal("unrelated process touched")
	}
	if len(deleter.deleted) != 1 || deleter.deleted[0].Server != deleter.expected || len(f.Runner.order) != 1 {
		t.Fatalf("deleted=%v launches=%v", deleter.deleted, f.Runner.order)
	}
	t.Logf("exact server %+v; retirement order %v; replacement launches %d", deleter.expected, world.events, len(f.Runner.order))
}

func TestSlotTreeOwnerRefusalLeavesAddedTreeAndLaunchUntouched(t *testing.T) {
	for _, mode := range []string{"empty", "null", "ambiguous", "foreign", "replacement", "quiesce error", "original error"} {
		t.Run(mode, func(t *testing.T) {
			f, world, deleter := slotTreeOwnerFixture(t)
			switch mode {
			case "empty", "null", "ambiguous":
				var actual any = ""
				if mode == "null" {
					actual = nil
				}
				if mode == "ambiguous" {
					actual = "pair wrap '$UNRESOLVED"
				}
				world.live = []byte(zellijtest.Reply(t, "panes", []map[string]any{{"id": 1, "pane_command": actual}}).Stdout)
			case "foreign":
				address := f.record.Address
				address.Tag = "couch-2222222222222222"
				foreign := newSlotTreeOwnerWorld(t, f.Proc, f.Couch.Artifacts.(ScopedThreadArtifactCollisionChecker).GlobalDataDir, address, world.server.Session)
				world.live = foreign.live
			case "replacement":
				world.replaceOnPanes = true
			case "quiesce error":
				deleter.err = errors.New("session deletion failed")
			case "original error":
				f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
					return RecoveryRetirement{Digest: "original", Retire: func(context.Context) error {
						world.events = append(world.events, "original:error")
						return errors.New("original retirement failed")
					}}, nil
				}
			}
			_, _, err := f.rebootFresh(t)
			if err == nil {
				t.Fatalf("%s admitted recovery: events=%v", mode, world.events)
			}
			if mode == "quiesce error" && !strings.Contains(err.Error(), "session deletion failed") {
				t.Fatalf("did not exercise deletion failure: %v", err)
			}
			for _, pid := range []int{500, 501, 502} {
				if f.Proc.Exists(pid) != Live || len(f.Proc.Signals[pid]) != 0 {
					t.Fatalf("%s signalled pid %d before quiescence: events=%v err=%v", mode, pid, world.events, err)
				}
			}
			if len(deleter.deleted) != 0 || len(f.Runner.order) != 0 {
				t.Fatalf("%s caused effects: deleted=%v launches=%v", mode, deleter.deleted, f.Runner.order)
			}
			if mode == "original error" && !reflect.DeepEqual(world.events, []string{"original:error"}) {
				t.Fatalf("original failure leaked to quiescence: %v", world.events)
			}
			t.Logf("%s: %v; events=%v", mode, err, world.events)
		})
	}
}

func TestSlotTreeOwnerPreservesOriginalRetirementFirst(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprintf("original=%v", present), func(t *testing.T) {
			f, world, _ := slotTreeOwnerFixture(t)
			original := RecoveryRetirement{}
			if present {
				original.Retire = func(context.Context) error { world.events = append(world.events, "original:retire"); return nil }
			}
			retirement, _, err := f.Couch.withSlotTree(t.Context(), f.record, original)
			if err != nil {
				t.Fatal(err)
			}
			if err = retirement.Retire(t.Context()); err != nil {
				t.Fatal(err)
			}
			want := []string{"panes:live", "panes:live", "quiesce:owned", "stop:500", "stop:501", "stop:502"}
			if present {
				want = append([]string{"original:retire"}, want...)
			}
			if !reflect.DeepEqual(world.events, want) {
				t.Fatalf("events=%v want=%v", world.events, want)
			}
		})
	}
}

func TestSlotTreeOwnerProbeCharacterizationNativeCommandEvidence(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-1111111111111111"}
	root := t.TempDir()
	for _, tc := range []struct {
		name                 string
		actual               any
		foreign, replacement bool
		want                 launcher.SessionOwnerState
	}{
		{name: "valid", want: launcher.SessionOwnerOwned},
		{name: "empty actual command", actual: "", want: launcher.SessionOwnerUnknown},
		{name: "null actual command", actual: nil, want: launcher.SessionOwnerUnknown},
		{name: "ambiguous actual command", actual: "pair wrap '$UNRESOLVED", want: launcher.SessionOwnerUnknown},
		{name: "foreign owner", foreign: true, want: launcher.SessionOwnerForeign},
		{name: "server replaced during observation", replacement: true, want: launcher.SessionOwnerUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proc := NewFakeProcOps()
			proc.Set(501, "codex")
			observedAddress := address
			if tc.foreign {
				observedAddress.Tag = "couch-2222222222222222"
			}
			world := newSlotTreeOwnerWorld(t, proc, root, observedAddress, "pair-owner-797")
			if tc.name != "valid" && !tc.foreign && !tc.replacement {
				world.live = []byte(zellijtest.Reply(t, "panes", []map[string]any{{"id": 1, "pane_command": tc.actual}}).Stdout)
			}
			world.replaceOnPanes = tc.replacement
			got, err := (launcher.SessionOwnerProbe{IO: world}).Probe(t.Context(), world.server.Session, root, address.RepoScope, string(address.Tag))
			if err != nil || got.State != tc.want {
				t.Fatalf("owner=%+v err=%v want=%v", got, err, tc.want)
			}
			if tc.replacement && got.Diagnostic != "session server changed during observation" {
				t.Fatalf("replacement diagnostic=%q", got.Diagnostic)
			}
		})
	}
}
