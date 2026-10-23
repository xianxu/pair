package testcouch

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

func TestDriveBindingUsesExactRegisteredIdentity(t *testing.T) {
	primary := filepath.Join(t.TempDir(), "fixture")
	worker := filepath.Join(filepath.Dir(primary), "slot1", "fixture")
	fleet := &Fleet{Workspaces: Workspaces{Primary: primary, Workers: map[int]string{1: worker}}, Options: Options{AgentKind: "claude"}}
	ready := agentfixture.Ready{Agent: "claude", CWD: worker}
	binding := couchmessage.Binding{Slot: "fixture:1", Repository: filepath.Join(primary, ".git"), Scope: "observed-scope", Tag: "observed-tag", Session: "observed-session", Nonce: "observed-nonce", Agent: "claude", Version: "observed-version", PID: 123, Start: "observed-generation"}
	actors := []couchmessage.Candidate{{Binding: binding}}
	got, err := fleet.driveBinding(1, ready, actors)
	if err != nil || got != binding {
		t.Fatalf("registered binding changed: %+v %v", got, err)
	}
	for _, tc := range []struct {
		name   string
		n      int
		mutate func(*agentfixture.Ready, *[]couchmessage.Candidate)
	}{
		{"unknown-slot", 2, func(*agentfixture.Ready, *[]couchmessage.Candidate) {}},
		{"negative-slot", -1, func(*agentfixture.Ready, *[]couchmessage.Candidate) {}},
		{"wrong-workspace", 1, func(r *agentfixture.Ready, _ *[]couchmessage.Candidate) { r.CWD = primary }},
		{"wrong-fixture-personality", 1, func(r *agentfixture.Ready, _ *[]couchmessage.Candidate) { r.Agent = "codex" }},
		{"missing-registration", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) { *a = nil }},
		{"duplicate-registration", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) { *a = append(*a, (*a)[0]) }},
		{"slot-alias", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) { (*a)[0].Binding.Slot = "other:1" }},
		{"foreign-repository", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) {
			(*a)[0].Binding.Repository = filepath.Join(filepath.Dir(primary), "foreign", ".git")
		}},
		{"wrong-registration-personality", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) { (*a)[0].Binding.Agent = "codex" }},
		{"incomplete-registration", 1, func(_ *agentfixture.Ready, a *[]couchmessage.Candidate) { (*a)[0].Binding.Nonce = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := ready
			a := append([]couchmessage.Candidate(nil), actors...)
			tc.mutate(&r, &a)
			if _, err := fleet.driveBinding(tc.n, r, a); err == nil {
				t.Fatal("foreign or incomplete actor identity accepted")
			}
		})
	}
	ready.CWD = primary
	binding.Slot = "fixture:0"
	if got, err := fleet.driveBinding(0, ready, []couchmessage.Candidate{{Binding: binding}}); err != nil || got != binding {
		t.Fatalf("steward binding rejected: %+v %v", got, err)
	}
}

func TestObserveDriveReadinessFailureReleasesFleetAdmission(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	fleet := &Fleet{Roots: RootsPlan{Root: root, Mutable: map[string]string{"store": filepath.Join(root, "store"), "control": filepath.Join(root, "control")}}, Workspaces: Workspaces{Primary: filepath.Join(root, "fixture")}, Options: Options{AgentKind: "claude"}, ctx: t.Context()}
	// No readiness file exists. The observation must stop before discovering
	// actors and release its owner's joined-operation slot even on this error.
	if _, err := fleet.ObserveDrive(t.Context(), 0); err == nil {
		t.Fatal("unready actor observed")
	}
	if fleet.active != 0 {
		t.Fatalf("observation leaked active operation: %d", fleet.active)
	}
	select {
	case <-fleet.idle:
	default:
		t.Fatal("observation did not release joined-operation lifetime")
	}
	fleet.Roots.Mutable["store"] = filepath.Dir(root)
	if _, err := fleet.ObserveDrive(t.Context(), 0); err == nil || !strings.Contains(err.Error(), "owned store namespace") {
		t.Fatalf("foreign namespace admitted: %v", err)
	}
}

func TestObserveDriveRequiresSelectedOwnedRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, selected := range []string{"", filepath.Dir(root), root + string(filepath.Separator)} {
		t.Run("selected="+selected, func(t *testing.T) {
			t.Setenv("COUCH_ISOLATED_ROOT", selected)
			fleet := &Fleet{Roots: RootsPlan{Root: root}, ctx: t.Context()}
			if _, err := fleet.ObserveDrive(t.Context(), 1); err == nil || !strings.Contains(err.Error(), "canonical owned fleet root") {
				t.Fatalf("unselected or foreign isolated root admitted: %v", err)
			}
			if fleet.active != 0 {
				t.Fatalf("root refusal leaked active operation: %d", fleet.active)
			}
			select {
			case <-fleet.idle:
			default:
				t.Fatal("root refusal did not release joined-operation lifetime")
			}
		})
	}
}

func TestObserveDriveHonorsFleetAdmissionRefusals(t *testing.T) {
	for _, reason := range []string{"canceled", "closing", "uncertain"} {
		t.Run(reason, func(t *testing.T) {
			fleet := &Fleet{ctx: t.Context()}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch reason {
			case "canceled":
				cancel()
			case "closing":
				fleet.closing = true
			case "uncertain":
				fleet.model.Error = "cleanup not joined"
			}
			_, err := fleet.ObserveDrive(ctx, 1)
			if err == nil || reason == "canceled" && !errors.Is(err, context.Canceled) || reason == "uncertain" && !errors.Is(err, agentfixture.ErrCleanupUncertain) {
				t.Fatalf("owner refusal lost: %v", err)
			}
			if fleet.active != 0 || fleet.idle != nil {
				t.Fatal("refused read entered fleet operation lifetime")
			}
		})
	}
}
