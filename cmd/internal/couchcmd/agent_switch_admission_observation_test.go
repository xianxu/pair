package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type switchAdmissionObservationRig struct {
	cold      *coldResumeRig
	service   *serviceRig
	queue     *leanSwitchQueue
	hub       couchmessage.Binding
	hubs      *hubRuntime
	resolved  atomic.Int32
	workspace atomic.Int32
	record    atomic.Int32
	git       atomic.Int32
}

func TestSwitchAdmissionObservationAllowsNestedWorkingDirectory(t *testing.T) {
	for _, kind := range []string{"nested", "symlink-escape", "file", "symlink-during-git", "missing-during-git"} {
		t.Run(kind, func(t *testing.T) {
			r := newSwitchAdmissionObservationRig(t)
			before := r.cold.parked
			next := before
			next.Address.Tag += "-nested"
			next.Revision = 1
			next.WorkingPath = filepath.Join(before.StartingPath, "nested")
			next.ParkHistory, next.VerifiedPark, next.PreviousSession = nil, nil, nil
			var err error
			switch kind {
			case "symlink-escape":
				err = os.Symlink(t.TempDir(), next.WorkingPath)
			case "file":
				err = os.WriteFile(next.WorkingPath, []byte("not a directory"), 0600)
			default:
				err = os.MkdirAll(next.WorkingPath, 0700)
			}
			if err != nil {
				t.Fatal(err)
			}
			proof, err := couchcore.TestOnlyTeardownProof(before, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := r.cold.c.Threads.ReplaceThreadExpected(before.Address, before.Revision, next, proof); err != nil {
				t.Fatal(err)
			}
			r.cold.rt.artifacts.SetNativeBinding(next.Address, "claude", sessioninventory.BindingEstablished, "nested-native")
			r.cold.rt.artifacts.SetPairSession(next.Address, "nested-session", false)
			r.cold.parked = next
			if kind == "symlink-during-git" || kind == "missing-during-git" {
				outside := t.TempDir()
				r.service.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					if err := os.Remove(next.WorkingPath); err != nil {
						return couchcore.SlotGitStatus{}, err
					}
					if kind == "symlink-during-git" {
						return couchcore.SlotGitStatus{}, os.Symlink(outside, next.WorkingPath)
					}
					return couchcore.SlotGitStatus{}, nil
				}
			}
			response := switchCall(t, r.service, r.request(t, "codex"))
			if kind == "nested" {
				if response.Code != "accepted" || r.queue.count() != 1 {
					t.Fatalf("nested working directory refused: %+v", response)
				}
				if r.resolved.Load() != 1 || r.workspace.Load() != 0 {
					t.Fatal("nested directory repeated workspace probes")
				}
			} else if response.Code == "accepted" || r.queue.count() != 0 {
				t.Fatalf("unsafe working directory admitted: %+v", response)
			}
		})
	}
}

func newSwitchAdmissionObservationRig(t *testing.T) *switchAdmissionObservationRig {
	t.Helper()
	cold := newColdResumeRig(t)
	rig, queue, hub, _, _ := leanSwitchRig(t)
	r := &switchAdmissionObservationRig{cold: cold, service: rig, queue: queue, hub: hub, hubs: guardHubState(t)}
	rig.s.installAgentSwitchService(cold.console, cold.c, &switchServiceGuard{})
	switches := rig.s.switches.Load()
	switches.enqueue = queue.enqueue
	resolve := switches.resolveTarget
	switches.resolveTarget = func(ctx context.Context, target string) (couchcore.ThreadAddress, couchcore.WorkspaceIdentity, error) {
		r.resolved.Add(1)
		return resolve(ctx, target)
	}
	rig.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: cold.parked.StartingPath, Dir: "repo"}}, nil
	}
	workspace := cold.c.Slots.(couchcore.SlotWorkspaceResolver).ResolveWorkspace
	rig.s.authority.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
		r.workspace.Add(1)
		return workspace(ctx, path)
	}
	rig.s.authority.record = func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
		r.record.Add(1)
		return cold.c.Threads.PreviewThreadContext(ctx, address)
	}
	rig.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
		r.git.Add(1)
		return couchcore.SlotGitStatus{}, nil
	}
	rig.s.authority.hubs = r.hubs
	return r
}

