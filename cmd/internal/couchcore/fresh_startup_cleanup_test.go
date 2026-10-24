package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keep catalog resolution real, controlling only the post-claim observation.
// The explicit marker models the native claim's ownership across that gap;
// FakeThreadArtifactCollisionChecker's ordinary claim is deliberately a no-op.
type startupCleanupArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	claimed    ThreadAddress
	held       bool
	releases   int
	releaseErr error
}

func (a *startupCleanupArtifacts) Claim(addr ThreadAddress) (ThreadArtifactClaim, error) {
	if _, err := a.FakeThreadArtifactCollisionChecker.Claim(addr); err != nil {
		return nil, err
	}
	a.claimed, a.held = addr, true
	return startupCleanupClaim{a}, nil
}
func (a *startupCleanupArtifacts) Release(addr ThreadAddress) error {
	a.releases++
	if a.releaseErr != nil {
		return a.releaseErr
	}
	if addr == a.claimed {
		a.held = false
	}
	return a.FakeThreadArtifactCollisionChecker.Release(addr)
}

type startupCleanupClaim struct{ owner *startupCleanupArtifacts }

func (c startupCleanupClaim) Release() error { return c.owner.Release(c.owner.claimed) }

type startupCleanupCatalog struct {
	SlotCatalog
	SlotWorkspaceResolver
	artifacts  *startupCleanupArtifacts
	afterClaim func(context.Context, SlotRepository) (SlotRepository, error)
}

func (c startupCleanupCatalog) Discover(ctx context.Context, path string) (SlotRepository, error) {
	repo, err := c.SlotCatalog.Discover(ctx, path)
	if err != nil {
		return repo, err
	}
	if c.artifacts.held && c.afterClaim != nil {
		return c.afterClaim(ctx, repo)
	}
	return repo, nil
}

func startupCleanupFixture(t *testing.T) (*testEnv, *ThreadStore, *startupCleanupArtifacts, SlotCatalog) {
	t.Helper()
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	env := newTestEnv(t, f.Primary, f.host(1))
	base := NewOSSlotCatalog(f)
	env.Couch.Slots = base
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	env.Git.replies[GitCall{Dir: f.host(1), Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	repo, err := base.Discover(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(env.Couch.Namespace, repo.Slots[0].Identity)
	artifacts := &startupCleanupArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts}
	env.Couch.Artifacts = artifacts
	return env, local, artifacts, base
}

func TestFreshStartupPostClaimDiscoveryFailureRollsBack(t *testing.T) {
	env, local, artifacts, base := startupCleanupFixture(t)
	refused := errors.New("post-claim discovery unavailable")
	witnessed := false
	env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
		record, err := local.GetThread(artifacts.claimed)
		if err != nil || len(record.Incarnations) != 1 || record.Incarnations[0].Start == nil {
			t.Fatalf("failure occurred before durable claim: %+v %v", record, err)
		}
		witnessed = true
		return repo, refused
	}}
	start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
	if !errors.Is(err, refused) || !witnessed || start.Handle != nil || len(env.Runner.Ops) != 0 {
		t.Fatalf("wrong failure boundary: start=%+v err=%v witnessed=%t effects=%v", start, err, witnessed, env.Runner.Ops)
	}
	if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("failed startup retained creating/current record: %v", err)
	}
	if artifacts.held || artifacts.releases != 1 {
		t.Errorf("failed startup retained native marker: held=%t releases=%d", artifacts.held, artifacts.releases)
	}
}

