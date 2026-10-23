package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Seed the public claim/register/park transitions before Console.Run starts
// any observers against the shared thread store.
func editedRestartConsoleSource(t *testing.T, cold *coldResumeRig, agent string) {
	t.Helper()
	if agent != "claude" {
		profile := couchcore.LaunchProfile{Agent: agent, Argv: []string{}}
		record, err := cold.c.Threads.CommitStartClaim(cold.parked.Address, cold.parked.Revision, filepath.Join(cold.parked.StartingPath, ".git"), time.Unix(3, 0), couchcore.StartEvent{
			Kind: couchcore.StartClaimed, Nonce: "console-personality", Owner: couchcore.SupervisorOwner{PID: 41, Identity: "fixture-owner"}, Profile: &profile,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range []couchcore.StartEvent{
			{Kind: couchcore.StartHelperRecorded, Nonce: "console-personality", Helper: couchcore.ProcessIdentity{PID: 42, Identity: "pair-helper"}},
			{Kind: couchcore.StartRegistered, Nonce: "console-personality"},
		} {
			record, err = cold.c.Threads.AdvanceStart(record.Address, record.Revision, event)
			if err != nil {
				t.Fatal(err)
			}
		}
		identity := couchcore.ParkIdentity{Nonce: "console-personality-park", Address: record.Address, PID: 42, ProcessIdentity: "pair-helper"}
		record, err = cold.c.Threads.BeginPark(record.Address, record.Revision, identity)
		if err != nil {
			t.Fatal(err)
		}
		cold.parked, err = cold.c.Threads.FinalizePark(context.Background(), record.Address, record.Revision, identity, 1, time.Unix(4, 0))
		if err != nil {
			t.Fatal(err)
		}
	}
	cold.rt.artifacts.SetNativeBinding(cold.parked.Address, agent, sessioninventory.BindingEstablished, "11111111-1111-4111-8111-111111111111")
	cold.c.FreshRegistration = func(context.Context, couchcore.ThreadAddress, string, string) (bool, error) { return true, nil }
	cold.rt.runner.AfterAcknowledge = func(id string) error {
		cold.rt.artifacts.SetPaneSidecar(cold.parked.Address, agent)
		cold.rt.artifacts.SetPairSession(cold.parked.Address, managedChildSession(t, cold.rt.runner, id), true)
		cold.rt.artifacts.SetSessionPresence(cold.parked.Address, couchcore.SessionObservation{State: couchcore.SessionPresent})
		return nil
	}
}

func newEditedRestartConsoleAcceptanceRig(t *testing.T, agent string, configure ...func(*coldResumeRig)) (*coldResumeRig, *serviceRig, couchmessage.Binding, *hubRuntime) {
	t.Helper()
	cold := newColdResumeRig(t, func(rig *coldResumeRig) {
		editedRestartConsoleSource(t, rig, agent)
		for _, setup := range configure {
			setup(rig)
		}
	})
	world := newMessageWorld()
	world.hub = "/repo0"
	service := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
	service.slotOps = newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: cold.parked.StartingPath, Dir: "repo"}}, nil
	}, time.Now)
	hubs := guardHubState(t)
	workspace := cold.c.Slots.(couchcore.SlotWorkspaceResolver).ResolveWorkspace
	service.decorateAuthority = func(a messageAuthority) messageAuthority {
		hubWorkspace, hubBranch := a.workspace, a.branch
		a.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
			if path == world.hub {
				return hubWorkspace(ctx, path)
			}
			return workspace(ctx, path)
		}
		a.record = cold.c.Threads.PreviewThreadContext
		a.branch = func(ctx context.Context, path string) (couchcore.SlotGitStatus, error) {
			if path == world.hub {
				return hubBranch(ctx, path)
			}
			return couchcore.SlotGitStatus{}, nil
		}
		a.hubs = hubs
		return a
	}
	service.init()
	hub := service.connect(0)
	guard := &switchServiceGuard{}
	cold.c.SwitchGuard = guard
	service.s.installAgentSwitchService(cold.console, cold.c, guard)
	return cold, service, hub, hubs
}

