package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type recoveryExecutionFixture struct {
	*testEnv
	record              ThreadRecord
	observation         RecoveryObservation
	preserved, prepared int
}

func newRecoveryExecutionFixture(t *testing.T) *recoveryExecutionFixture {
	t.Helper()
	local := testLocalThreadStore(t)
	env := newTestEnv(t, local.slot.WorktreeRoot)
	env.Couch.Threads = local
	env.Git.replies[GitCall{Dir: local.slot.WorktreeRoot, Args: "rev-parse --git-common-dir"}] = local.slot.RepoIdentity
	record := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-1111111111111111")
	record.Reservation = false
	record.CreatedAt, record.LastActiveAt = env.Now, env.Now
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--model", "saved-model"}}
	record.LatestLaunchProfile = &profile
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "old-helper", State: IncarnationLive, LaunchProfile: &profile}}
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	_, observation, _ := recoveryPreviewFixture()
	observation.Capture.CapturedAt = env.Now.Add(-time.Second)
	observation.Ranked[0].Candidate.CWD = record.WorkingPath
	observation.Ranked[0].Candidate.SemanticLast = env.Now.Add(-time.Second)
	f := &recoveryExecutionFixture{testEnv: env, record: created, observation: observation}
	env.Artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionAbsent})
	env.Artifacts.SetPairSession(created.Address, "old-session", false)
	env.Couch.RecoveryEvidence = func(context.Context, ThreadRecord) (RecoveryObservation, error) { return f.observation, nil }
	env.Couch.RecoveryPreserve = func(_ context.Context, _ ThreadRecord, o RecoveryObservation) (RecoveryObservation, error) {
		f.preserved++
		return o, nil
	}
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		f.prepared++
		return slotReadyResult(local), nil
	})
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPaneSidecar(created.Address, "codex")
		env.Artifacts.SetPairSession(created.Address, continuationChildSession(t, env.Runner, id), true)
		env.Artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionPresent, Terminal: &launcher.TerminalSafety{State: launcher.TerminalUsable}})
		return nil
	}
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	return f
}

func (f *recoveryExecutionFixture) preview(t *testing.T) RecoveryPreview {
	t.Helper()
	p, err := f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Address: f.record.Address})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Candidates) != 1 || p.Fresh == nil {
		t.Fatalf("preview %+v", p)
	}
	return p
}

func TestResumeCandidateRefusesWithoutEffects(t *testing.T) {
	for _, mode := range []string{"invalid-key", "stale-key", "unknown-owner", "live-owner", "unknown-terminal"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			key := f.preview(t).Candidates[0].Key
			switch mode {
			case "invalid-key":
				key = "not-a-key"
			case "stale-key":
				f.observation.Capture.RawIdentity.Inode++
			case "unknown-owner":
				f.Proc.SetUnknown(42)
			case "live-owner":
				f.Proc.Set(42, "old-helper")
			case "unknown-terminal":
				f.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("session inspection failed") }
			}
			before, err := f.Couch.Threads.GetThread(f.record.Address)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
			if err == nil {
				t.Fatal("unsafe selection accepted")
			}
			after, readErr := f.Couch.Threads.GetThread(f.record.Address)
			if readErr != nil || !reflect.DeepEqual(before, after) || len(f.Runner.Ops) != 0 || f.preserved != 0 || f.prepared != 0 {
				t.Fatalf("refusal changed state: err=%v record=%+v operations=%v preserve=%d prepare=%d", err, after, f.Runner.Ops, f.preserved, f.prepared)
			}
		})
	}
}

