package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// TestEveryParkedProducerIsAcceptedByEveryActionTheMenuOffers is the class guard
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
// `menu.go`'s parked row offers {resume, switch-agent, archive, name, describe}.
// `name` and `describe` are metadata and have no guard.
func TestEveryParkedProducerIsAcceptedByEveryActionTheMenuOffers(t *testing.T) {
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

	// Totality: a producer added to ClassifyThread without a cell above is the
	// exact gap C1 came through, so the count is asserted rather than assumed.
	if len(classified) != len(producers) {
		t.Fatalf("classified %d of %d parked producers", len(classified), len(producers))
	}
}

// TestSwitchAgentRefusesAParkedRowWhoseSessionSurvives is the fail-closed half
// of the guard C1 replaced. Dropping the receipt check must not make
// switch-agent start a second agent behind a session that is still up -- that is
// #272's shape from the other side, and a switch launches FRESH rather than
// reattaching.
func TestSwitchAgentRefusesAParkedRowWhoseSessionSurvives(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
		absent  bool
	}{
		{name: "session still up", present: true},
		{name: "session unanswerable"},
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
				env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)
			} else {
				env.Artifacts.BeforePairSession = func(ThreadAddress) error {
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
