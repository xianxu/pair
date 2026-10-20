package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestRecoveryNativeOnlyReadWithoutScreen(t *testing.T) {
	for _, mode := range []string{"missing", "oversized", "changed-device", "absent-witness", "invalid-witness", "refresh-retained-native"} {
		t.Run(mode, func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, true)
			record.WorkingPath, _ = filepath.EvalSymlinks(record.WorkingPath)
			paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: reader.DataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
			if err != nil {
				t.Fatal(err)
			}
			files, _ := paths.ScrollbackArtifacts("codex")
			native := filepath.Join(reader.HomeDir, ".codex", "sessions", "2026", "10", "15")
			if err := os.MkdirAll(native, 0700); err != nil {
				t.Fatal(err)
			}
			const id = "019d1111-1111-7111-8111-111111111111"
			stamp := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
			raw := fmt.Sprintf("{\"timestamp\":%q,\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"cli\",\"cwd\":%q}}\n{\"timestamp\":%q,\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"retained working context\"}]}}\n", stamp, id, record.WorkingPath, stamp)
			if err := os.WriteFile(filepath.Join(native, "rollout-2026-10-15T00-00-00-"+id+".jsonl"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "refresh-retained-native":
				record.RecoveryContext = &threadrecord.RecoveryContext{Version: 2, Source: &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: time.Now().UTC().Add(-10 * time.Minute), Revision: record.Revision}}
				record.RecoveryIntent = &threadrecord.RecoveryIntent{Context: *record.RecoveryContext}
			case "missing":
				if err := os.Remove(files.Raw); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.Truncate(files.Raw, sessioninventory.RecoveryTotalReadLimit+1); err != nil {
					t.Fatal(err)
				}
			case "changed-device":
				data, err := os.ReadFile(files.Events)
				if err != nil {
					t.Fatal(err)
				}
				witness, err := artifactpath.DecodeCaptureWitness(data)
				if err != nil {
					t.Fatal(err)
				}
				witness.RawFile.Device++ // Matching inode is not file identity continuity.
				data, err = json.Marshal(witness)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(files.Events, append(data, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "absent-witness":
				if err := os.WriteFile(files.Events, nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "invalid-witness":
				if err := os.WriteFile(files.Events, []byte("{\"type\":\"capture_witness\",\"version\":999}\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := reader.Read(context.Background(), record)
			if err != nil || got.Incomplete || len(got.Ranked) != 1 || got.Source == nil || got.Source.Kind != sessioninventory.RecoverySourceNative {
				t.Fatalf("screen failure hid complete native evidence: got=%+v err=%v", got, err)
			}
			if mode != "absent-witness" && mode != "refresh-retained-native" && (got.Screen.Availability != ScreenUnavailable || got.Screen.Reason == "") {
				t.Fatalf("screen failure hidden: %+v", got.Screen)
			}
			if got.BytesRead > sessioninventory.RecoveryTotalReadLimit {
				t.Fatal("screen fallback exceeded shared budget")
			}
		})
	}
}

func TestRecoveryNativeOnlyPreviewUsesFixedSource(t *testing.T) {
	record, observed, now := recoveryPreviewFixture()
	observed.Capture = threadrecord.RecoveryCapture{}
	observed.Generation = ""
	observed.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: now, Revision: record.Revision}
	observed.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
	preview, err := DecideSlotRecovery(record, observed, now)
	if err != nil || len(preview.Candidates) != 1 || preview.Snapshot == nil || preview.Snapshot.Version != 2 || preview.Automatic || preview.Proven != nil {
		t.Fatalf("native-only preview=%+v err=%v", preview, err)
	}
	again, err := DecideSlotRecovery(record, observed, now.Add(time.Minute))
	if err != nil || again.Candidates[0].Key != preview.Candidates[0].Key {
		t.Fatalf("cutoff drift: %+v %v", again, err)
	}
	expired, err := DecideSlotRecovery(record, observed, now.Add(5*time.Minute))
	if err == nil && len(expired.Candidates) > 0 {
		t.Fatal("expired native selection remained actionable")
	}
}

func TestRecoveryNativeOnlyFreshDoesNotMislabelCaptureFileTime(t *testing.T) {
	record, observed, now := recoveryPreviewFixture()
	observed.Capture.LastActivity = now.Add(-time.Minute)
	observed.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: now, Revision: record.Revision}
	observed.Screen = ScreenObservation{Availability: ScreenUnavailable, LastActivity: observed.Capture.LastActivity, ActivitySource: "capture-file activity"}
	observed.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
	preview, err := DecideSlotRecovery(record, observed, now)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Fresh.LastActivity.IsZero() || !preview.Screen.LastActivity.Equal(observed.Capture.LastActivity) {
		t.Fatalf("file time became Fresh conversation activity: %+v", preview)
	}
}

func TestRecoveryNativeOnlySelectionCarriesCutoffAndPreservesNoScreen(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.observation.Capture = threadrecord.RecoveryCapture{}
	f.observation.Generation = ""
	source := &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: f.Now, Revision: f.record.Revision}
	f.observation.Source = source
	f.observation.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
	key := f.preview(t).Candidates[0].Key
	f.Runner.BeforeAcknowledge = func(string) error {
		current, err := f.Couch.Threads.observeSlotCurrent()
		if err != nil || current.Record == nil {
			t.Fatalf("read durable native claim: %+v %v", current, err)
		}
		persisted := current.Record
		if persisted.Transcript != nil || len(persisted.Incarnations) != 1 || persisted.Incarnations[0].Start == nil {
			t.Fatalf("native choice manufactured binding or lost claim: %+v", persisted)
		}
		intent := persisted.Incarnations[0].Start.RecoveryIntent
		if intent == nil || intent.Context.Version != 2 || intent.Context.Source == nil || !intent.Context.Source.Cutoff.Equal(source.Cutoff) || intent.SourceRevision != source.Revision || intent.Context.Capture != (threadrecord.RecoveryCapture{}) {
			t.Fatalf("native evidence not durable before launch: %+v", intent)
		}
		return nil
	}
	f.Couch.RecoveryEvidence = func(_ context.Context, _ ThreadRecord, q RecoveryObservationRequest) (RecoveryObservation, error) {
		if q.Source == nil || !q.Source.Cutoff.Equal(source.Cutoff) {
			return RecoveryObservation{}, fmt.Errorf("selected cutoff not propagated")
		}
		return f.observation, nil
	}
	result, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Started(); !ok || f.preserved != 0 {
		t.Fatalf("native-only launch/preservation: %+v preserves=%d", result, f.preserved)
	}
	if strings.Contains(result.Notice, "tail match") {
		t.Fatalf("native result invented screen evidence: %s", result.Notice)
	}
}

func TestRecoveryObservationRetainsPartialFactsAndRetryBudget(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	calls := 0
	var firstSource *threadrecord.NativeRecoverySource
	f.Couch.RecoveryEvidence = func(_ context.Context, _ ThreadRecord, q RecoveryObservationRequest) (RecoveryObservation, error) {
		calls++
		if calls == 1 {
			firstSource = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: q.ObservedAt.UTC(), Revision: f.record.Revision}
			return RecoveryObservation{Source: firstSource, Agent: "codex", Screen: ScreenObservation{Availability: ScreenUnavailable, Reason: "screen identity changed"}, BytesRead: 100, Incomplete: true, Reasons: []string{"partial native scan"}}, nil
		}
		if q.BytesAlreadyRead != 100 || q.Source == nil || !q.Source.Cutoff.Equal(firstSource.Cutoff) {
			t.Fatalf("retry reset evidence/budget: %+v", q)
		}
		return RecoveryObservation{BytesRead: 150, Incomplete: true}, fmt.Errorf("second scan refused")
	}
	observed := f.Couch.recoveryObservation(context.Background(), f.record)
	if calls != 2 || observed.BytesRead != 150 || observed.Screen.Reason != "screen identity changed" || observed.Source == nil || !observed.Incomplete {
		t.Fatalf("partial facts lost: %+v calls=%d", observed, calls)
	}
}