func TestResumeCandidateLaunchKeepsExactSelectionAndSavedConfiguration(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "fresh"}[fresh], func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			preview := f.preview(t)
			choice := preview.Candidates[0]
			if fresh {
				choice = *preview.Fresh
			}
			preference := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: f.Couch.Threads.slot.RepoIdentity, PhysicalPath: f.record.StartingPath, LastAgent: "claude", ArgvByAgent: map[string][]string{"claude": {"--model", "different-default"}}, Revision: 1}
			if err := writePathLaunchPreferenceForTest(f.Couch.Threads, preference); err != nil {
				t.Fatal(err)
			}
			before, found, err := f.Couch.Threads.GetPathLaunchPreference(preference.RepoIdentity, preference.PhysicalPath)
			if err != nil || !found {
				t.Fatalf("preference %v %v", found, err)
			}
			f.Runner.BeforeAcknowledge = func(id string) error {
				current, err := f.Couch.Threads.GetThread(f.record.Address)
				if err != nil {
					t.Fatal(err)
				}
				foundIntent := false
				for _, inc := range current.Incarnations {
					if inc.Start != nil && inc.Start.RecoveryIntent != nil {
						foundIntent = true
						if inc.Start.RecoveryIntent.NativeID != choice.NativeID {
							t.Fatal("persisted different native ID")
						}
					}
				}
				if !foundIntent {
					t.Fatal("released helper without durable recovery intent")
				}
				return nil
			}
			result, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, choice.Key)
			if err != nil {
				t.Fatal(err)
			}
			started, ok := result.Started()
			if !ok {
				t.Fatal("no started child")
			}
			child := f.Runner.Child(started.Handle.ID())
			var profile launcher.TrustedLaunchProfile
			if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
				t.Fatal(err)
			}
			if profile.Tag != string(f.record.Address.Tag) || profile.Agent != "codex" || !reflect.DeepEqual(profile.Argv, f.record.LatestLaunchProfile.Argv) || profile.Recovery == nil || profile.Recovery.Intent.NativeID != choice.NativeID {
				t.Fatalf("launch drift: %+v", profile)
			}
			if profile.Orientation == nil || profile.Orientation.Tag != string(f.record.Address.Tag) || !strings.Contains(profile.Orientation.Body, f.observation.Capture.Raw) {
				t.Fatalf("missing original screen orientation: %+v", profile.Orientation)
			}
			if fresh != (profile.FreshNonce != "") {
				t.Fatalf("fresh nonce %q", profile.FreshNonce)
			}
			after, found, err := f.Couch.Threads.GetPathLaunchPreference(preference.RepoIdentity, preference.PhysicalPath)
			if err != nil || !found || !reflect.DeepEqual(before, after) {
				t.Fatalf("selection changed defaults: %+v -> %+v (%v)", before, after, err)
			}
		})
	}
}

func TestResumeCandidateRechecksEvidenceAfterWorkspacePreparation(t *testing.T) {
	for name, mutate := range map[string]func(*RecoveryObservation){
		"candidate": func(o *RecoveryObservation) { o.Ranked[0].Candidate.Entry.MutationToken = "changed-during-compile" },
		"raw":       func(o *RecoveryObservation) { o.Capture.RawIdentity.Inode++ },
		"events":    func(o *RecoveryObservation) { o.Capture.EventsIdentity.Inode++ },
	} {
		t.Run(name, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			key := f.preview(t).Candidates[0].Key
			f.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
				f.prepared++
				mutate(&f.observation)
				return slotReadyResult(f.Couch.Threads), nil
			})
			_, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
			if err == nil || !strings.Contains(err.Error(), "changed") || f.prepared != 1 || f.preserved != 1 || len(f.Runner.Ops) != 0 {
				t.Fatalf("second gate err=%v prepare=%d preserve=%d ops=%v", err, f.prepared, f.preserved, f.Runner.Ops)
			}
			after, readErr := f.Couch.Threads.GetThread(f.record.Address)
			if readErr != nil {
				t.Fatal(readErr)
			}
			for _, inc := range after.Incarnations {
				if inc.Start != nil {
					t.Fatal("failed second gate left pending start claim")
				}
			}
		})
	}
}

func TestResumeCandidateBusyOwnedSourceRefusesBeforePreservation(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	key := f.preview(t).Candidates[0].Key
	f.Proc.Set(42, "old-helper")
	f.Couch.reg = f.Couch.reg.Insert(ActorRecord{ID: "source-actor", Thread: f.record.Address, Args: StartArgs{Worktree: Worktree(f.record.StartingPath)}, PID: 42, Identity: "old-helper"})
	probe := &fakeLiveRestartProbe{busyErr: errors.New("agent busy")}
	f.Couch.LiveRestart = probe
	before, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("busy refusal = %v", err)
	}
	after, readErr := f.Couch.Threads.GetThread(f.record.Address)
	if readErr != nil || !reflect.DeepEqual(before, after) || len(f.Runner.Ops) != 0 || f.preserved != 0 || f.prepared != 0 || f.Proc.Exists(42) != Live {
		t.Fatal("busy source was changed")
	}
	if len(probe.busyChecks) != 1 || len(probe.forced) != 1 || probe.forced[0] {
		t.Fatalf("busy guard bypassed: %+v", probe)
	}
}

