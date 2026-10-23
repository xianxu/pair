package couchcmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

type forceSwitchRig struct {
	guard    *switchServiceGuard
	service  *messageService
	couch    *couchcore.Couch
	prepared couchcore.PreparedAgentSwitch
	binding  couchmessage.Binding
	endpoint *driveServiceFake
}

func newForceSwitchRig(t *testing.T, vacant bool, sourceAgent string) forceSwitchRig {
	t.Helper()
	rt := newRT(t, "/repo")
	var address couchcore.ThreadAddress
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	if vacant {
		scope, err := launcher.ResolveRepoScope("/repo")
		if err != nil {
			t.Fatal(err)
		}
		profile := couchcore.LaunchProfile{Agent: sourceAgent, Argv: []string{}}
		record, err := c.Threads.CreateThread(couchcore.ThreadRecord{
			SchemaVersion: couchcore.ThreadSchemaVersion,
			Address:       couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"},
			StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
			Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: couchcore.IncarnationLive, RepoIdentity: "/repo/.git", LaunchProfile: &profile}},
			LatestLaunchProfile: &profile,
		})
		if err != nil {
			t.Fatal(err)
		}
		park := couchcore.ParkIdentity{Nonce: "force-fixture-park", Address: record.Address, PID: 42, ProcessIdentity: "pair-helper"}
		record, err = c.Threads.BeginPark(record.Address, record.Revision, park)
		if err != nil {
			t.Fatal(err)
		}
		record, err = c.Threads.FinalizePark(context.Background(), record.Address, record.Revision, park, 1, time.Unix(2, 0).UTC())
		if err != nil {
			t.Fatal(err)
		}
		rt.artifacts.SetPairSession(record.Address, "pair-"+string(record.Address.Tag), false)
		address = record.Address
	} else {
		c.RootAgent = sourceAgent
		actor, handle, err := c.Spawn(couchcore.StartArgs{Worktree: "/repo"})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { rt.runner.SetExited(handle.ID(), 0) })
		rt.proc.Set(handle.PID(), handle.Identity())
		rt.artifacts.SetPairSession(actor.Thread, managedChildSession(t, rt.runner, handle.ID()), true)
		rt.artifacts.SetSessionPresence(actor.Thread, couchcore.SessionObservation{State: couchcore.SessionPresent})
		c.PairLifecycle = &couchcore.PairLifecycleController{}
		address = actor.Thread
	}
	argv := []string{}
	targetAgent := "claude"
	if sourceAgent == "claude" {
		targetAgent = "codex"
	}
	prepared, err := c.PrepareAgentSwitch(context.Background(), address, targetAgent, &argv)
	if err != nil {
		t.Fatal(err)
	}
	binding := couchmessage.Binding{Slot: "repo:0", Scope: address.RepoScope, Tag: string(address.Tag), Agent: sourceAgent, PID: 42, Session: "session", Nonce: "nonce", Repository: "/repo/.git"}
	rest, number, addressText := "main", 0, "repo:0"
	identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: "/repo/.git", PrimaryRoot: "/repo", FleetRoot: "/", EnvironmentRoot: "/", WorktreeRoot: "/repo", Kind: "primary", Slot: &number, Address: &addressText, RestingBranch: &rest}
	endpoint := &driveServiceFake{}
	service := &messageService{authority: messageAuthority{
		thread:     func(context.Context, couchmessage.Binding) (string, error) { return "/repo", nil },
		wrapperPID: func(couchmessage.Binding) (int, error) { return binding.PID, nil },
		recorded:   func(context.Context, couchmessage.Binding) error { return nil },
		process:    func(couchmessage.Binding) error { return nil },
		workspace:  func(context.Context, string) (couchcore.WorkspaceIdentity, error) { return identity, nil },
		branch: func(context.Context, string) (couchcore.SlotGitStatus, error) {
			return couchcore.ParseSlotGitStatus("# branch.head main\n")
		},
		record:   c.Threads.PreviewThreadContext,
		endpoint: func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return endpoint },
	}, workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{binding: identity}}
	connected := map[couchmessage.Binding]bool{}
	liveness := map[string]couchmessage.SlotLiveness{}
	if !vacant {
		busy := false
		connected[binding] = true
		liveness[binding.Slot] = couchmessage.SlotLiveness{Binding: binding, Session: 1, Settled: &busy}
	}
	service.connected.Store(&connected)
	service.liveness.Store(&liveness)
	guard := &switchServiceGuard{}
	guard.service.Store(service)
	return forceSwitchRig{guard: guard, service: service, couch: c, prepared: prepared, binding: binding, endpoint: endpoint}
}

