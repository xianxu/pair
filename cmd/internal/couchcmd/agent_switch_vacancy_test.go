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