// Keep filesystem IO fake, but use the production owner to select the capture
// from the durable claim/context instead of always returning the current screen.
func (f *recoveryExecutionFixture) useRetainedCaptureSource(t *testing.T) {
	t.Helper()
	f.Couch.RecoveryEvidence = func(_ context.Context, record ThreadRecord) (RecoveryObservation, error) {
		observation := f.observation
		retained := record.RecoveryContext != nil
		for _, inc := range record.Incarnations {
			retained = retained || (inc.Start != nil && inc.Start.RecoveryIntent != nil)
		}
		if retained {
			capture, err := recoveryCaptureSource(record, artifactpath.Paths{}, "codex")
			if err != nil {
				return RecoveryObservation{}, err
			}
			observation.Capture = capture
		}
		return observation, nil
	}
}

func (f *recoveryExecutionFixture) registerLiveChildren(t *testing.T) {
	t.Helper()
	after := f.Runner.AfterAcknowledge
	f.Runner.AfterAcknowledge = func(id string) error {
		if err := after(id); err != nil {
			return err
		}
		h := &fakeHandle{runner: f.Runner, id: id}
		f.Proc.Set(h.PID(), h.Identity())
		return nil
	}
}

func TestResumeCandidateLiveSourcePersistsArchivedScreenBeforePark(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.useRetainedCaptureSource(t)
	after := f.Runner.AfterAcknowledge
	live := withParkLifecycle(t, f.testEnv, f.record, 42, "old-helper")
	f.Runner.AfterAcknowledge = after
	f.registerLiveChildren(t)
	f.Couch.reg = f.Couch.reg.Insert(ActorRecord{ID: "source-actor", Thread: f.record.Address, Args: StartArgs{Worktree: Worktree(f.record.StartingPath)}, PID: 42, Identity: "old-helper"})
	probe := &fakeLiveRestartProbe{}
	f.Couch.LiveRestart = probe
	key := f.preview(t).Candidates[0].Key
	original := f.observation.Capture
	archived := original
	archived.Raw, archived.Events = "/archive/original.raw", "/archive/original.events"
	archived.RawIdentity.Inode += 10
	archived.EventsIdentity.Inode += 10
	f.Couch.RecoveryPreserve = func(_ context.Context, _ ThreadRecord, observation RecoveryObservation) (RecoveryObservation, error) {
		f.preserved++
		if f.Proc.Exists(42) != Live || observation.Capture != original || live.Lifecycle.lastRequest.Attempt != 0 {
			t.Fatal("original screen not preserved before park")
		}
		observation.Capture = archived
		return observation, nil
	}
	published := 0
	live.Lifecycle.onPublish = func(pairlifecycle.QuitRequest) {
		published++
		record, err := f.Couch.Threads.GetThread(f.record.Address)
		if err != nil {
			t.Fatal(err)
		}
		if record.RecoveryContext == nil || record.RecoveryContext.Capture != archived || record.RecoveryContext.SelectedNativeID != "native" || record.RecoveryContext.SelectedKey == "" {
			t.Fatalf("quit published without retained original selection: %+v", record.RecoveryContext)
		}
		if f.preserved != 1 {
			t.Fatal("park preceded preservation")
		}
	}
	result, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Started(); !ok || published != 1 || len(probe.busyChecks) != 2 || f.Proc.Exists(42) != Dead {
		t.Fatalf("live transition incomplete: result=%+v parks=%d checks=%d", result, published, len(probe.busyChecks))
	}
	record, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.RecoveryContext == nil || record.RecoveryContext.Capture != archived {
		t.Fatal("successful launch dropped original capture")
	}
}

