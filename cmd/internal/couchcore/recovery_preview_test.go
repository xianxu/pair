package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func recoveryPreviewFixture() (ThreadRecord, RecoveryObservation, time.Time) {
	now := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	record := ThreadRecord{Address: ThreadAddress{RepoScope: "abcdef1234567890", Tag: "1-pair-42"}, Revision: 3, WorkingPath: "/repo", LatestLaunchProfile: &LaunchProfile{Agent: "codex"}}
	id := artifactpath.CaptureFileIdentity{Device: 1, Inode: 1, Size: 50, ModifiedAt: now.Add(-time.Minute)}
	events := id
	events.Inode = 2
	observation := RecoveryObservation{Capture: threadrecord.RecoveryCapture{Raw: "/captures/a.raw", Events: "/captures/a.events", RawIdentity: id, EventsIdentity: events, CapturedAt: now.Add(-time.Minute), LaunchOrdinal: 1, Nonce: "launch"}, Generation: "generation", Agent: "codex"}
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "native.jsonl"}, StableFileID: "file", MutationToken: "mutation", Size: 100, ModTime: &now}
	candidate := sessioninventory.RecoveryCandidate{Agent: sessioninventory.AgentCodex, NativeID: "native", Entry: entry, ParserVersion: sessioninventory.RecoveryParserVersion, RootEligible: true, Interactive: sessioninventory.InteractiveEvidence{State: sessioninventory.InteractiveYes}, CWD: "/repo", SemanticLast: now.Add(-time.Minute)}
	observation.Ranked = []sessioninventory.RankedRecoveryCandidate{{Candidate: candidate, Score: sessioninventory.RecoveryTailScore{Matched: 20, Total: 20, Percent: 100, Excerpt: "where we were"}, RequiresExplicitSelection: true}}
	return record, observation, now
}

func TestRecoveryPreviewUniquePerfectMatchRemainsExplicit(t *testing.T) {
	record, observation, now := recoveryPreviewFixture()
	p, err := DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Automatic || len(p.Candidates) != 1 || p.Candidates[0].Label != "resume (best candidate)" || p.Fresh == nil {
		t.Fatalf("bad explicit preview: %+v", p)
	}
	if p.Candidates[0].Key == p.Fresh.Key || p.Candidates[0].TailMatch != 100 {
		t.Fatal("lost selection evidence")
	}
	second, err := DecideSlotRecovery(record, observation, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if p.Candidates[0].Key != second.Candidates[0].Key {
		t.Fatal("clock-only preview change invalidated stable key")
	}
	revised := record
	revised.Revision++
	changedRevision, err := DecideSlotRecovery(revised, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if changedRevision.Candidates[0].Key == p.Candidates[0].Key {
		t.Fatal("changed record retained selection key")
	}
	observation.Capture.RawIdentity.Inode++
	changed, err := DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Candidates[0].Key == p.Candidates[0].Key {
		t.Fatal("replacement capture retained key")
	}
}

func TestRecoveryPreviewBoundsAlternativesAndRefusesIncomplete(t *testing.T) {
	record, observation, now := recoveryPreviewFixture()
	base := observation.Ranked[0]
	for _, id := range []string{"two", "three", "four", "five", "six"} {
		c := base
		c.Candidate.NativeID = id
		observation.Ranked = append(observation.Ranked, c)
	}
	p, err := DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Candidates) != 5 || !p.Truncated || p.Candidates[4].Label != "resume (option 4)" {
		t.Fatalf("bad alternatives %+v", p)
	}
	observation.Incomplete = true
	observation.Reasons = []string{"unreadable transcript"}
	p, err = DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Incomplete || p.Fresh == nil || p.Fresh.Key != "" || len(p.Candidates) != 0 {
		t.Fatalf("incomplete scan hid Fresh or granted native authority %+v", p)
	}
}

func TestRecoveryPreviewExpiredEvidenceCannotSelect(t *testing.T) {
	record, observation, now := recoveryPreviewFixture()
	preview, err := DecideSlotRecovery(record, observation, now.AddDate(2, 0, 0))
	if err != nil || len(preview.Candidates) != 0 || preview.Fresh == nil || preview.Fresh.Key != "" {
		t.Fatalf("expired screen must offer only Fresh without native authority: %+v %v", preview, err)
	}
}

func TestRecoveryPreviewFreshUsesOriginalActivity(t *testing.T) {
	record, observation, now := recoveryPreviewFixture()
	for _, original := range []time.Time{now.Add(-time.Hour), {}} {
		observation.Capture.LastActivity = original
		p, err := DecideSlotRecovery(record, observation, now)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Fresh.LastActivity.Equal(original) {
			t.Fatalf("fresh activity = %v; original = %v", p.Fresh.LastActivity, original)
		}
	}
}

func TestFreshProjectionSurvivesIncompleteOrMissingScreen(t *testing.T) {
	for _, kind := range []string{"incomplete", "missing", "expired"} {
		t.Run(kind, func(t *testing.T) {
			record, observed, now := recoveryPreviewFixture()
			switch kind {
			case "incomplete":
				observed.Incomplete = true
			case "missing":
				observed.Capture = threadrecord.RecoveryCapture{}
			case "expired":
				now = now.AddDate(2, 0, 0)
			}
			preview, _ := DecideSlotRecovery(record, observed, now)
			if preview.Fresh == nil || preview.Fresh.Label == "" {
				t.Fatalf("fresh hidden: %+v", preview)
			}
			if preview.Fresh.Key != "" {
				t.Fatal("projection manufactured execution authority for missing evidence")
			}
		})
	}
}

func TestResumeCandidatesReadFailureStillProjectsFresh(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	calls := 0
	f.Couch.RecoveryEvidence = func(context.Context, ThreadRecord, RecoveryObservationRequest) (RecoveryObservation, error) {
		calls++
		return RecoveryObservation{}, errors.New("unreadable capture")
	}
	preview, err := f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Address: f.record.Address})
	if err != nil || preview.Fresh == nil || len(preview.Candidates) != 0 || calls != 2 {
		t.Fatalf("failed observation hid recovery: preview=%+v err=%v calls=%d", preview, err, calls)
	}
}