func TestFreshStartupPostClaimPreparationFailuresRollBack(t *testing.T) {
	for _, phase := range []string{"identity", "enrollment", "cancel-enrollment"} {
		t.Run(phase, func(t *testing.T) {
			env, local, artifacts, base := startupCleanupFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			witnessed := false
			env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
				if _, err := local.GetThread(artifacts.claimed); err != nil {
					t.Fatalf("failure preceded committed claim: %v", err)
				}
				witnessed = true
				switch phase {
				case "identity":
					repo.Identity.RepoIdentity += "-replacement"
				case "enrollment":
					repo.Slots = append([]SlotCandidate(nil), repo.Slots...)
					repo.Slots[0].Identity.RepoIdentity += "-other-repository"
				case "cancel-enrollment":
					cancel()
				}
				return repo, nil
			}}
			start, err := env.Couch.StartFreshHub(ctx, local.slot.WorktreeRoot)
			if err == nil || !witnessed || start.Handle != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("phase %s boundary: start=%+v err=%v witnessed=%t effects=%v", phase, start, err, witnessed, env.Runner.Ops)
			}
			if phase == "identity" && !errors.Is(err, ErrStartResolutionChanged) {
				t.Errorf("identity error lost: %v", err)
			}
			if phase == "enrollment" && !strings.Contains(err.Error(), "slot enrollment mixes repositories") {
				t.Errorf("enrollment error lost: %v", err)
			}
			if phase == "cancel-enrollment" && !errors.Is(err, context.Canceled) {
				t.Errorf("cancellation lost: %v", err)
			}
			if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("preparation failure kept creating/current record: %v", err)
			}
			if artifacts.held || artifacts.releases != 1 {
				t.Errorf("preparation failure leaked native marker: held=%t releases=%d", artifacts.held, artifacts.releases)
			}
		})
	}
}

func TestFreshStartupPostClaimRollbackPreservesUncertainSuccessor(t *testing.T) {
	for _, kind := range []string{"revision", "nonce", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			env, local, artifacts, base := startupCleanupFixture(t)
			refused := errors.New("post-claim catalog refused after successor")
			var successor []byte
			env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
				current, err := local.GetThread(artifacts.claimed)
				if err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "revision":
					if _, err := local.updateExistingThread(current.Address, current.Revision, func(*ThreadRecord) error { return nil }); err != nil {
						t.Fatal(err)
					}
				case "nonce":
					// Isolate the nonce fence: leave revision unchanged while replacing the
					// native authority. Normal CAS writers change both, tested above.
					current.Incarnations[0].Start.Nonce = "start-0123456789abcdef"
					raw, err := encodeThreadRecord(current)
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(local.recordPath(current.Address), append(raw, '\n'), 0600); err != nil {
						t.Fatal(err)
					}
				case "unreadable":
					if err = os.WriteFile(local.recordPath(current.Address), []byte("unreadable successor must stay untouched\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				successor, err = os.ReadFile(local.recordPath(current.Address))
				if err != nil {
					t.Fatal(err)
				}
				return repo, refused
			}}
			start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
			if !errors.Is(err, refused) || start.Handle != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("wrong successor failure: %+v %v effects=%v", start, err, env.Runner.Ops)
			}
			if err == refused {
				t.Error("cleanup refusal/uncertainty was not joined to discovery failure")
			}
			after, readErr := os.ReadFile(local.recordPath(artifacts.claimed))
			if readErr != nil || string(after) != string(successor) {
				t.Errorf("rollback changed successor bytes: %v", readErr)
			}
			if !artifacts.held || artifacts.releases != 0 {
				t.Errorf("rollback released successor authority: held=%t releases=%d", artifacts.held, artifacts.releases)
			}
		})
	}
}

func TestFreshStartupPostClaimRollbackReleaseErrorIsSurfaced(t *testing.T) {
	env, local, artifacts, base := startupCleanupFixture(t)
	refused, cleanup := errors.New("post-claim discovery failure"), errors.New("native marker release unavailable")
	artifacts.releaseErr = cleanup
	env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) { return repo, refused }}
	start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
	if !errors.Is(err, refused) || !errors.Is(err, cleanup) || start.Handle != nil || len(env.Runner.Ops) != 0 {
		t.Fatalf("cleanup error lost: %+v %v effects=%v", start, err, env.Runner.Ops)
	}
	if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("release error retained creating/current record: %v", err)
	}
	if !artifacts.held || artifacts.releases != 1 {
		t.Errorf("release error had wrong authority result: held=%t releases=%d", artifacts.held, artifacts.releases)
	}
}

