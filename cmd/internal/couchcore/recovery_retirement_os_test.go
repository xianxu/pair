package couchcore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestConfirmedRetirementAdapterBindsExactRecordedNameAndDefersEffects(t *testing.T) {
	reader, record := recoveryEvidenceFixture(t, false)
	record.SessionBinding = &couchidentity.SessionBinding{Name: "obsolete"}
	record.Incarnations = []ThreadIncarnation{{Start: &ThreadStartClaim{SessionBinding: &couchidentity.SessionBinding{Name: "pending-exact"}}}}
	prepared, checked, retired := 0, 0, 0
	failure := errors.New("generation changed")
	expected := context.WithValue(context.Background(), struct{}{}, "effect-context")
	summary := []launcher.RecoveryProcess{{PID: 13, Identity: "native-generation", Role: "native-agent", ParentPID: 12}}
	prepare := func(_ context.Context, root, scope, tag, name string) (recoveryRetirementEvidence, error) {
		prepared++
		physical, _ := filepath.EvalSymlinks(reader.DataDir)
		if root != physical || scope != record.Address.RepoScope || tag != string(record.Address.Tag) || name != "pending-exact" {
			t.Fatalf("wrong address: %q %q %q %q", root, scope, tag, name)
		}
		return recoveryRetirementEvidence{processes: summary, digest: "exact-set-digest", check: func(ctx context.Context) error {
			checked++
			if ctx != expected {
				t.Fatal("check lost caller context")
			}
			return failure
		}, retire: func(ctx context.Context) error {
			retired++
			if ctx != expected {
				t.Fatal("retire lost caller context")
			}
			return failure
		}}, nil
	}
	got, err := reader.prepareConfirmedTerminal(context.Background(), record, prepare)
	if err != nil || prepared != 1 || checked != 0 || retired != 0 {
		t.Fatalf("prepare=%d check=%d retire=%d err=%v", prepared, checked, retired, err)
	}
	if got.Digest != "exact-set-digest" || len(got.Stops) != 1 || got.Stops[0].PID != 13 || got.Stops[0].Identity != "native-generation" || got.Stops[0].Role != "native-agent" {
		t.Fatalf("retirement=%+v", got)
	}
	summary[0].PID = 99
	if got.Stops[0].PID != 13 {
		t.Fatal("summary aliases launcher result")
	}
	if err := got.Check(expected); !errors.Is(err, failure) || checked != 1 {
		t.Fatalf("check=%d err=%v", checked, err)
	}
	if err := got.Retire(expected); !errors.Is(err, failure) || retired != 1 {
		t.Fatalf("retire=%d err=%v", retired, err)
	}
}

func TestConfirmedRetirementAdapterRefusesBeforePreparingWithoutExactName(t *testing.T) {
	for _, mode := range []string{"missing-name", "canceled", "nil-context", "prepare-error"} {
		t.Run(mode, func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, false)
			record.Incarnations = nil
			record.SessionBinding = &couchidentity.SessionBinding{Name: "exact"}
			ctx := context.Background()
			if mode == "missing-name" {
				record.SessionBinding = nil
			}
			if mode == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if mode == "nil-context" {
				ctx = nil
			}
			calls := 0
			failure := errors.New("unowned terminal")
			got, err := reader.prepareConfirmedTerminal(ctx, record, func(context.Context, string, string, string, string) (recoveryRetirementEvidence, error) {
				calls++
				return recoveryRetirementEvidence{}, failure
			})
			if err == nil || got.Check != nil || got.Retire != nil || len(got.Stops) != 0 || got.Digest != "" {
				t.Fatalf("unproven retirement=%+v err=%v", got, err)
			}
			if mode == "prepare-error" {
				if calls != 1 || !errors.Is(err, failure) {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
			} else if calls != 0 {
				t.Fatalf("invalid address reached preparation: %d", calls)
			}
		})
	}
}
