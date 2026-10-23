package couchcmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestSwitchAgentInterruptConsentQueuesBusySource(t *testing.T) {
	rig, queue, hub, worker, client := leanSwitchRig(t)
	client.Settle(false)
	testwait.Until(t, "busy source", func() bool {
		live, ok := rig.s.LivenessForThread(worker.Scope, worker.Tag)
		return ok && live.Settled != nil && !*live.Settled
	})
	ordinary := switchRequest(hub, "ordinary", worker.Slot)
	if response := switchCall(t, rig, ordinary); response.Code == "accepted" || queue.count() != 0 {
		t.Fatalf("ordinary busy switch admitted: %+v", response)
	}
	cooperative := switchRequest(hub, "cooperative", worker.Slot)
	cooperative.InterruptFirst = true
	response := switchCall(t, rig, cooperative)
	if response.Code != "accepted" || queue.count() != 1 {
		t.Fatalf("confirmed cooperative intent refused: %+v", response)
	}
	call, err := queue.job(0).prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if call.Args["interrupt-first"] != "true" {
		t.Fatalf("queue lost cooperative intent: %+v", call.Args)
	}
	status := statusRequest(hub, "cooperative")
	if response := switchCall(t, rig, status); response.Operation == nil || response.Operation.Status != couchmessage.ReceiptQueued {
		t.Fatalf("no retained cooperative receipt: %+v", response)
	}
}

type switchInterruptEndpoint struct {
	driveServiceFake
	observed func(context.Context)
	started  func()
}

func (e *switchInterruptEndpoint) ObserveDrive(ctx context.Context) (couchmessage.DriveObservation, error) {
	observation, err := e.driveServiceFake.ObserveDrive(ctx)
	if e.observed != nil {
		e.observed(ctx)
	}
	return observation, err
}
func (e *switchInterruptEndpoint) StartDrive(ctx context.Context, id string, start couchmessage.DriveStart) (agentdrive.Result, error) {
	result, err := e.driveServiceFake.StartDrive(ctx, id, start)
	if e.started != nil {
		e.started()
	}
	return result, err
}