func (r forceSwitchRig) replaceRecord(t *testing.T) {
	t.Helper()
	name := "changed generation"
	if _, err := r.couch.Threads.ApplyThreadMetadata(r.prepared.Address, r.prepared.SourceRevision, couchcore.ThreadMetadataPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
}

func TestForcedSwitchSupervisorAllowsOnlyIdleAndGitPolicyWaivers(t *testing.T) {
	for _, sourceAgent := range []string{"claude", "codex"} {
		for _, vacant := range []bool{false, true} {
			for _, scenario := range []string{"busy", "unknown-idle", "dirty-count", "unknown-git"} {
				t.Run(sourceAgent+"/"+scenario+map[bool]string{true: "/vacant", false: "/live"}[vacant], func(t *testing.T) {
					r := newForceSwitchRig(t, vacant, sourceAgent)
					if scenario == "unknown-idle" && !vacant {
						live, _ := r.service.LivenessForThread(r.binding.Scope, r.binding.Tag)
						live.Settled = nil
						r.service.liveness.Store(&map[string]couchmessage.SlotLiveness{r.binding.Slot: live})
					}
					status := "# branch.head main\n"
					if scenario == "dirty-count" {
						status += "? one.txt\n? two.txt\n"
					}
					gitCalls := 0
					r.service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
						gitCalls++
						if scenario == "unknown-git" {
							untrusted := 99
							return couchcore.SlotGitStatus{ChangedFiles: &untrusted}, errors.New("Git observation unavailable")
						}
						return couchcore.ParseSlotGitStatus(status)
					}
					source, count, err := r.guard.PrepareForcedAgentSwitch(context.Background(), r.prepared.Address, r.prepared.SourceEvidence())
					if err != nil || gitCalls != 1 || (source == "") != vacant {
						t.Fatalf("force preparation token=%q count=%v calls=%d err=%v", source, count, gitCalls, err)
					}
					if scenario == "unknown-git" {
						if count != nil {
							t.Fatal("unknown Git invented a count")
						}
					} else {
						want := 0
						if scenario == "dirty-count" {
							want = 2
						}
						if count == nil || *count != want {
							t.Fatalf("count=%v want=%d", count, want)
						}
					}
					if err := r.guard.ConfirmForcedAgentSwitch(context.Background(), r.prepared.Address, source, r.prepared.SourceEvidence()); err != nil {
						t.Fatal(err)
					}
					if starts, cancels := r.endpoint.counts(); starts != 0 || cancels != 0 {
						t.Fatalf("force proof sent input: starts=%d cancels=%d", starts, cancels)
					}
				})
			}
		}
	}
}

func TestForcedSwitchSupervisorRejectsChangedAuthority(t *testing.T) {
	for _, sourceAgent := range []string{"claude", "codex"} {
		for _, vacant := range []bool{false, true} {
			for _, scenario := range []string{"record-before", "record-during-git", "record-after-offer", "workspace-error", "workspace-changed", "workspace-during-git", "record-during-workspace", "wrapper-during-workspace", "cancelled", "zero-evidence", "wrapper-changed", "disconnect", "process", "recorded-owner", "missing-token"} {
				t.Run(sourceAgent+"/"+scenario+map[bool]string{true: "/vacant", false: "/live"}[vacant], func(t *testing.T) {
					r := newForceSwitchRig(t, vacant, sourceAgent)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					source, _, err := r.guard.PrepareForcedAgentSwitch(ctx, r.prepared.Address, r.prepared.SourceEvidence())
					if err != nil {
						t.Fatal(err)
					}
					evidence := r.prepared.SourceEvidence()
					switch scenario {
					case "record-before", "record-after-offer":
						r.replaceRecord(t)
					case "record-during-git":
						r.service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
							r.replaceRecord(t)
							return couchcore.SlotGitStatus{}, nil
						}
					case "workspace-error":
						r.service.authority.workspace = func(context.Context, string) (couchcore.WorkspaceIdentity, error) {
							return couchcore.WorkspaceIdentity{}, errors.New("workspace unavailable")
						}
					case "workspace-changed":
						original := r.service.authority.workspace
						r.service.authority.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
							id, err := original(ctx, path)
							id.RepoIdentity = "/other/.git"
							return id, err
						}
					case "workspace-during-git":
						r.service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
							r.service.authority.workspace = func(context.Context, string) (couchcore.WorkspaceIdentity, error) {
								return couchcore.WorkspaceIdentity{}, errors.New("workspace changed during observation")
							}
							return couchcore.SlotGitStatus{}, nil
						}
					case "record-during-workspace", "wrapper-during-workspace":
						original := r.service.authority.workspace
						calls := 0
						r.service.authority.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
							identity, err := original(ctx, path)
							calls++
							if calls == 2 {
								if scenario == "record-during-workspace" {
									r.replaceRecord(t)
								} else {
									live, _ := r.service.LivenessForThread(r.binding.Scope, r.binding.Tag)
									live.Binding, live.Session = r.binding, live.Session+1
									r.service.connected.Store(&map[couchmessage.Binding]bool{r.binding: true})
									r.service.liveness.Store(&map[string]couchmessage.SlotLiveness{r.binding.Slot: live})
								}
							}
							return identity, err
						}
					case "cancelled":
						cancel()
					case "zero-evidence":
						evidence = couchcore.SwitchSourceEvidence{}
					case "wrapper-changed":
						if vacant {
							r.service.connected.Store(&map[couchmessage.Binding]bool{r.binding: true})
							r.service.liveness.Store(&map[string]couchmessage.SlotLiveness{r.binding.Slot: {Binding: r.binding, Session: 2}})
						} else {
							live, _ := r.service.LivenessForThread(r.binding.Scope, r.binding.Tag)
							live.Session++
							r.service.liveness.Store(&map[string]couchmessage.SlotLiveness{r.binding.Slot: live})
						}
					case "disconnect":
						if vacant {
							source = "old-live-token"
						} else {
							r.service.connected.Store(&map[couchmessage.Binding]bool{})
						}
					case "process":
						if vacant {
							source = "old-live-token"
						} else {
							r.service.authority.process = func(couchmessage.Binding) error { return errors.New("process replaced") }
						}
					case "recorded-owner":
						if vacant {
							source = "old-live-token"
						} else {
							r.service.authority.recorded = func(context.Context, couchmessage.Binding) error { return errors.New("recorded owner replaced") }
						}
					case "missing-token":
						if vacant {
							source = "old-live-token"
						} else {
							source = ""
						}
					}
					if err := r.guard.ConfirmForcedAgentSwitch(ctx, r.prepared.Address, source, evidence); err == nil {
						t.Fatal("force confirmation accepted changed authority")
					}
				})
			}
		}
	}
}
