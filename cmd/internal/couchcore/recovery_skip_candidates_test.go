package couchcore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// TestRecoverySkipCandidatesKeepsEveryNonCandidateFact is pair:3's matrix for
// pair#726 M2: with SkipCandidates the reader reads no native transcript (no
// candidate is ranked although a matching one exists), and every fact a direct
// Proven or Fresh selection consumes is identical to a full read.
func TestRecoverySkipCandidatesKeepsEveryNonCandidateFact(t *testing.T) {
	for _, mode := range []string{"valid", "expired", "missing-screen", "retained-source"} {
		t.Run(mode, func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, true)
			record.WorkingPath, _ = filepath.EvalSymlinks(record.WorkingPath)
			native := filepath.Join(reader.HomeDir, ".codex", "sessions", "2026", "10", "15")
			if err := os.MkdirAll(native, 0700); err != nil {
				t.Fatal(err)
			}
			const id = "019d2222-2222-7222-8222-222222222222"
			stamp := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
			raw := fmt.Sprintf("{\"timestamp\":%q,\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"cli\",\"cwd\":%q}}\n{\"timestamp\":%q,\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"previous objective and remaining work\"}]}}\n", stamp, id, record.WorkingPath, stamp)
			if err := os.WriteFile(filepath.Join(native, "rollout-2026-10-15T00-00-00-"+id+".jsonl"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			observedAt := time.Now().UTC()
			switch mode {
			case "expired":
				observedAt = observedAt.Add(storagegc.CaptureRetentionPeriod + time.Hour)
			case "missing-screen":
				paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: reader.DataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
				if err != nil {
					t.Fatal(err)
				}
				files, _ := paths.ScrollbackArtifacts("codex")
				if err := os.Remove(files.Raw); err != nil {
					t.Fatal(err)
				}
			case "retained-source":
				record.RecoveryContext = &threadrecord.RecoveryContext{Version: 2, Source: &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: observedAt.Add(-10 * time.Minute), Revision: record.Revision}}
			}
			full, err := reader.ReadRequest(context.Background(), record, RecoveryObservationRequest{ObservedAt: observedAt})
			if err != nil {
				t.Fatal(err)
			}
			skip, err := reader.ReadRequest(context.Background(), record, RecoveryObservationRequest{ObservedAt: observedAt, SkipCandidates: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(full.Ranked) == 0 {
				t.Fatalf("control: the full read ranked no candidate, so it proves nothing (reasons %v)", full.Reasons)
			}
			if len(skip.Ranked) != 0 {
				t.Fatalf("skip read a native transcript: %+v", skip.Ranked)
			}
			if !reflect.DeepEqual(skip.Source, full.Source) || skip.Generation != full.Generation || skip.Agent != full.Agent || skip.Branch != full.Branch || skip.Issue != full.Issue || !reflect.DeepEqual(skip.Screen, full.Screen) || !reflect.DeepEqual(skip.Capture, full.Capture) {
				t.Fatalf("a consumed fact differs:\nfull %+v\nskip %+v", full, skip)
			}
			for _, reason := range skip.Reasons {
				if !slices.Contains(full.Reasons, reason) {
					t.Fatalf("skip invented a reason %q (full %v)", reason, full.Reasons)
				}
			}
		})
	}
}

// TestPreparedRecoveryScansCandidatesOnlyForAnInferredChoice: direct Proven
// and Fresh selections ask the reader to skip the native scan; the preview and
// a keyed (inferred) choice still scan.
func TestPreparedRecoveryScansCandidatesOnlyForAnInferredChoice(t *testing.T) {
	record := func(f *recoveryExecutionFixture) *[]bool {
		var asked []bool
		f.Couch.RecoveryEvidence = func(_ context.Context, _ ThreadRecord, q RecoveryObservationRequest) (RecoveryObservation, error) {
			asked = append(asked, q.SkipCandidates)
			return f.observation, nil
		}
		return &asked
	}
	t.Run("fresh", func(t *testing.T) {
		f := newRecoveryExecutionFixture(t)
		asked := record(f)
		if _, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh, Reference: "pair:99"}); err != nil {
			t.Fatal(err)
		}
		if len(*asked) == 0 || slices.Contains(*asked, false) {
			t.Fatalf("fresh observations %v", *asked)
		}
	})
	t.Run("proven", func(t *testing.T) {
		f := newRecoveryExecutionFixture(t)
		f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
		asked := record(f)
		if _, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryProven}); err != nil {
			t.Fatal(err)
		}
		if len(*asked) == 0 || slices.Contains(*asked, false) {
			t.Fatalf("proven observations %v", *asked)
		}
	})
	t.Run("inferred", func(t *testing.T) {
		f := newRecoveryExecutionFixture(t)
		asked := record(f)
		key := f.preview(t).Candidates[0].Key
		if _, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key); err != nil {
			t.Fatal(err)
		}
		if len(*asked) == 0 || slices.Contains(*asked, true) {
			t.Fatalf("inferred observations %v", *asked)
		}
	})
}
