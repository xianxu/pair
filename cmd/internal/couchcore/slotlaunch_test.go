package couchcore

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type slotReadinessFunc func(context.Context, ProvisionRequest) (ProvisionResult, error)

func (f slotReadinessFunc) Ensure(ctx context.Context, r ProvisionRequest) (ProvisionResult, error) {
	return f(ctx, r)
}

func slotReadyResult(s *ThreadStore) ProvisionResult {
	return ProvisionResult{SchemaVersion: 1, Address: fmt.Sprintf("%s:%d", s.slot.Repo, s.slot.Number), Path: s.slot.WorktreeRoot, RestingBranch: fmt.Sprintf("main-slot%d", s.slot.Number), Disposition: "ready"}
}

func slotLaunchFixture(t *testing.T) (*Couch, ThreadRecord) {
	t.Helper()
	s := testLocalThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = s.slot.WorktreeRoot, s.slot.WorktreeRoot
	created, err := s.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	claimed, err := s.CommitStartClaim(created.Address, created.Revision, s.slot.RepoIdentity, time.Now(), StartEvent{Kind: StartClaimed, Nonce: "slot-start", Owner: SupervisorOwner{PID: 42, Identity: "owner"}, Profile: &profile, Shape: StartFreshExisting})
	if err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: s, Git: NewFakeGit(map[GitCall]string{{Dir: s.slot.WorktreeRoot, Args: "rev-parse --git-common-dir"}: s.slot.RepoIdentity})}
	return c, claimed
}

func TestPrepareTrackedWorkspaceClaimAndReadiness(t *testing.T) {
	c, claimed := slotLaunchFixture(t)
	calls := 0
	c.Workspaces = slotReadinessFunc(func(ctx context.Context, r ProvisionRequest) (ProvisionResult, error) {
		calls++
		if r.Path != c.Threads.slot.PrimaryRoot || r.Slot != c.Threads.slot.Number {
			t.Fatalf("request %+v", r)
		}
		return slotReadyResult(c.Threads), nil
	})
	if err := c.prepareTrackedWorkspace(context.Background(), claimed, "wrong", false); err == nil {
		t.Fatal("accepted wrong claim")
	}
	if calls != 0 {
		t.Fatal("provisioned before claim validation")
	}
	if err := c.prepareTrackedWorkspace(context.Background(), claimed, "slot-start", false); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("readiness calls %d", calls)
	}
}

func TestPrepareTrackedWorkspaceRejectsChangesAndCancellation(t *testing.T) {
	for _, mode := range []string{"revision", "result", "repo", "cancel", "failure"} {
		t.Run(mode, func(t *testing.T) {
			c, claimed := slotLaunchFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
				result := slotReadyResult(c.Threads)
				switch mode {
				case "revision":
					if _, err := c.Threads.ApplyThreadMetadata(claimed.Address, claimed.Revision, ThreadMetadataPatch{Name: ptrSlotName("changed")}); err != nil {
						t.Fatal(err)
					}
				case "result":
					result.Address = "other:1"
				case "repo":
					c.Git.(*FakeGit).replies[GitCall{Dir: claimed.WorkingPath, Args: "rev-parse --git-common-dir"}] = "/other/.git"
				case "cancel":
					cancel()
				case "failure":
					return ProvisionResult{}, errors.New("compile failed")
				}
				return result, nil
			})
			if err := c.prepareTrackedWorkspace(ctx, claimed, "slot-start", false); err == nil {
				t.Fatal("readiness change accepted")
			}
			current, err := c.Threads.GetThread(claimed.Address)
			if err != nil || len(current.Incarnations) != 1 {
				t.Fatalf("helper rolled back caller-owned claim: %+v %v", current, err)
			}
		})
	}
}
func ptrSlotName(s string) *string { return &s }

func TestPrepareTrackedWorkspaceWarmAndPrimaryBypass(t *testing.T) {
	c, claimed := slotLaunchFixture(t)
	c.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		t.Fatal("unexpected provision")
		return ProvisionResult{}, nil
	})
	if err := c.prepareTrackedWorkspace(context.Background(), claimed, "", true); err != nil {
		t.Fatal(err)
	}
	global, _ := newTestThreadStore(t)
	record := validThreadRecord(t)
	if _, err := global.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	c.Threads = global
	if err := c.prepareTrackedWorkspace(context.Background(), record, "", false); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareTrackedWorkspaceUsesProvedGitIdentity(t *testing.T) {
	c, claimed := slotLaunchFixture(t)
	actual := "/custom/git-common-dir"
	claimed, err := c.Threads.updateExistingThread(claimed.Address, claimed.Revision, func(r *ThreadRecord) error { r.Incarnations[0].RepoIdentity = actual; return nil })
	if err != nil {
		t.Fatal(err)
	}
	c.Git.(*FakeGit).replies[GitCall{Dir: claimed.WorkingPath, Args: "rev-parse --git-common-dir"}] = actual
	c.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		return slotReadyResult(c.Threads), nil
	})
	if err := c.prepareTrackedWorkspace(context.Background(), claimed, "slot-start", false); err != nil {
		t.Fatal(err)
	}
}

func TestSlotColdResumeRechecksBindingAfterReadiness(t *testing.T) {
	local := testLocalThreadStore(t)
	env := newTestEnv(t, local.slot.WorktreeRoot)
	env.Couch.Threads = local
	env.Git.replies[GitCall{Dir: local.slot.WorktreeRoot, Args: "rev-parse --git-common-dir"}] = local.slot.RepoIdentity
	record := actionableTestThread("couch-1111111111111111", env.Now)
	record.StartingPath, record.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, env.Now)
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-before")
	calls := 0
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		calls++
		if calls == 1 {
			return ProvisionResult{}, errors.New("compile failed once")
		}
		env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-after")
		return slotReadyResult(local), nil
	})
	_, _, err = env.Couch.ResumeContext(context.Background(), created.Address)
	if calls != 1 || err == nil || len(env.Runner.Ops) != 0 {
		t.Fatalf("compile failure released helper: calls=%d error=%v ops=%v", calls, err, env.Runner.Ops)
	}
	_, _, err = env.Couch.ResumeContext(context.Background(), created.Address)
	if calls != 2 || err == nil {
		t.Fatalf("readiness calls=%d resume error=%v", calls, err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("launched after binding changed: %v", env.Runner.Ops)
	}
	current, readErr := local.GetThread(created.Address)
	if readErr != nil || len(current.Incarnations) != 0 || current.VerifiedPark == nil {
		t.Fatalf("park not preserved: %+v %v", current, readErr)
	}
}

func TestSlotWarmResumeSkipsWorkspaceReadiness(t *testing.T) {
	local := testLocalThreadStore(t)
	env := newTestEnv(t, local.slot.WorktreeRoot)
	env.Couch.Threads = local
	env.Git.replies[GitCall{Dir: local.slot.WorktreeRoot, Args: "rev-parse --git-common-dir"}] = local.slot.RepoIdentity
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	record.Reservation = false
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	session := "pair-" + string(created.Address.Tag)
	env.Artifacts.SetPairSession(created.Address, session, true)
	env.Artifacts.SetDetachedSession(created.Address, session)
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		t.Fatal("warm reattach compiled")
		return ProvisionResult{}, nil
	})
	_, handle, err := env.Couch.ResumeContext(context.Background(), created.Address)
	if err != nil || handle == nil {
		t.Fatalf("warm reattach failed: %v", err)
	}
}