// An independent observer is admitted at the final pre-Run fixture seam. Join
// its first store read there, making observation-before-source deterministic
// even when Console's actionable inventory provider is held. Then verify the
// real continuation worker observes that same complete source after Run starts.
func TestEditedRestartConsoleSourcePrecedesObservers(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			type observation struct {
				record  couchcore.ThreadRecord
				binding couchcore.NativeBindingResolution
				err     error
			}
			observe := func(cold *coldResumeRig) observation {
				record, err := cold.c.Threads.GetThread(cold.parked.Address)
				if err != nil {
					return observation{record: record, err: err}
				}
				binding, err := cold.rt.artifacts.ResolveEstablished(context.Background(), record.Address.RepoScope, string(record.Address.Tag), agent)
				return observation{record: record, binding: binding, err: err}
			}
			assertComplete := func(observed observation) {
				t.Helper()
				if observed.err != nil || observed.record.LatestLaunchProfile == nil || observed.record.LatestLaunchProfile.Agent != agent || observed.record.Park != nil || observed.record.VerifiedPark == nil || observed.binding.NativeID != "11111111-1111-4111-8111-111111111111" {
					t.Fatalf("observer admitted before %s source setup completed: profile=%+v binding=%+v park=%+v verified=%+v err=%v", agent, observed.record.LatestLaunchProfile, observed.binding, observed.record.Park, observed.record.VerifiedPark, observed.err)
				}
			}
			worker := make(chan observation, 1)
			newEditedRestartConsoleAcceptanceRig(t, agent, func(cold *coldResumeRig) {
				first := make(chan observation, 1)
				go func() {
					first <- observe(cold)
				}()
				select {
				case observed := <-first:
					assertComplete(observed)
				case <-testwait.After(t):
					t.Fatal("independent source observer did not finish")
				}
				cold.console.SetContinuationProvider(func(context.Context, []couchcore.ThreadAddress) ([]couchcore.ContinuationStatus, error) {
					observed := observe(cold)
					select {
					case worker <- observed:
					default:
					}
					return nil, observed.err
				})
			})
			select {
			case observed := <-worker:
				assertComplete(observed)
			case <-testwait.After(t):
				t.Fatal("Console continuation observer did not run")
			}
		})
	}
}