func TestInterruptBeforeAgentSwitchUsesOneDriveAndFreshIdleProof(t *testing.T) {
	for _, sourceAgent := range []string{"claude", "codex"} {
		t.Run(sourceAgent, func(t *testing.T) {
			for _, scenario := range []string{"working-local", "already-idle-local", "working", "already-idle", "unknown-becomes-idle", "unknown-cancel", "dirty-before", "dirty-after-observation", "source-change", "record-change", "lost-reply", "unconfirmed", "confirmed-without-idle"} {
				t.Run(scenario, func(t *testing.T) {
					local := strings.HasSuffix(scenario, "-local")
					scenario := strings.TrimSuffix(scenario, "-local")
					rt := newRT(t, "/repo")
					c, err := rt.NewCouch()
					if err != nil {
						t.Fatal(err)
					}
					c.RootAgent = sourceAgent
					targetAgent := "claude"
					if sourceAgent == "claude" {
						targetAgent = "codex"
					}
					actor, handle, err := c.Spawn(couchcore.StartArgs{Worktree: "/repo"})
					if err != nil {
						t.Fatal(err)
					}
					rt.proc.Set(handle.PID(), handle.Identity())
					rt.artifacts.SetPairSession(actor.Thread, managedChildSession(t, rt.runner, handle.ID()), true)
					rt.artifacts.SetSessionPresence(actor.Thread, couchcore.SessionObservation{State: couchcore.SessionPresent})
					c.PairLifecycle = &couchcore.PairLifecycleController{}
					argv := []string{}
					prepared, err := c.PrepareAgentSwitch(context.Background(), actor.Thread, targetAgent, &argv)
					if err != nil {
						t.Fatal(err)
					}
					binding := couchmessage.Binding{Slot: "repo:2", Scope: actor.Thread.RepoScope, Tag: string(actor.Thread.Tag), PID: handle.PID(), Session: "session", Nonce: "nonce"}
					rest := "main-slot2"
					identity := couchcore.WorkspaceIdentity{WorktreeRoot: "/repo", RestingBranch: &rest}
					endpoint := &switchInterruptEndpoint{driveServiceFake: driveServiceFake{observation: couchmessage.DriveObservation{Interruptible: true, Instance: "working-wrapper", Sequence: 51}}}
					service := &messageService{authority: messageAuthority{
						thread:     func(context.Context, couchmessage.Binding) (string, error) { return "/repo", nil },
						wrapperPID: func(couchmessage.Binding) (int, error) { return binding.PID, nil },
						recorded:   func(context.Context, couchmessage.Binding) error { return nil },
						branch:     func(context.Context, string) (couchcore.SlotGitStatus, error) { return couchcore.SlotGitStatus{}, nil },
						record: func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
							return c.Threads.PreviewThreadContext(ctx, address)
						},
						endpoint: func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return endpoint },
					}, workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{binding: identity}}
					service.connected.Store(&map[couchmessage.Binding]bool{binding: true})
					settle := func(idle bool) {
						service.liveness.Store(&map[string]couchmessage.SlotLiveness{binding.Slot: {Binding: binding, Session: 1, Settled: &idle}})
					}
					settle(scenario == "already-idle")
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					endpoint.observed = func(ctx context.Context) {
						deadline, ok := ctx.Deadline()
						if !ok || deadline.After(time.Now().Add(30*time.Second)) {
							t.Error("interrupt phase did not inherit one bounded deadline")
						}
					}
					endpoint.started = func() { settle(true) }
					switch scenario {
					case "unknown-becomes-idle", "unknown-cancel":
						endpoint.observation.Interruptible = false
						check := endpoint.observed
						endpoint.observed = func(ctx context.Context) {
							check(ctx)
							if scenario == "unknown-cancel" {
								cancel()
							} else {
								settle(true)
							}
						}
					case "dirty-before":
						service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
							return couchcore.SlotGitStatus{Dirty: true}, nil
						}
					case "dirty-after-observation":
						check := endpoint.observed
						endpoint.observed = func(ctx context.Context) {
							check(ctx)
							service.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
								return couchcore.SlotGitStatus{Dirty: true}, nil
							}
						}
					case "source-change":
						check := endpoint.observed
						endpoint.observed = func(ctx context.Context) {
							check(ctx)
							live, _ := service.LivenessForThread(binding.Scope, binding.Tag)
							live.Session++
							service.liveness.Store(&map[string]couchmessage.SlotLiveness{binding.Slot: live})
						}
					case "record-change":
						check := endpoint.observed
						endpoint.observed = func(ctx context.Context) {
							check(ctx)
							name := "changed"
							if _, err := c.Threads.ApplyThreadMetadata(prepared.Address, prepared.SourceRevision, couchcore.ThreadMetadataPatch{Name: &name}); err != nil {
								t.Fatal(err)
							}
						}
					case "lost-reply":
						endpoint.lost = true
						endpoint.started = nil
					case "unconfirmed":
						endpoint.pending = true
						endpoint.started = cancel
					case "confirmed-without-idle":
						endpoint.started = cancel
					}
					guard := &switchServiceGuard{}
					guard.service.Store(service)
					expected := switchSourceToken(couchmessage.SlotLiveness{Binding: binding, Session: 1})
					token := expected
					if local {
						token = ""
					}
					pinned, err := guard.InterruptBeforeAgentSwitch(ctx, prepared.Address, token, prepared.SourceEvidence())
					success := scenario == "working" || scenario == "already-idle" || scenario == "unknown-becomes-idle"
					if (err == nil) != success {
						t.Fatalf("%s: %v", scenario, err)
					}
					if success && pinned != expected {
						t.Fatalf("lost verified source token: %q", pinned)
					}
					starts, _ := endpoint.counts()
					wantStarts := 0
					if scenario == "working" || scenario == "lost-reply" || scenario == "unconfirmed" || scenario == "confirmed-without-idle" {
						wantStarts = 1
					}
					if starts != wantStarts {
						t.Fatalf("interrupt starts=%d want=%d", starts, wantStarts)
					}
					if scenario == "unknown-cancel" && (!errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), agentdrive.InterruptRefusal)) {
						t.Fatalf("lost exact no-input diagnostic: %v", err)
					}
				})
			}
		})
	}
}
