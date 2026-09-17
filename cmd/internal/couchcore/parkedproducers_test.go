package couchcore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive is the class guard
// for #256 M2's C1, and it is the enumeration the finding asked for.
//
// The defect: Task 6b widened who PRODUCES `ThreadParked` -- a record with no
// receipt, no incarnation and no park now classifies `parked` when its ledger
// resolves -- and the sweep re-derived only the consumers of the evidence field
// that widened (`ThreadEvidence.Parked`). `ThreadParked` itself has seven
// readers, and one of them, the switcher's action list, offers `switch-agent`
// on every parked row while `PrepareAgentSwitch` still demanded a park receipt.
// So a ledger-parked row was offered an action that always failed, which
// menu.go names in its own words as how a switcher teaches an operator to
// distrust it.
//
// The rule: WHEN A STATE'S PRODUCER SET WIDENS, ITS CONSUMERS ARE THE
// ENUMERATION -- not the consumers of the evidence field that widened it. This
// test makes that mechanically checkable for the state whose meaning changed:
// every way of producing `parked` must be accepted by every action the menu
// offers a parked row. A new producer with no cell here fails the totality
// check below; a guard that refuses one producer and not another fails its row.
//
// SCOPE, stated because the name used to claim more than the body checks: this
// covers the three guarded actions a parked row offers — resume, switch-agent,
// archive. `name` and `describe` are metadata with no guard. That the menu's
// OFFER set matches the guards' permission set is a separate claim, derived from
// `AllThreadStates() × AllThreadReasons()` in couchtty's
// `TestSwitchAgentOfferedImpliesPermitted`; this test asks the complementary
// question, whether every way of REACHING the state survives those guards.
func TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive(t *testing.T) {
	// The producers of ThreadParked, as ClassifyThread can reach it. Both run
	// through the production gather path, so a classification that stops
	// agreeing fails here rather than in a hand-built evidence literal.
	producers := map[string]func(*testing.T, *testEnv) ThreadRecord{
		"park receipt": func(t *testing.T, env *testEnv) ThreadRecord {
			return createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
		},
		"ledger only": func(t *testing.T, env *testEnv) ThreadRecord {
			// #256 M2's new producer: never parked, session gone, and the
			// ledger still names the conversation.
			record := validThreadRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
			env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
			record.Reservation = false
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			record.LatestLaunchProfile = &profile
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			if created.VerifiedPark != nil {
				t.Fatal("the ledger-only producer grew a park receipt; it would prove the old rule instead")
			}
			return created
		},
		"driverless start claim with a ledger": func(t *testing.T, env *testEnv) ThreadRecord {
			// BR-33's shape: a couch died mid-start, so the record still carries
			// a `creating` incarnation, and the ledger still names the
			// conversation. It classifies `parked` -- and reaches every action
			// guard with debris the guard must clear rather than trip over.
			record := validThreadRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
			env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
			record.Reservation = false
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			record.LatestLaunchProfile = &profile
			record.Incarnations = []ThreadIncarnation{{
				State: IncarnationCreating,
				Start: &ThreadStartClaim{
					Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor",
				},
			}}
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			return created
		},
		"park receipt whose session could not be asked about": func(t *testing.T, env *testEnv) ThreadRecord {
			// The asymmetry ClassifyThread keeps deliberately: a receipt says
			// couch tore the session down itself, so an unanswerable
			// `list-sessions` does not demote the row. It is a producer like any
			// other and every action must accept it.
			record := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error {
				return errTestSessionUnanswerable
			}
			return record
		},
	}

	classified := map[string]bool{}
	for name, build := range producers {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			record := build(t, env)
			env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
			env.Artifacts.SetSessionPresence(record.Address, SessionObservation{State: SessionAbsent})

			rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			var row ActionableThreadSummary
			for _, candidate := range rows {
				if candidate.Address == record.Address {
					row = candidate
				}
			}
			if row.State != ThreadParked {
				t.Fatalf("producer %q classified %q/%q, not parked -- the fixture no longer exercises what it names",
					name, row.State, row.Reason)
			}
			classified[name] = true

			// switch-agent, the reader the sweep missed. It must not refuse for
			// a reason about the RECORD's bookkeeping; a refusal naming the
			// agent, the path or the launch services is a different question.
			if _, err := env.Couch.PrepareAgentSwitch(context.Background(), record.Address, "codex", nil); err != nil {
				t.Fatalf("switch-agent is offered on a %s parked row and refuses it: %v", name, err)
			}

			// resume, through the same eligibility rule the action path uses.
			binding := NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "native-root-1"}
			if _, err := DecideResume(ResumeEligibilityInput{
				Thread: record, WorkingPathExists: true, Binding: binding,
			}); err != nil {
				t.Fatalf("resume is offered on a %s parked row and refuses it: %v", name, err)
			}

			// archive, last: it moves the record.
			if _, err := env.Couch.ArchiveThread(context.Background(), record.Address); err != nil {
				t.Fatalf("archive is offered on a %s parked row and refuses it: %v", name, err)
			}
		})
	}

	// Totality, DERIVED rather than restated. Counting `classified` against
	// `producers` was vacuous -- both come from the literal above, so the
	// assertion could not fail, which is the same "input derived from the
	// expectation" defect this milestone already fixed in a different table.
	//
	// The domain is the classifier's own shape table: every fixture there that
	// classifies `parked` is a producer, and every producer needs a cell here.
	shaped := map[string]bool{}
	for _, shape := range everyThreadShape(t) {
		if state, _ := ClassifyThread(shape.record, shape.evidence); state == ThreadParked {
			shaped[shape.name] = true
		}
	}
	if len(shaped) > len(classified) {
		t.Fatalf("everyThreadShape produces %d parked shapes (%v) but this table covers %d; "+
			"a producer with no cell is the gap BR-33 came through", len(shaped), keysOf(shaped), len(classified))
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestSwitchAgentRefusesARowWhoseSessionSurvives is the fail-closed half of the
// guard BR-33 replaced. Admitting a thread whose agent is alive gives one tree
// two agents -- #272's shape from the other side, because a switch launches
// FRESH rather than reattaching.
//
// It drives SessionPresence, the world fact the CLASSIFICATION reads, not
// recoverySession, which the guard this test was written for used to read. An
// earlier version set the other channel and so proved nothing about the guard
// that exists: a test must read the same world fact the same way the code does.
func TestSwitchAgentRefusesARowWhoseSessionSurvives(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
	}{
		{name: "session still up, so the agent is too", present: true},
		{name: "session unanswerable, so ignorance fails closed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			record := validThreadRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
			env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
			record.Reservation = false
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			record.LatestLaunchProfile = &profile
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			if tc.present {
				env.Artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionPresent})
			} else {
				env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error {
					return errTestSessionUnanswerable
				}
			}

			_, err = env.Couch.PrepareAgentSwitch(context.Background(), created.Address, "codex", nil)
			if err == nil {
				t.Fatal("switch-agent started a fresh agent on a thread whose session it could not prove gone")
			}
			if !strings.Contains(err.Error(), "switch-agent") {
				t.Fatalf("refusal does not name the action the operator took: %v", err)
			}
		})
	}
}