func TestResumeCandidateSecondAlternativeUsesOriginalRetainedScreen(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.useRetainedCaptureSource(t)
	f.registerLiveChildren(t)
	alternative := f.observation.Ranked[0]
	alternative.Candidate.NativeID = "second-native"
	alternative.Candidate.Entry.Artifact.RelativePath = "second.jsonl"
	f.observation.Ranked = append(f.observation.Ranked, alternative)
	p, err := f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Address: f.record.Address})
	if err != nil {
		t.Fatal(err)
	}
	original := f.observation.Capture
	first, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, p.Candidates[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil {
		t.Fatal(err)
	}
	after := f.Runner.AfterAcknowledge
	live := withParkLifecycle(t, f.testEnv, current, first.Start.Record.PID, first.Start.Record.Identity)
	session := recordSessionName(current)
	live.model.SetSession(session, true)
	f.Artifacts.SetPairSession(current.Address, session, true)
	f.Runner.AfterAcknowledge = after
	f.Couch.LiveRestart = &fakeLiveRestartProbe{}
	// The current screen belongs to the first attempted candidate. It must not
	// replace the retained screen that the operator originally selected from.
	f.observation.Capture.Raw = "/current/first-attempt.raw"
	f.observation.Capture.Events = "/current/first-attempt.events"
	p, err = f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Address: f.record.Address})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Candidates) != 2 || p.Snapshot.Capture != original {
		t.Fatalf("alternatives forgot original screen: %+v", p)
	}
	second, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, p.Candidates[1].Key)
	if err != nil {
		t.Fatal(err)
	}
	var profile launcher.TrustedLaunchProfile
	child := f.Runner.Child(second.Start.Handle.ID())
	if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Recovery == nil || profile.Recovery.Intent.NativeID != "second-native" || profile.Recovery.Intent.Context.Capture != original || profile.Orientation == nil || !strings.Contains(profile.Orientation.Body, original.Raw) || strings.Contains(profile.Orientation.Body, "first-attempt.raw") {
		t.Fatalf("second selection changed original context: %+v", profile)
	}
}

func TestResumeCandidateOrientationNoticeDoesNotResend(t *testing.T) {
	for _, phase := range []orientation.DeliveryPhase{orientation.DeliveryWaiting, orientation.DeliveryIndeterminate} {
		t.Run(string(phase), func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.registerLiveChildren(t)
			reads := 0
			f.Couch.OrientationStatus = func(_ context.Context, address ThreadAddress, agent, attempt string) (orientation.DeliveryState, error) {
				reads++
				if address != f.record.Address || agent != "codex" || attempt == "" {
					t.Fatal("status read lost exact target")
				}
				return orientation.DeliveryState{Phase: phase, BodyWritten: phase == orientation.DeliveryIndeterminate, Reason: "delivery status evidence"}, nil
			}
			result, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, f.preview(t).Candidates[0].Key)
			if err != nil {
				t.Fatal(err)
			}
			expected := string(phase)
			if expected == "" {
				expected = "pending"
			}
			if _, ok := result.Started(); !ok || result.Orientation.Phase != phase || !strings.Contains(result.Notice, "orientation "+expected) || reads != 1 {
				t.Fatalf("status lost or polled for retry: %+v reads=%d", result, reads)
			}
			child := f.Runner.Child(result.Start.Handle.ID())
			if child.ExecCount != 1 || len(child.Signals) != 0 || len(f.Runner.order) != 1 {
				t.Fatalf("orientation uncertainty restarted child: %+v", child)
			}
		})
	}
}

func TestResumeCandidateRejectsAnotherNativeOwnerAtBothGates(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "preflight", true: "after-prepare"}[late], func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			key := f.preview(t).Candidates[0].Key
			other := ThreadAddress{RepoScope: f.record.Address.RepoScope, Tag: "another-slot"}
			occupy := func() {
				f.Artifacts.SetNativeBinding(other, "codex", sessioninventory.BindingEstablished, "native")
				f.Artifacts.SetPairSession(other, "other-live-terminal", true)
			}
			if late {
				f.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
					f.prepared++
					occupy()
					return slotReadyResult(f.Couch.Threads), nil
				})
			} else {
				occupy()
			}
			_, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
			if err == nil || !strings.Contains(err.Error(), "live wrapper") || len(f.Runner.Ops) != 0 {
				t.Fatalf("duplicate native owner admitted: err=%v ops=%v", err, f.Runner.Ops)
			}
			if !late && (f.preserved != 0 || f.prepared != 0) {
				t.Fatal("first gate ran after effects")
			}
		})
	}
}