func TestRecoveryNativeOnlyInterruptedRetirementKeepsSnapshot(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.observation.Capture = threadrecord.RecoveryCapture{}
	f.observation.Generation = ""
	f.observation.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: f.Now, Revision: f.record.Revision}
	f.observation.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
	key := f.preview(t).Candidates[0].Key
	interrupted := errors.New("retirement interrupted")
	retired := 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Digest: "exact inactive terminal", Retire: func(context.Context) error {
			retired++
			current, err := f.Couch.Threads.GetThread(f.record.Address)
			if err != nil {
				return err
			}
			if current.RecoveryContext == nil || current.RecoveryContext.Source == nil || current.RecoveryContext.SelectedKey != key || !current.RecoveryContext.Source.Cutoff.Equal(f.Now) {
				return errors.New("native evidence was not persisted before retirement")
			}
			return interrupted
		}}, nil
	}
	_, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
	if !errors.Is(err, interrupted) || retired != 1 || len(f.Runner.Ops) != 0 || f.preserved != 0 {
		t.Fatalf("interruption lost order: err=%v retired=%d ops=%v preserved=%d", err, retired, f.Runner.Ops, f.preserved)
	}
	reopened, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil || reopened.RecoveryContext == nil || reopened.RecoveryContext.Source == nil || reopened.RecoveryContext.SelectedKey != key {
		t.Fatalf("retained snapshot lost on reopen: %+v %v", reopened, err)
	}
}