var errTestSessionUnanswerable = errors.New("zellij is not answering")

// TestSwitchAgentRefusesAThreadCouchHostsWithNoIncarnation pins the FAIL-OPEN
// that the first BR-33 fix introduced and the second closed.
//
// #256 M1 made `unrecorded` a live row: couch hosting the process IS the proof,
// and the record's incarnation is not consulted. So a thread couch hosts can
// carry no incarnation at all -- and SwitchAgent parks the source only when one
// is there (`hasOccupiedIncarnation`). A guard that asked the session instead of
// the classification admitted exactly that row whenever the session index had no
// binding, which is two agents on one tree.
//
// The classification says `live`; the guard consumes it and then demands
// something to park.
func TestSwitchAgentRefusesAThreadCouchHostsWithNoIncarnation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	record.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	// Couch hosts a pty child for it; the record names no incarnation. The
	// registry is what classifyForAction reads as live proof, so this is the
	// production shape rather than an injected observation.
	env.Couch.reg = env.Couch.reg.Insert(ActorRecord{
		ID: ActorID("hosted-actor"), Thread: created.Address,
		Args: StartArgs{Worktree: Worktree(created.StartingPath), Cwd: created.WorkingPath},
		PID:  4242, Identity: "hosted",
	})

	state, reason, err := env.Couch.classifyForAction(context.Background(), created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if state != ThreadLive {
		t.Fatalf("fixture classified %q/%q, not live -- it no longer exercises the hosted shape", state, reason)
	}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), created.Address, "codex", nil); err == nil {
		t.Fatal("switch-agent accepted a thread couch is hosting with nothing to park; that starts a second agent on one tree")
	}
}

// TestSwitchAgentCommitAcceptsWhatItsPreviewAccepted closes the preview/commit
// half of the same class.
//
// `PrepareAgentSwitch` consumes the classification, but `SwitchAgent` decided
// whether to park the source from `hasOccupiedIncarnation` -- bookkeeping, and
// only an approximation of "an agent is running". A thread can be `parked` and
// still carry a `creating` incarnation whose couch died mid-start, so the
// preview admitted it and the commit then tried to park a thread with no agent
// and failed `park-incomplete`. An action whose preview says yes and whose
// commit says no is the same lie as a row that offers what its guard refuses.
//
// The classification the preview was ADMITTED on is now carried to the commit,
// so both rest on one observation.
func TestSwitchAgentCommitAcceptsWhatItsPreviewAccepted(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	record.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.LatestLaunchProfile = &profile
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor"},
	}}
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionAbsent})

	if state, reason, err := env.Couch.classifyForAction(context.Background(), created.Address); err != nil || state != ThreadParked {
		t.Fatalf("fixture classified %q/%q (%v), not parked -- it no longer exercises the driverless claim", state, reason, err)
	}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), created.Address, "codex", nil)
	if err != nil {
		t.Fatalf("preview refused: %v", err)
	}
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(created.Address, "pair-switched", true)
		return nil
	}
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{
		Address: created.Address, Agent: "codex", Argv: prepared.Profile.Argv, AcceptedFingerprint: prepared.Fingerprint,
	})
	if err != nil {
		t.Fatalf("the commit refused what its own preview accepted: outcome=%q %v", result.Outcome, err)
	}
	if _, started := result.Started(); !started {
		t.Fatalf("switch did not start: outcome=%q", result.Outcome)
	}
}
