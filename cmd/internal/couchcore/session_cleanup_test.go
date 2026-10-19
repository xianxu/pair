package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
)

type refusingCleanupArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	release   <-chan struct{}
	deadlines []time.Time
}

func TestUnknownCleanupNeverReleasesOwnership(t *testing.T) {
	for _, shape := range []StartShape{StartSpawn, StartColdResume, StartFreshExisting, StartWarmReattach, "future-shape"} {
		for _, dead := range []bool{false, true} {
			for _, live := range []bool{false, true} {
				for _, presence := range []SessionPresence{PresenceUnobserved, PresenceAbsent, PresencePresent} {
					in := StartCleanupInput{Shape: shape, HelperDead: dead, LiveRecord: live, Presence: presence, Cleanup: CleanupUnconfirmed}
					if got := DecideStartCleanup(in); got != DurableMarkUnknown {
						t.Fatalf("unconfirmed cleanup released ownership: %+v -> %s", in, got)
					}
				}
			}
		}
	}
}

func TestSessionCleanupUnconfirmedDetailsUseExistingFormat(t *testing.T) {
	cause := errors.New("observation refused")
	err := &SessionCleanupUnconfirmed{Session: "📁9-4", Cause: errors.Join(context.DeadlineExceeded, cause)}
	if !errors.Is(err, cause) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("cleanup error lost causes")
	}
	if err.FailurePhase() != "launch" || err.FailureOutcome() != "unproven" || len(err.FailureHolders()) != 0 {
		t.Fatalf("unexpected failure details: %+v", err)
	}
	for _, exit := range []*int{nil, new(int)} {
		f := &launchFailureError{failure: LaunchFailure{BeforeExit: exit, Stage: "registration", Nonce: "start-exact", StartingPath: "/repo"}, cause: err}
		if line := OperatorError(f); !strings.Contains(line, "session cleanup unconfirmed; ownership retained") || !strings.Contains(line, "couch --show") {
			t.Fatalf("operator lost cleanup uncertainty: %s", line)
		}
	}
}

func TestPersistentSessionCleanupRetainsExactStartAndPublishesFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
		env.Artifacts.SetRegistration(address, RegistrationUnknown, errors.New("registration unreadable"))
		var startNonce string
		env.Runner.BeforeAcknowledge = func(string) error {
			record, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				return err
			}
			startNonce = record.Incarnations[0].Start.Nonce
			return nil
		}
		release := make(chan struct{})
		a := &refusingCleanupArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, release: release}
		env.Couch.Artifacts = a
		watchdog := time.AfterFunc(7*time.Second, func() { close(release) })
		defer watchdog.Stop()
		_, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
		var uncertain *SessionCleanupUnconfirmed
		if !errors.As(err, &uncertain) || h == nil || h.Alive() {
			t.Fatalf("launch did not return exact unconfirmed cleanup: %T %v", h, err)
		}
		record, getErr := env.Couch.Threads.GetThread(address)
		if getErr != nil || len(record.Incarnations) != 1 {
			t.Fatalf("ownership vanished: %+v %v", record, getErr)
		}
		inc := record.Incarnations[0]
		if inc.State != IncarnationUnknown || inc.Start != nil || record.SessionBinding == nil || record.SessionBinding.Name != uncertain.Session || record.SessionBinding.StartNonce != startNonce || inc.PID != h.PID() || inc.Identity != h.Identity() {
			t.Fatalf("exact unresolved start was not retained: %+v", inc)
		}
		env.Couch.LaunchFailures = NewLaunchFailureStore(env.Couch.Namespace)
		history, readErr := env.Couch.LaunchFailures.Read()
		if readErr != nil || len(history) != 1 {
			t.Fatalf("cleanup failure not persisted: %+v %v", history, readErr)
		}
		f := history[0]
		if f.Phase != "launch" || f.Outcome != "unproven" || !strings.Contains(f.CleanupError, "session cleanup unconfirmed") || !strings.Contains(f.CleanupError, "persistent session observation refusal") || FailureNonce(err) != f.Nonce {
			t.Fatalf("cleanup history lost identity/detail: %+v %v", f, err)
		}
		if line := OperatorError(err); !strings.Contains(line, "ownership retained") {
			t.Fatalf("cleanup history not exposed: %s", line)
		}
		result, showErr := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": string(address.Tag), "repo-scope": address.RepoScope})
		if showErr != nil {
			t.Fatal(showErr)
		}
		show := result.(ShowResult)
		if len(show.LaunchFailures) != 1 || show.LaunchFailures[0].Nonce != f.Nonce || show.LaunchFailures[0].CleanupError != f.CleanupError {
			t.Fatalf("show lost retained cleanup diagnostic: %+v", show)
		}
	})
}