func TestFreshStartupPostClaimSuccessTransfersOwnership(t *testing.T) {
	env, local, artifacts, base := startupCleanupFixture(t)
	postClaimDiscoveries := 0
	env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
		postClaimDiscoveries++
		return repo, nil
	}}
	start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
	if err != nil || start.Handle == nil || postClaimDiscoveries == 0 {
		t.Fatalf("handoff failed: %+v %v observations=%d", start, err, postClaimDiscoveries)
	}
	current, err := local.GetThread(artifacts.claimed)
	if err != nil || len(current.Incarnations) != 1 || current.Incarnations[0].State != IncarnationLive {
		t.Fatalf("handoff lost live record: %+v %v", current, err)
	}
	if !artifacts.held || artifacts.releases != 0 {
		t.Errorf("successful handoff rolled back native authority: held=%t releases=%d", artifacts.held, artifacts.releases)
	}
}

func TestFreshStartupPostClaimHandoffFailureHasOneRollback(t *testing.T) {
	env, local, artifacts, base := startupCleanupFixture(t)
	refused := errors.New("post-enrollment workspace preparation refused")
	postClaimDiscoveries := 0
	env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
		postClaimDiscoveries++
		return repo, nil
	}}
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) { return ProvisionResult{}, refused })
	start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
	if !errors.Is(err, refused) || start.Handle != nil || postClaimDiscoveries == 0 || len(env.Runner.Ops) != 0 {
		t.Fatalf("failure missed handoff boundary: %+v %v observations=%d effects=%v", start, err, postClaimDiscoveries, env.Runner.Ops)
	}
	if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("handoff failure kept current record: %v", err)
	}
	if artifacts.held || artifacts.releases != 1 {
		t.Errorf("handoff rollback was duplicated or omitted: held=%t releases=%d", artifacts.held, artifacts.releases)
	}
}

func TestFreshStartupPostClaimEnrollmentPublicationFailureRollsBack(t *testing.T) {
	env, local, artifacts, base := startupCleanupFixture(t)
	refused := errors.New("enrollment manifest publication interrupted")
	publications := 0
	env.Couch.Slots = startupCleanupCatalog{SlotCatalog: base, SlotWorkspaceResolver: base.(SlotWorkspaceResolver), artifacts: artifacts, afterClaim: func(_ context.Context, repo SlotRepository) (SlotRepository, error) {
		if _, err := local.GetThread(artifacts.claimed); err != nil {
			t.Fatalf("publication setup preceded durable claim: %v", err)
		}
		env.Couch.Threads.hooks.AfterPublicationWrite = func(path string) error {
			if path == env.Couch.Threads.manifestPath() {
				publications++
				return refused
			}
			return nil
		}
		return repo, nil
	}}
	start, err := env.Couch.StartFreshHub(t.Context(), local.slot.WorktreeRoot)
	if !errors.Is(err, refused) || start.Handle != nil || publications != 1 || len(env.Runner.Ops) != 0 {
		t.Fatalf("wrong publication boundary: start=%+v err=%v publications=%d effects=%v", start, err, publications, env.Runner.Ops)
	}
	if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("failed enrollment kept local claim: %v", err)
	}
	if artifacts.held || artifacts.releases != 1 {
		t.Errorf("failed publication kept native marker: held=%t releases=%d", artifacts.held, artifacts.releases)
	}
	// Failure happened after a real journal was published. Its later replay may
	// finish enrollment, but must not resurrect the now-rolled-back local claim.
	if _, err := os.Lstat(env.Couch.Threads.journalPath()); err != nil {
		t.Fatalf("publication control missed durable journal: %v", err)
	}
	env.Couch.Threads.hooks = threadStoreHooks{}
	if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(local.recordPath(artifacts.claimed)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("enrollment replay resurrected failed claim: %v", err)
	}
}