func (r *switchAdmissionObservationRig) request(t *testing.T, agent string) couchmessage.Request {
	t.Helper()
	request := switchRequest(r.hub, "one-observation", r.cold.address)
	request.Agent = agent
	var err error
	request.AgentArgs, err = couchmessage.NewSwitchArguments([]string{})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func (r *switchAdmissionObservationRig) moveHub() error {
	r.hubs.mu.Lock()
	defer r.hubs.mu.Unlock()
	next, err := r.hubs.state.Apply(couchcore.HubMove{Caller: "/repo0", Target: "/repo1", Revision: r.hubs.state.Revision(), Slots: []couchcore.HubSlot{{Repository: r.cold.parked.StartingPath, Number: 0}}})
	if err == nil {
		r.hubs.state = next
	}
	return err
}

func TestSwitchAdmissionReusesOneWorkspaceObservationAndRechecksExecution(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			r := newSwitchAdmissionObservationRig(t)
			response := switchCall(t, r.service, r.request(t, agent))
			if response.Code != "accepted" || r.queue.count() != 1 {
				t.Fatalf("vacant admission: %+v jobs=%d", response, r.queue.count())
			}
			if r.resolved.Load() != 1 || r.workspace.Load() != 0 {
				t.Errorf("admission duplicated workspace proof: resolutions=%d workspace=%d; want 1/0", r.resolved.Load(), r.workspace.Load())
			}
			if r.record.Load() < 2 || r.git.Load() != 1 {
				t.Errorf("admission lost before/after record or fresh Git proof: records=%d git=%d", r.record.Load(), r.git.Load())
			}
			resolved, workspace := r.resolved.Load(), r.workspace.Load()
			call, err := r.queue.job(0).prepare(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if r.resolved.Load() <= resolved || r.workspace.Load() <= workspace {
				t.Errorf("queued proof reused admission workspace: resolutions=%d->%d workspace=%d->%d", resolved, r.resolved.Load(), workspace, r.workspace.Load())
			}
			resolved, workspace = r.resolved.Load(), r.workspace.Load()
			if _, err := unwrapHubMutation(context.Background(), call.TypedPayload); err != nil {
				t.Fatal(err)
			}
			if r.resolved.Load() <= resolved || r.workspace.Load() <= workspace {
				t.Errorf("effect proof reused earlier workspace: resolutions=%d->%d workspace=%d->%d", resolved, r.resolved.Load(), workspace, r.workspace.Load())
			}
		})
	}
}

func TestSwitchAdmissionObservationPreservesFreshRefusals(t *testing.T) {
	for _, fault := range []string{"observation-repository", "observation-path", "selection", "selection-during-git", "record-revision", "reconnected", "dirty", "unknown-git", "cancelled", "hub-change"} {
		t.Run(fault, func(t *testing.T) {
			r := newSwitchAdmissionObservationRig(t)
			switches := r.service.s.switches.Load()
			resolve := switches.resolveTarget
			switches.resolveTarget = func(ctx context.Context, target string) (couchcore.ThreadAddress, couchcore.WorkspaceIdentity, error) {
				address, identity, err := resolve(ctx, target)
				if err != nil || r.resolved.Load() != 1 {
					return address, identity, err
				}
				switch fault {
				case "observation-repository":
					identity.RepoIdentity += "-other"
				case "observation-path":
					identity.WorktreeRoot += "-other"
				case "selection":
					replacement := r.cold.parked
					replacement.Address.Tag += "-replacement"
					replacement.Revision = 1
					replacement.ParkHistory, replacement.VerifiedPark, replacement.PreviousSession = nil, nil, nil
					_, err = r.cold.c.Threads.CreateThread(replacement)
				case "hub-change":
					err = r.moveHub()
				}
				return address, identity, err
			}
			r.service.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
				r.git.Add(1)
				switch fault {
				case "selection-during-git":
					replacement := r.cold.parked
					replacement.Address.Tag += "-replacement"
					replacement.Revision = 1
					replacement.ParkHistory, replacement.VerifiedPark, replacement.PreviousSession = nil, nil, nil
					_, err := r.cold.c.Threads.CreateThread(replacement)
					return couchcore.SlotGitStatus{}, err
				case "record-revision":
					name := "changed during Git"
					_, err := r.cold.c.Threads.ApplyThreadMetadata(r.cold.parked.Address, r.cold.parked.Revision, couchcore.ThreadMetadataPatch{Name: &name})
					return couchcore.SlotGitStatus{}, err
				case "reconnected":
					binding := couchmessage.Binding{Slot: r.cold.address, Scope: r.cold.parked.Address.RepoScope, Tag: string(r.cold.parked.Address.Tag)}
					r.service.s.connected.Store(&map[couchmessage.Binding]bool{binding: true})
					r.service.s.liveness.Store(&map[string]couchmessage.SlotLiveness{binding.Slot: {Binding: binding}})
				case "dirty":
					return couchcore.SlotGitStatus{Dirty: true}, nil
				case "unknown-git":
					return couchcore.SlotGitStatus{}, errors.New("Git unavailable")
				case "cancelled":
					return couchcore.SlotGitStatus{}, context.Canceled
				}
				return couchcore.SlotGitStatus{}, nil
			}
			response := switchCall(t, r.service, r.request(t, "codex"))
			if response.Code == "accepted" || r.queue.count() != 0 {
				t.Fatalf("changed admission %s queued: %+v jobs=%d", fault, response, r.queue.count())
			}
		})
	}
}
