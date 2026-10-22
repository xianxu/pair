package couchcmd

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestSwitchGuardAdmitsCoreProvedVacancyWithoutWrapper(t *testing.T) {
	for _, fault := range []string{"none", "no-proof", "old-live-token", "dirty", "unknown-git", "workspace-error", "unreadable", "changed-before", "changed-during-git", "reconnected-during-git", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			rig := newColdResumeRig(t)
			argv := []string{}
			prepared, err := rig.c.PrepareAgentSwitch(context.Background(), rig.parked.Address, "codex", &argv)
			if err != nil {
				t.Fatal(err)
			}
			resolver := rig.c.Slots.(couchcore.SlotWorkspaceResolver)
			service := &messageService{authority: messageAuthority{
				workspace: resolver.ResolveWorkspace,
				record: func(_ context.Context, address couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
					return rig.c.Threads.GetThread(address)
				},
				branch: func(context.Context, string) (couchcore.SlotGitStatus, error) { return couchcore.SlotGitStatus{}, nil },
			}}
			replace := func() error {
				name := "replacement"
				_, err := rig.c.Threads.ApplyThreadMetadata(prepared.Address, prepared.SourceRevision, couchcore.ThreadMetadataPatch{Name: &name})
				return err
			}
			token, evidence := "", prepared.SourceEvidence()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch fault {
			case "no-proof":
				evidence = couchcore.SwitchSourceEvidence{}
			case "old-live-token":
				token = "previous-live-binding"
			case "dirty":
				service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{Dirty: true}, nil
				}
			case "unknown-git":
				service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{}, errors.New("git unavailable")
				}
			case "workspace-error":
				service.authority.workspace = func(context.Context, string) (couchcore.WorkspaceIdentity, error) {
					return couchcore.WorkspaceIdentity{}, errors.New("workspace unknown")
				}
			case "unreadable":
				service.authority.record = func(context.Context, couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
					return couchcore.ThreadRecord{}, errors.New("record unreadable")
				}
			case "changed-before":
				if err := replace(); err != nil {
					t.Fatal(err)
				}
			case "changed-during-git":
				service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{}, replace()
				}
			case "reconnected-during-git":
				service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					binding := couchmessage.Binding{Slot: "repo:0", Scope: prepared.Address.RepoScope, Tag: string(prepared.Address.Tag)}
					service.connected.Store(&map[couchmessage.Binding]bool{binding: true})
					service.liveness.Store(&map[string]couchmessage.SlotLiveness{binding.Slot: {Binding: binding}})
					return couchcore.SlotGitStatus{}, nil
				}
			case "cancelled":
				cancel()
			}
			guard := &switchServiceGuard{}
			guard.service.Store(service)
			err = guard.ConfirmAgentSwitch(ctx, prepared.Address, token, evidence)
			if (err == nil) != (fault == "none") {
				t.Fatalf("vacant %s: %v", fault, err)
			}
		})
	}
}

func TestSwitchAgentSocketSelectsRetainedParkedTargetWithoutWrapper(t *testing.T) {
	for _, fault := range []string{"none", "foreign-hub", "queued-selection", "queued-dirty", "queued-owner", "effect-owner"} {
		t.Run(fault, func(t *testing.T) {
			cold := newColdResumeRig(t)
			rig, queue, hub, _, _ := leanSwitchRig(t)
			guard := &switchServiceGuard{}
			rig.s.installAgentSwitchService(cold.console, cold.c, guard)
			rig.s.switches.Load().enqueue = queue.enqueue
			rig.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
				return []couchcore.RepositoryName{{Key: cold.parked.StartingPath, Dir: "repo"}}, nil
			}
			rig.s.authority.workspace = cold.c.Slots.(couchcore.SlotWorkspaceResolver).ResolveWorkspace
			rig.s.authority.record = func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
				return cold.c.Threads.PreviewThreadContext(ctx, address)
			}
			rig.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) { return couchcore.SlotGitStatus{}, nil }
			hubs := guardHubState(t)
			rig.s.authority.hubs = hubs
			move := func() {
				hubs.mu.Lock()
				defer hubs.mu.Unlock()
				next, err := hubs.state.Apply(couchcore.HubMove{Caller: "/repo0", Target: "/repo1", Revision: hubs.state.Revision(), Slots: []couchcore.HubSlot{{Repository: cold.parked.StartingPath, Number: 0}}})
				if err != nil {
					t.Fatal(err)
				}
				hubs.state = next
			}
			if fault == "foreign-hub" {
				move()
			}
			request := switchRequest(hub, "parked", cold.address)
			request.Agent = "codex"
			var err error
			request.AgentArgs, err = couchmessage.NewSwitchArguments([]string{})
			if err != nil {
				t.Fatal(err)
			}
			response := switchCall(t, rig, request)
			if fault == "foreign-hub" {
				if response.Code == "accepted" || queue.count() != 0 {
					t.Fatalf("foreign vacant source queued: %+v", response)
				}
				return
			}
			if response.Code != "accepted" || queue.count() != 1 {
				t.Fatalf("retained parked target refused: %+v queued=%d", response, queue.count())
			}
			switch fault {
			case "queued-selection":
				replacement := cold.parked
				replacement.Address.Tag += "-replacement"
				replacement.Revision = 1
				replacement.ParkHistory = nil
				replacement.VerifiedPark = nil
				replacement.PreviousSession = nil
				if _, err := cold.c.Threads.CreateThread(replacement); err != nil {
					t.Fatal(err)
				}
			case "queued-dirty":
				rig.s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{Dirty: true}, nil
				}
			case "queued-owner":
				move()
			}
			call, err := queue.job(0).prepare(context.Background())
			if fault == "queued-selection" || fault == "queued-dirty" || fault == "queued-owner" {
				if err == nil {
					t.Fatalf("changed source passed queued proof: %+v", call)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if call.Args["repo-scope"] != cold.parked.Address.RepoScope || call.Args["tag"] != string(cold.parked.Address.Tag) || call.Args["source"] != "" || call.Args["argv"] != "[]" {
				t.Fatalf("wrong retained source: %+v", call.Args)
			}
			if fault == "effect-owner" {
				move()
				if _, err := unwrapHubMutation(context.Background(), call.TypedPayload); err == nil {
					t.Fatal("hub move passed the retained-source effect guard")
				}
			}
		})
	}
}