func (a *refusingCleanupArtifacts) QuiesceNamed(ctx context.Context, _ ThreadAddress, _ string) error {
	deadline, _ := ctx.Deadline()
	a.deadlines = append(a.deadlines, deadline)
	select {
	case <-a.release:
		return nil
	default:
		return errors.New("persistent session observation refusal")
	}
}

func TestPostAckSessionCleanupHasOneTotalBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		record := validThreadRecord(t)
		record.SessionBinding = &couchidentity.SessionBinding{C: 9, M: 4, Name: "📁9-4", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: "original"}
		if _, err := env.Couch.Threads.CreateThread(record); err != nil {
			t.Fatal(err)
		}
		h, err := env.Runner.Start("/repo", []string{"pair"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		release := make(chan struct{})
		a := &refusingCleanupArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, release: release}
		env.Couch.Artifacts = a
		// The virtual watchdog lets the broken implementation unwind too; it is
		// later than the product contract and never leaves a leaked test task.
		watchdog := time.AfterFunc(7*time.Second, func() { close(release) })
		defer watchdog.Stop()
		began := time.Now()
		err = env.Couch.quiescePostAckStart(record.Address, h, StartSpawn)
		if elapsed := time.Since(began); elapsed > 6*time.Second {
			t.Fatalf("persistent session refusal exceeded the shared budget: %v (%d attempts)", elapsed, len(a.deadlines))
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cleanup error lacks deadline: %v", err)
		}
		if h.Alive() {
			t.Fatal("cleanup returned with a live helper")
		}
		if len(a.deadlines) < 2 {
			t.Fatalf("no persistent-error retry exercised: %d", len(a.deadlines))
		}
		for _, deadline := range a.deadlines {
			if deadline.IsZero() || !deadline.Equal(a.deadlines[0]) {
				t.Fatalf("session attempt reset/lost its deadline: %v", a.deadlines)
			}
		}
	})
}

// A contextual deleter consumes the caller's remaining admission budget.
type deadlineSessionDeleter struct {
	legacyCalls int
	deadline    time.Time
}

func (d *deadlineSessionDeleter) DeleteSession(string) error {
	d.legacyCalls++
	return nil
}
func (d *deadlineSessionDeleter) DeleteSessionContext(ctx context.Context, _ string) error {
	d.deadline, _ = ctx.Deadline()
	<-ctx.Done()
	return ctx.Err()
}
func TestSessionCleanupCarriesDeadlineThroughNativeDelete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
		name := "📁1-7"
		checker := NewScopedThreadArtifactCollisionChecker(t.TempDir())
		checker.OwnerProbe = testOwnerProbe(checker.GlobalDataDir, name, address)
		deleter := &deadlineSessionDeleter{}
		checker.Sessions = deleter
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		deadline, _ := ctx.Deadline()
		err := checker.QuiesceNamed(ctx, address, name)
		if !errors.Is(err, context.DeadlineExceeded) || deleter.legacyCalls != 0 || !deleter.deadline.Equal(deadline) {
			t.Fatalf("delete lost budget: err=%v legacy=%d deadline=%v want=%v", err, deleter.legacyCalls, deleter.deadline, deadline)
		}
	})
}