func TestRecoveryNativeOnlyStaleChoiceHasNoEffects(t *testing.T) {
	for _, kind := range []string{"expired", "artifact", "revision", "copied"} {
		t.Run(kind, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.observation.Capture = threadrecord.RecoveryCapture{}
			f.observation.Generation = ""
			f.observation.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: f.Now, Revision: f.record.Revision}
			f.observation.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
			key := f.preview(t).Candidates[0].Key
			switch kind {
			case "expired":
				f.Couch.Clock = FixedClock{T: f.Now.Add(threadrecord.NativeRecoverySelectionLifetime)}
			case "artifact":
				f.observation.Ranked[0].Candidate.Entry.MutationToken = "changed"
			case "revision":
				f.observation.Source.Revision++
			case "copied":
				selector, err := decodeRecoveryCandidateKey(key)
				if err != nil {
					t.Fatal(err)
				}
				selector.Address.Tag = "other-tag"
				key, err = EncodeRecoveryCandidateKey(selector)
				if err != nil {
					t.Fatal(err)
				}
			}
			before, err := f.Couch.Threads.GetThread(f.record.Address)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
			after, readErr := f.Couch.Threads.GetThread(f.record.Address)
			if err == nil || readErr != nil || !reflect.DeepEqual(before, after) || f.prepared != 0 || f.preserved != 0 || len(f.Runner.Ops) != 0 {
				t.Fatalf("stale choice acted: err=%v read=%v prepare=%d preserve=%d ops=%v", err, readErr, f.prepared, f.preserved, f.Runner.Ops)
			}
		})
	}
}

func TestRecoveryNativeOnlyFreshKeepsReadableScreen(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.observation.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: f.Now, Revision: f.record.Revision}
	f.observation.Screen = ScreenObservation{Availability: ScreenAvailable}
	f.observation.Ranked[0].Score = sessioninventory.RecoveryTailScore{}
	result, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Started(); !ok || f.preserved != 1 {
		t.Fatalf("Fresh discarded readable orientation: %+v preserves=%d", result, f.preserved)
	}
}
