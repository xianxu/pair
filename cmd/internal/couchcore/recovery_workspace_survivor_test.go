package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRetainedRecoveryFreshUsesReadOnlyWorkspaceForUnprovedOwner(t *testing.T) {
	for _, mode := range []string{"unknown-owner", "unverified-checkout", "missing-catalog"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			slot := *f.Couch.Threads.slot
			f.Proc.SetUnknown(42)
			f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
				return RecoveryRetirement{}, errors.New("owner cannot be attributed")
			}
			candidate := SlotCandidate{Identity: slot, Verified: mode != "unverified-checkout"}
			f.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{slot.PrimaryRoot: {Slots: []SlotCandidate{candidate}}}}
			if mode == "missing-catalog" {
				f.Couch.Slots = nil
			}
			sentinel := filepath.Join(slot.WorktreeRoot, "operator-work")
			original := []byte("working files must survive recovery\n")
			if err := os.WriteFile(sentinel, original, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
			if mode == "unknown-owner" {
				if err != nil || result.Start.Handle == nil {
					t.Fatalf("valid read-only Fresh: %+v %v", result, err)
				}
			} else if err == nil || len(f.Runner.Ops) != 0 {
				t.Fatalf("invalid checkout launched: %+v %v", result, err)
			}
			raw, readErr := os.ReadFile(sentinel)
			if readErr != nil || string(raw) != string(original) || f.prepared != 0 || len(f.Proc.Signals[42]) != 0 || len(f.Proc.GroupSignals[42]) != 0 {
				t.Fatalf("surviving workspace changed: prepare=%d file=%q read=%v signals=%v groups=%v", f.prepared, raw, readErr, f.Proc.Signals, f.Proc.GroupSignals)
			}
		})
	}
}

func TestPreparedRecoveryAbsentSourceKeepsNormalWorkspaceReadiness(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "fresh"}[fresh], func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			preview := f.preview(t)
			key := preview.Candidates[0].Key
			if fresh {
				key = preview.Fresh.Key
			}
			_, err := f.Couch.ResumeCandidate(t.Context(), ResumeTarget{Address: f.record.Address}, key)
			if err != nil || f.prepared != 1 {
				t.Fatalf("normal readiness: count=%d err=%v", f.prepared, err)
			}
		})
	}
}
