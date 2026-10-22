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
}

func (w *slotTreeOwnerWorld) SessionServers(context.Context, string) ([]launcher.SessionServerIdentity, error) {
	return []launcher.SessionServerIdentity{w.server}, nil
}
func (*slotTreeOwnerWorld) SessionPresent(context.Context, string) (bool, error) { return true, nil }
func (*slotTreeOwnerWorld) Socket(string) launcher.SocketState                   { return launcher.SocketPresent }
func (w *slotTreeOwnerWorld) SessionPanes(context.Context, string) ([]byte, error) {
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

// Diagnostic characterization for pair#797, not the desired recovery contract.
// Convert the refusal assertion to successful recovery when the fix is approved.
func TestSlotTreeOwnerCharacterizationLosesPaneEvidenceAfterStoppingTree(t *testing.T) {
	f := slotTreeFixture(t)
	root := t.TempDir()
	name := recordSessionName(f.record)
	world := newSlotTreeOwnerWorld(t, f.Proc, root, f.record.Address, name)
	originalServer := world.server
	checker := NewScopedThreadArtifactCollisionChecker(root)
	deleter := &fakeSessionDeleter{}
	checker.Sessions = deleter
	checker.OwnerProbe = launcher.SessionOwnerProbe{IO: world}
	checker.Servers = &ownerPassSource{states: map[string]launcher.ServerState{name: {Verdict: launcher.ServerReachable, Server: world.server}}}
	f.Couch.Artifacts = checker
	f.Proc.OnSignal = func(pid int, _ os.Signal) { world.events = append(world.events, fmt.Sprintf("stop:%d", pid)) }

	// Establish that this exact owner and native reply are valid before retirement.
	binding, err := checker.NamedPairSessionContext(t.Context(), f.record.Address, name)
	if err != nil || !binding.Present || binding.Owner.Server != originalServer {
		t.Fatalf("before stop: binding=%+v err=%v", binding, err)
	}
	world.events = nil
	selection := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh}
	_, err = f.Couch.RecoverSelection(t.Context(), selection)
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) {
		t.Fatalf("selection did not offer confirmation: %v", err)
	}
	if len(world.events) != 0 {
		t.Fatalf("selection had effects: %v", world.events)
	}
	roles := []string{}
	for _, stop := range required.Offer.Stops {
		roles = append(roles, stop.Role)
	}
	if !strings.Contains(strings.Join(roles, ","), slotSessionRole+name) {
		t.Fatalf("confirmation omitted session: %v", roles)
	}
	result, err := f.Couch.ConfirmRecovery(t.Context(), required.Offer.Token)
	if err == nil || !strings.Contains(err.Error(), "kill not confirmed") || !strings.Contains(err.Error(), "pane has no unambiguous actual command") {
		t.Fatalf("expected characterized post-stop owner refusal: result=%+v err=%v events=%v", result, err, world.events)
	}
	want := []string{"stop:500", "stop:501", "stop:502", "panes:stopped"}
	if !reflect.DeepEqual(world.events, want) {
		t.Fatalf("retirement order=%v want=%v", world.events, want)
	}
	for _, pid := range []int{500, 501, 502} {
		if f.Proc.Exists(pid) != Dead {
			t.Fatalf("pid %d survived", pid)
		}
	}
	if f.Proc.Exists(600) != Live || len(f.Proc.Signals[600]) != 0 {
		t.Fatal("unrelated process touched")
	}
	if world.server != originalServer {
		t.Fatalf("fixture replaced server: %+v", world.server)
	}
	if len(deleter.deleted) != 0 || len(f.Runner.order) != 0 || result.Start.Handle != nil {
		t.Fatalf("refusal caused effects: deleted=%v launches=%v result=%+v", deleter.deleted, f.Runner.order, result)
	}
	t.Logf("same server %+v; order %v; refusal %v; no session deletion or launch", world.server, world.events, err)
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
