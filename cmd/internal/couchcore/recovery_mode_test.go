package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestPreparedRecoveryReportsActualContinuityMode(t *testing.T) {
	for _, tc := range []struct {
		name        string
		kind        RecoverySelectionKind
		warm, keyed bool
		want        ContinuityAction
	}{
		{"proven surviving session", RecoveryProven, true, false, Continue},
		{"proven absent session", RecoveryProven, false, false, ResumeDisk},
		{"inferred native conversation", RecoveryInferred, false, true, ResumeDisk},
		{"explicit fresh", RecoveryFresh, false, false, Fresh},
		{"candidate fresh", RecoveryInferred, false, true, Fresh},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
			selection := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: tc.kind}
			if tc.keyed {
				p := f.preview(t)
				selection.CandidateKey = p.Candidates[0].Key
				if tc.want == Fresh {
					selection.CandidateKey = p.Fresh.Key
				}
			}
			if tc.warm {
				f.Artifacts.SetDetachedSession(f.record.Address, "pair-recovery-mode")
			}
			result, err := f.Couch.RecoverSelection(t.Context(), selection)
			if err != nil {
				t.Fatal(err)
			}
			if result.Start.Handle == nil || result.Mode != tc.want {
				t.Fatalf("actual recovery mode = %q, want %q (handle=%v notice=%q)", result.Mode, tc.want, result.Start.Handle, result.Notice)
			}
		})
	}
}
