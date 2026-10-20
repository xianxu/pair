package sessioninventory

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func nativeCollectionFixture() (*recoveryFake, RecoveryCollectRequest) {
	f, q := recoveryCollectionFixture()
	q.SourceKind = RecoverySourceNative
	q.ObservationCutoff = q.CapturedAt
	q.ProducingLaunch, q.CapturedAt, q.RenderedTail, q.TTYActivity = nil, nil, nil, nil
	return f, q
}

func TestRecoveryNativeOnlyCollectorNoCaptureOrLaunchHint(t *testing.T) {
	f, q := nativeCollectionFixture()
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if got.Incomplete || len(got.Ranked) != 1 {
		t.Fatalf("native-only evidence=%+v", got)
	}
	r := got.Ranked[0]
	if r.Score != (RecoveryTailScore{}) || r.ActivityDelta != nil || !r.RequiresExplicitSelection {
		t.Fatalf("fabricated screen authority: %+v", r)
	}
	if r.Candidate.CWD != "/repo" {
		t.Fatalf("physical cwd=%q", r.Candidate.CWD)
	}
}

func TestRecoveryNativeOnlyCollectorRejectsMixedOrMissingSource(t *testing.T) {
	for _, kind := range []string{"missing-cutoff", "zero-cutoff", "capture", "tail", "tty", "future-launch", "zero-launch", "unknown-kind", "capture-cutoff"} {
		t.Run(kind, func(t *testing.T) {
			f, q := nativeCollectionFixture()
			at := *q.ObservationCutoff
			switch kind {
			case "missing-cutoff":
				q.ObservationCutoff = nil
			case "zero-cutoff":
				q.ObservationCutoff = &time.Time{}
			case "capture":
				q.CapturedAt = &at
			case "tail":
				q.RenderedTail = []string{recoveryTestText}
			case "tty":
				q.TTYActivity = &at
			case "future-launch":
				at = at.Add(time.Minute)
				q.ProducingLaunch = &at
			case "zero-launch":
				q.ProducingLaunch = &time.Time{}
			case "unknown-kind":
				q.SourceKind = "unknown"
			case "capture-cutoff":
				q.SourceKind = RecoverySourceCapture
			}
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if !got.Incomplete || len(got.Candidates) != 0 || len(f.calls) != 0 {
				t.Fatalf("invalid source scanned: %+v calls=%v", got, f.calls)
			}
		})
	}
}

func TestRecoveryNativeOnlyCollectorCutoffAndHint(t *testing.T) {
	for _, kind := range []string{"future-message", "future-created", "old-hint", "other-cwd", "exec", "child"} {
		t.Run(kind, func(t *testing.T) {
			f, q := nativeCollectionFixture()
			p := f.entries[0].Artifact.RelativePath
			raw := string(f.data[p])
			switch kind {
			case "future-message":
				later := strings.Replace(string(recoveryCodexRecords(`"cli"`)[1].Bytes), "10:02:00Z", "11:02:00Z", 1)
				f.data[p] = append(f.data[p], []byte(later+"\n")...)
			case "future-created":
				f.data[p] = []byte(strings.ReplaceAll(raw, "10:00:00Z", "11:00:00Z"))
			case "old-hint":
				at := q.ObservationCutoff.Add(-30 * time.Second)
				q.ProducingLaunch = &at
			case "other-cwd":
				f.data[p] = []byte(strings.ReplaceAll(raw, "/repo", "/other"))
			case "exec":
				f.data[p] = []byte(strings.ReplaceAll(raw, `"source":"cli"`, `"source":"exec"`))
			case "child":
				f.data[p] = []byte(strings.ReplaceAll(raw, `"source":"cli"`, `"source":"cli","parent_thread_id":"019d2222-2222-7222-8222-222222222222"`))
			}
			f.entries[0].Size = int64(len(f.data[p]))
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if kind == "future-message" {
				if got.Incomplete || len(got.Ranked) != 1 || got.Ranked[0].Candidate.SemanticLast.After(*q.ObservationCutoff) {
					t.Fatalf("cutoff ignored: %+v", got)
				}
			} else if len(got.Ranked) != 0 {
				t.Fatalf("ineligible candidate: %+v", got)
			}
		})
	}
}

type countedNativeRuntime struct {
	*recoveryFake
	cwdCalls int
}

func (f *countedNativeRuntime) ResolveRecoveryCWD(ctx context.Context, cwd string) (string, error) {
	f.cwdCalls++
	return f.recoveryFake.ResolveRecoveryCWD(ctx, cwd)
}

func TestRecoveryNativeOnlyCollectorLinearWork(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f, q := nativeCollectionFixture()
			original := f.entries[0]
			raw := string(f.data[original.Artifact.RelativePath])
			f.entries = nil
			var bytes int64
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("019d1111-1111-7111-8111-%012d", i)
				e := original
				e.Artifact.RelativePath = strings.Replace(e.Artifact.RelativePath, recoveryTestID, id, 1)
				// Reverse semantic activity relative to native identity; mtime remains identical.
				data := []byte(strings.ReplaceAll(strings.ReplaceAll(raw, recoveryTestID, id), "10:02:00Z", fmt.Sprintf("10:%02d:00Z", 2+i)))
				e.Size = int64(len(data))
				bytes += e.Size
				f.entries = append(f.entries, e)
				f.data[e.Artifact.RelativePath] = data
				f.entries = append(f.entries, FileEntry{Artifact: Artifact{StorageRoot: "codex-sessions", RelativePath: fmt.Sprintf("unrelated-%d.txt", i)}})
			}
			cutoff := q.ObservationCutoff.Add(time.Hour)
			q.ObservationCutoff = &cutoff
			runtime := &countedNativeRuntime{recoveryFake: f}
			got := CollectRecoveryEvidence(context.Background(), runtime, q)
			counts := map[string]int{}
			for _, call := range f.calls {
				counts[call]++
			}
			if got.Incomplete || len(got.Ranked) != n || got.MetadataEntries != 2*n || got.BytesRead != bytes || counts["list"] != 1 || counts["stat"] != 2*n || counts["read"] != n || runtime.cwdCalls != n+1 {
				t.Fatalf("nonlinear/incomplete scan: evidence=%+v counts=%v cwd=%d", got, counts, runtime.cwdCalls)
			}
			for i, r := range got.Ranked {
				if r.Candidate.NativeID != fmt.Sprintf("019d1111-1111-7111-8111-%012d", n-1-i) || r.Score != (RecoveryTailScore{}) {
					t.Fatalf("rank %d=%+v", i, r)
				}
			}
		})
	}
}