func TestEditedRestartConsoleRemoteAdoption(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, mode := range []string{"resume", "fresh"} {
			for _, adoption := range []string{"adopted", "attach-refused", "publication-refused"} {
				t.Run(agent+"/"+mode+"/"+adoption, func(t *testing.T) {
					cold, service, hub, hubs := newEditedRestartConsoleAcceptanceRig(t, agent)
					// Keep the complete combined notice visible in this fake host,
					// including the storage diagnostic's private temporary path.
					cold.host.SetSize(ptychild.Size{Rows: 24, Cols: 512})
					// Admission creates the private accepted anchor while found; the
					// real queued operation must honor subsequent positive absence.
					prepare := service.s.switches.Load().prepare
					service.s.switches.Load().prepare = func(ctx context.Context, address couchcore.ThreadAddress, target string, argv *[]string) (couchcore.PreparedAgentSwitch, error) {
						p, err := prepare(ctx, address, target, argv)
						if err == nil && mode == "fresh" {
							cold.rt.artifacts.SetNativeBinding(address, agent, sessioninventory.BindingUnbound, "")
						}
						return p, err
					}
					entered := make(chan couchcore.StartResult, 1)
					completed := make(chan couchcore.SwitchAgentResult, 1)
					failed := make(chan error, 1)
					release := make(chan struct{})
					var once sync.Once
					unblock := func() { once.Do(func() { close(release) }) }
					t.Cleanup(unblock)
					actual := cold.console.Ops()
					cold.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
						if call.Name == "attach" {
							start, ok := call.TypedPayload.(couchcore.StartResult)
							if !ok {
								return nil, errors.New("attach lost StartedChild payload")
							}
							entered <- start
							select {
							case <-release:
							case <-call.Context.Done():
								return nil, call.Context.Err()
							}
							if adoption == "attach-refused" {
								start.Record.PID++ // Existing actual-attach identity refusal seam.
								call.TypedPayload = start
							}
						}
						// The shared cold console began without multiple hubs. Use the
						// production effect gate with the service's exact hub owner.
						value, err := dispatchHubOperation(hubs, call, actual)
						if call.Name == "switch-agent" && err != nil {
							failed <- err
						}
						if result, ok := value.(couchcore.SwitchAgentResult); ok && call.Name == "switch-agent" {
							completed <- result
						}
						return value, err
					})
					argv := []string{"--model", "a b", "", "$HOME", "$(literal)"}
					request := switchRequest(hub, "edited-console", cold.address)
					request.Agent = agent
					var err error
					request.AgentArgs, err = couchmessage.NewSwitchArguments(argv)
					if err != nil {
						t.Fatal(err)
					}
					response := switchCall(t, service, request)
					if response.Code != "accepted" || response.Operation == nil {
						t.Fatalf("actual remote admission: %+v", response)
					}
					statusRequest := callerRequest(hub, "operation-status")
					statusRequest.ID = request.ID
					var start couchcore.StartResult
					testwait.Until(t, "edited restart reaches final attach", func() bool {
						select {
						case start = <-entered:
							return true
						case err := <-failed:
							t.Fatalf("actual edited restart dispatcher failed: %v", err)
						default:
						}
						status := switchCall(t, service, statusRequest)
						if status.Operation != nil && status.Operation.Status.Terminal() {
							t.Fatalf("queued restart settled before adoption: %+v", status.Operation)
						}
						return false
					})
					path := filepath.Join(cold.c.Namespace.Dir(), "threadstore", "repo-preferences.json")
					if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) || start.Configuration == nil {
						t.Fatalf("publication preceded final attach or acceptance lost: %v config=%v", err, start.Configuration != nil)
					}
					held := switchCall(t, service, statusRequest)
					if held.Operation == nil || held.Operation.Status != couchmessage.ReceiptRunning {
						t.Fatalf("receipt settled before final adoption: %+v", held)
					}
					var profile launcher.TrustedLaunchProfile
					for _, entry := range cold.rt.runner.Child(start.Handle.ID()).Env {
						if raw, ok := strings.CutPrefix(entry, launcher.CouchLaunchProfileEnv+"="); ok {
							if err := json.Unmarshal([]byte(raw), &profile); err != nil {
								t.Fatal(err)
							}
						}
					}
					if profile.Agent != agent || !reflect.DeepEqual(profile.Argv, argv) || profile.ResumeRequired != (mode == "resume") || profile.FreshRequired != (mode == "fresh") || mode == "resume" && profile.RequiredSessionID != "11111111-1111-4111-8111-111111111111" {
						t.Fatalf("actual dispatcher lost accepted argv/native intent: %+v", profile)
					}
					if adoption == "publication-refused" {
						if err := os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
					}
					unblock()
					var receipt couchmessage.OperationReceipt
					testwait.Until(t, "edited restart final receipt", func() bool {
						status := switchCall(t, service, statusRequest)
						if status.Operation == nil {
							return false
						}
						receipt = *status.Operation
						return receipt.Status.Terminal()
					})
					if adoption == "attach-refused" {
						if receipt.Status != couchmessage.ReceiptFailed {
							t.Fatalf("actual failed attach succeeded: %+v", receipt)
						}
						if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
							t.Fatalf("failed attach published parameters: %v", err)
						}
						return
					}
					if receipt.Status != couchmessage.ReceiptSucceeded || receipt.Tag != string(cold.parked.Address.Tag) || !start.Handle.Alive() {
						t.Fatalf("adopted restart failed or child stopped: %+v alive=%t", receipt, start.Handle.Alive())
					}
					if strings.Contains(receipt.Warning, couchcore.EditedRestartFallbackNotice) != (mode == "fresh") || strings.Contains(receipt.Warning, "startup parameters were not remembered") != (adoption == "publication-refused") {
						t.Fatalf("remote receipt lost combined notices: %+v", receipt)
					}
					if receipt.Warning != "" {
						testwait.Until(t, "visible edited restart notices", func() bool {
							written := cold.host.Written()
							return (mode != "fresh" || strings.Contains(written, couchcore.EditedRestartFallbackNotice)) && (adoption != "publication-refused" || strings.Contains(written, "startup parameters were not remembered"))
						})
					}
					result := <-completed
					if result.Warning() != receipt.Warning {
						t.Fatalf("original result/receipt warnings diverged: %q / %q", result.Warning(), receipt.Warning)
					}
					if adoption == "adopted" {
						raw, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						doc, err := couchcore.DecodeRepoPreferences(raw)
						if err != nil {
							t.Fatal(err)
						}
						entry := doc.Repositories[filepath.Join(cold.parked.StartingPath, ".git")].Agents[agent]
						if !entry.Confirmed || !reflect.DeepEqual(entry.Argv, argv) {
							t.Fatalf("final attachment failed to publish accepted argv: %s", raw)
						}
					}
				})
			}
		}
	}
}
