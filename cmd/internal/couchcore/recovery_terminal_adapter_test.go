package couchcore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestRecoveryTerminalAdapterDefersRetirementAndPropagatesFailure(t *testing.T) {
	ctx := context.Background()
	reader, record := recoveryEvidenceFixture(t, false)
	record.SessionBinding = &couchidentity.SessionBinding{Name: "old-terminal"}
	prepared, retired := 0, 0
	failure := errors.New("generation changed")
	reader.prepareTerminal = func(_ context.Context, root, scope, tag, session string) (launcher.RecoveryTerminalPlan, error) {
		prepared++
		physical, _ := filepath.EvalSymlinks(reader.DataDir)
		if root != physical || scope != record.Address.RepoScope || tag != string(record.Address.Tag) || session != "old-terminal" {
			t.Fatalf("wrong owner %q %q %q %q", root, scope, tag, session)
		}
		return launcher.RecoveryTerminalPlan{}, nil
	}
	reader.retireTerminal = func(context.Context, launcher.RecoveryTerminalPlan) error { retired++; return failure }
	retire, err := reader.PrepareTerminal(ctx, record)
	if err != nil || prepared != 1 || retired != 0 || retire == nil {
		t.Fatalf("prepare=%d retire=%d err=%v", prepared, retired, err)
	}
	if err := retire(ctx); !errors.Is(err, failure) || retired != 1 {
		t.Fatalf("retire=%d err=%v", retired, err)
	}
	reader.prepareTerminal = func(context.Context, string, string, string, string) (launcher.RecoveryTerminalPlan, error) {
		return launcher.RecoveryTerminalPlan{}, failure
	}
	if retire, err := reader.PrepareTerminal(ctx, record); !errors.Is(err, failure) || retire != nil {
		t.Fatalf("failed preparation returned closure: %v", err)
	}
}

func TestRecoveryRetirementColdLaunchAllocatesNewTerminal(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.Identities = couchidentity.IdentityStore{HostDir: t.TempDir(), StoreDir: f.Dir}
	f.Artifacts.SetPairSession(f.record.Address, "old-session", true)
	f.Artifacts.SetSessionPresence(f.record.Address, SessionObservation{State: SessionPresent})
	retired := false
	f.Couch.RecoveryTerminal = func(_ context.Context, record ThreadRecord) (func(context.Context) error, error) {
		return func(context.Context) error {
			if f.preserved != 1 {
				t.Fatal("retired before preservation")
			}
			retired = true
			f.Artifacts.SetPairSession(record.Address, "old-session", false)
			f.Artifacts.SetSessionPresence(record.Address, SessionObservation{State: SessionAbsent})
			return nil
		}, nil
	}
	key := f.preview(t).Candidates[0].Key
	result, err := f.Couch.ResumeCandidate(context.Background(), ResumeTarget{Address: f.record.Address}, key)
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || !retired {
		t.Fatalf("started=%v retired=%v", ok, retired)
	}
	name := continuationChildSession(t, f.Runner, started.Handle.ID())
	if name == "" || name == "old-session" {
		t.Fatalf("reused old terminal/layout: %q", name)
	}
	current, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil || current.SessionBinding == nil || current.SessionBinding.Name != name {
		t.Fatalf("binding=%+v err=%v", current.SessionBinding, err)
	}
}

func TestRecoveryTerminalAdapterResolvesOnlyExactRecordedName(t *testing.T) {
	for _, mode := range []string{"legacy-index", "pending-binding", "missing-binding"} {
		t.Run(mode, func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, false)
			record.SessionBinding = nil
			record.Incarnations = nil
			paths, err := reader.recoveryPaths(record)
			if err != nil {
				t.Fatal(err)
			}
			want := "📁legacy-terminal"
			if mode == "legacy-index" {
				rt := launcher.NewScopedOSRuntime(reader.DataDir, paths.ScopeDir(), "")
				if err := rt.AppendSessionNameIndex(launcher.SessionNameEntry{SessionName: want, RepoRoot: record.WorkingPath, RepoName: "pair", Tag: string(record.Address.Tag), ScopeKey: record.Address.RepoScope}); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "pending-binding" {
				want = "pending-terminal"
				record.SessionBinding = &couchidentity.SessionBinding{Name: "obsolete-terminal"}
				record.Incarnations = []ThreadIncarnation{{Start: &ThreadStartClaim{SessionBinding: &couchidentity.SessionBinding{Name: want}}}}
			}
			calls := 0
			reader.prepareTerminal = func(_ context.Context, _, _, _, name string) (launcher.RecoveryTerminalPlan, error) {
				calls++
				if name != want {
					t.Fatalf("name=%q want=%q", name, want)
				}
				return launcher.RecoveryTerminalPlan{}, nil
			}
			reader.retireTerminal = func(context.Context, launcher.RecoveryTerminalPlan) error { return nil }
			retire, err := reader.PrepareTerminal(context.Background(), record)
			if mode == "missing-binding" {
				if err == nil || retire != nil || calls != 0 {
					t.Fatalf("missing binding reached authority: %v calls=%d", err, calls)
				}
				return
			}
			if err != nil || retire == nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