func TestRecoveryNativeOnlyCollectorExactSharedReadBudget(t *testing.T) {
	for _, over := range []int64{0, 1} {
		t.Run(fmt.Sprint(over), func(t *testing.T) {
			f, q := nativeCollectionFixture()
			q.BytesAlreadyRead = RecoveryTotalReadLimit - f.entries[0].Size + over
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if over == 0 {
				if got.Incomplete || len(got.Ranked) != 1 || got.BytesRead != RecoveryTotalReadLimit {
					t.Fatalf("exact bound rejected: %+v", got)
				}
			} else {
				if !got.Incomplete || len(got.Ranked) != 0 || got.BytesRead > RecoveryTotalReadLimit || strings.Contains(strings.Join(f.calls, ","), "read") {
					t.Fatalf("over bound read: %+v calls=%v", got, f.calls)
				}
			}
		})
	}
}

func TestRecoveryNativeOnlyCollectorDuplicateProvenanceIncomplete(t *testing.T) {
	for _, kind := range []string{"metadata", "root"} {
		t.Run(kind, func(t *testing.T) {
			f, q := nativeCollectionFixture()
			duplicate := f.entries[0]
			if kind == "root" {
				duplicate.Artifact.RelativePath = "2026/08/29/rollout-copy-" + recoveryTestID + ".jsonl"
				f.data[duplicate.Artifact.RelativePath] = append([]byte(nil), f.data[f.entries[0].Artifact.RelativePath]...)
			}
			f.entries = append(f.entries, duplicate)
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if !got.Incomplete || len(got.Ranked) > 1 {
				t.Fatalf("duplicate provenance claimed complete: %+v", got)
			}
		})
	}
}

func TestRecoveryNativeOnlyCollectorExactMetadataBoundary(t *testing.T) {
	for _, over := range []int{0, 1} {
		t.Run(fmt.Sprint(over), func(t *testing.T) {
			f, q := nativeCollectionFixture()
			for i := 1; i < RecoveryMetadataLimit+over; i++ {
				f.entries = append(f.entries, FileEntry{Artifact: Artifact{StorageRoot: "codex-sessions", RelativePath: fmt.Sprintf("unrelated-%d.txt", i)}})
			}
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if got.MetadataEntries != RecoveryMetadataLimit || len(got.Ranked) != 1 || got.Incomplete != (over == 1) {
				t.Fatalf("metadata boundary incomplete=%v entries=%d candidates=%d reasons=%v", got.Incomplete, got.MetadataEntries, len(got.Ranked), got.Reasons)
			}
		})
	}
}

func TestRecoveryNativeOnlyCollectorExactCandidateBoundary(t *testing.T) {
	for _, over := range []int{0, 1} {
		t.Run(fmt.Sprint(over), func(t *testing.T) {
			f, q := nativeCollectionFixture()
			original := f.entries[0]
			raw := string(f.data[original.Artifact.RelativePath])
			f.entries = nil
			for i := 0; i < RecoveryCandidateLimit+over; i++ {
				id := fmt.Sprintf("019d1111-1111-7111-8111-%012d", i)
				e := original
				e.Artifact.RelativePath = strings.Replace(e.Artifact.RelativePath, recoveryTestID, id, 1)
				f.entries = append(f.entries, e)
				f.data[e.Artifact.RelativePath] = []byte(strings.ReplaceAll(raw, recoveryTestID, id))
			}
			got := CollectRecoveryEvidence(context.Background(), f, q)
			if len(got.Ranked) != RecoveryCandidateLimit || got.Incomplete != (over == 1) {
				t.Fatalf("candidate boundary incomplete=%v candidates=%d reasons=%v", got.Incomplete, len(got.Ranked), got.Reasons)
			}
		})
	}
}

func TestRecoveryNativeOnlyExactRevalidationUsesCutoffWithoutEnumeration(t *testing.T) {
	f, q := nativeCollectionFixture()
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if got.Incomplete || len(got.Ranked) != 1 {
		t.Fatal("invalid fixture")
	}
	f.calls = nil
	c := got.Ranked[0].Candidate
	if err := RevalidateRecoveryCandidate(context.Background(), f, c.Entry, c.Agent, c.NativeID, c.CWD, *q.ObservationCutoff); err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.calls, ",") != "stat,read,stat" {
		t.Fatalf("exact selection enumerated or reread: %v", f.calls)
	}
	f.entries[0].MutationToken = "new generation"
	if err := RevalidateRecoveryCandidate(context.Background(), f, c.Entry, c.Agent, c.NativeID, c.CWD, *q.ObservationCutoff); err == nil {
		t.Fatal("changed native-only selection accepted")
	}
}
