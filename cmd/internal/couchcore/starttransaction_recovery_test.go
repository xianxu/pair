package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"reflect"
	"strings"
	"testing"
	"time"
)

func recoveryStartIntent(record ThreadRecord) *threadrecord.RecoveryIntent {
	at := record.CreatedAt
	identity := artifactpath.CaptureFileIdentity{Size: 10, ModifiedAt: at, Device: 1, Inode: 2}
	return &threadrecord.RecoveryIntent{SelectedKey: "fresh-key", Context: threadrecord.RecoveryContext{Version: 1, Address: toPersistedThreadAddress(record.Address), Capture: threadrecord.RecoveryCapture{Raw: "/capture/raw", Events: "/capture/events", RawIdentity: identity, EventsIdentity: identity, CapturedAt: at}, Generation: "generation", Digest: strings.Repeat("a", 64), CreatedAt: at, ExpiresAt: at.Add(time.Hour), SelectedKey: "fresh-key", Candidates: []threadrecord.RecoveryCandidate{{Key: "native-key", Agent: "codex", NativeID: "native", ArtifactRoot: "codex", ArtifactRelativePath: "rollout.jsonl", StableFileID: "file", GenerationToken: "birth", MutationToken: "mtime", Size: 10, ModTime: at, LastActivity: at, Eligible: true}}}}
}
func TestStartRecoveryIntentTransfersAtBothCompletionTransitions(t *testing.T) {
	for _, kind := range []StartEventKind{StartRegistered, StartRecoveredUnknown} {
		t.Run(string(kind), func(t *testing.T) {
			record := admittedStartRecord(t)
			intent := recoveryStartIntent(record)
			record, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "nonce", Owner: SupervisorOwner{PID: 1, Identity: "owner"}, RecoveryIntent: intent})
			if err != nil {
				t.Fatal(err)
			}
			intent.Context.Candidates[0].Excerpt = "mutated caller"
			if record.Incarnations[0].Start.RecoveryIntent.Context.Candidates[0].Excerpt != "" {
				t.Fatal("claim aliases caller")
			}
			record, err = AdvanceStartTransaction(record, StartEvent{Kind: StartHelperRecorded, Nonce: "nonce", Helper: ProcessIdentity{PID: 2, Identity: "helper"}})
			if err != nil {
				t.Fatal(err)
			}
			// Roundtrip models the persisted recovery claim surviving a restart.
			record = fromPersistedThreadRecord(toPersistedThreadRecord(record))
			next, err := AdvanceStartTransaction(record, StartEvent{Kind: kind, Nonce: "nonce"})
			if err != nil {
				t.Fatal(err)
			}
			if next.Incarnations[0].Start != nil || next.RecoveryContext == nil || next.RecoveryContext.SelectedKey != "fresh-key" {
				t.Fatalf("lost intent: %+v", next)
			}
			if next.VerifiedPark != nil {
				t.Fatal("source absence invented park proof")
			}
			want := IncarnationLive
			if kind == StartRecoveredUnknown {
				want = IncarnationUnknown
			}
			if next.Incarnations[0].State != want {
				t.Fatal("unknown became success")
			}
			next.RecoveryContext.Candidates[0].Excerpt = "changed output"
			if record.Incarnations[0].Start.RecoveryIntent.Context.Candidates[0].Excerpt != "" {
				t.Fatal("transfer aliases old record")
			}
		})
	}
}
func TestStartRecoveryContextSupersessionAndAlternative(t *testing.T) {
	for _, alternative := range []bool{false, true} {
		record := admittedStartRecord(t)
		intent := recoveryStartIntent(record)
		record.RecoveryContext = threadrecord.CloneRecoveryContext(&intent.Context)
		event := StartEvent{Kind: StartClaimed, Nonce: "nonce", Owner: SupervisorOwner{PID: 1, Identity: "owner"}}
		if alternative {
			intent.SelectedKey = "native-key"
			intent.NativeID = "native"
			event.RecoveryIntent = intent
		}
		claimed, err := AdvanceStartTransaction(record, event)
		if err != nil {
			t.Fatal(err)
		}
		if claimed.RecoveryContext != nil {
			t.Fatal("old context not superseded by claim")
		}
		if alternative && claimed.Incarnations[0].Start.RecoveryIntent.Context.Capture != record.RecoveryContext.Capture {
			t.Fatal("alternative changed original screen")
		}
		if !alternative && claimed.Incarnations[0].Start.RecoveryIntent != nil {
			t.Fatal("ordinary start gained recovery")
		}
	}
}
func TestRecoveryConversionAndCloneDoNotAlias(t *testing.T) {
	record := admittedStartRecord(t)
	intent := recoveryStartIntent(record)
	record.RecoveryContext = &intent.Context
	before := cloneThreadRecord(record)
	for _, copy := range []ThreadRecord{cloneThreadRecord(record), fromPersistedThreadRecord(toPersistedThreadRecord(record))} {
		copy.RecoveryContext.Candidates[0].Excerpt = "changed"
	}
	if !reflect.DeepEqual(record, before) {
		t.Fatal("conversion or clone shares snapshot")
	}
}

func TestStartRecoveryRejectsExpiredOrPrematureIntent(t *testing.T) {
	for _, expired := range []bool{false, true} {
		record := admittedStartRecord(t)
		intent := recoveryStartIntent(record)
		if expired {
			record.Incarnations[0].StartedAt = intent.Context.ExpiresAt
		} else {
			record.Incarnations[0].StartedAt = intent.Context.CreatedAt.Add(-time.Second)
		}
		before := cloneThreadRecord(record)
		_, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "nonce", Owner: SupervisorOwner{PID: 1, Identity: "owner"}, RecoveryIntent: intent})
		if err == nil {
			t.Fatal("accepted out-of-lifetime recovery")
		}
		if !reflect.DeepEqual(record, before) {
			t.Fatal("failed claim mutated input")
		}
	}
}

func TestDeleteStartRetainsRecoveryContext(t *testing.T) {
	for _, parked := range []bool{false, true} {
		name := "previously-started"
		if parked {
			name = "verified-park"
		}
		t.Run(name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			record := admittedStartRecord(t)
			record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
			record.LatestLaunchProfile = &LaunchProfile{Agent: "codex"}
			intent := recoveryStartIntent(record)
			intent.SelectedKey, intent.NativeID = "native-key", "native"
			record, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			record, err = store.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartClaimed, Nonce: "start-0123456789abcdef", Owner: SupervisorOwner{PID: 41, Identity: "owner"}, RecoveryIntent: intent})
			if err != nil {
				t.Fatal(err)
			}
			if parked {
				record, err = store.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
					revision := next.Revision
					markActionableParked(next, next.CreatedAt)
					next.Revision = revision
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = store.DeleteStart(record.Address, record.Revision, "start-0123456789abcdef"); err != nil {
				t.Fatal(err)
			}
			got, err := store.GetThread(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			want := threadrecord.CloneRecoveryContext(&intent.Context)
			want.SelectedKey, want.SelectedNativeID = intent.SelectedKey, intent.NativeID
			if !reflect.DeepEqual(got.RecoveryContext, want) {
				t.Fatalf("rollback lost selected recovery context: got %+v; want %+v", got.RecoveryContext, want)
			}
			if len(got.Incarnations) != 0 || (got.VerifiedPark != nil) != parked {
				t.Fatal("rollback changed park truth or retained claim")
			}
		})
	}
}
