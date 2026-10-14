package couchcore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// Use the same accepted shapes as the classifier contract: a positive exact
// terminal listing must not become permission to retire somebody else's work.
func TestPublicWarmAttachPreservesEveryListedRecordShape(t *testing.T) {
	// These fixtures each contain exactly one complete historical client and
	// no lifecycle claim. Only their explicitly Dead variant may be repaired.
	repairShapes := map[string]bool{
		"live and hosted": true,
		"recorded incarnation is gone and so is the session":            true,
		"recorded helper could not be asked about, session survived":    true,
		"hosted, with a second recorded process nobody could ask about": true,
		"recorded helper could not be asked about, session gone":        true,
		"session survived a dead launcher":                              true,
	}
	for _, tc := range everyThreadShape(t) {
		if len(tc.record.Incarnations) == 0 {
			continue
		}
		for _, owner := range []Liveness{Live, Dead, Unknown} {
			t.Run(fmt.Sprintf("%s/owner=%v", tc.name, owner), func(t *testing.T) {
				evidence := tc.evidence
				evidence.Session = SessionObservation{State: SessionPresent}
				evidence.StartOwner = owner
				state, reason := ClassifyThread(tc.record, evidence)
				if state != ThreadLive && state != ThreadDetached {
					t.Fatalf("classifier refused fixture: %s/%s", state, reason)
				}
				env, before, session := readonlyWarmFixture(t, tc.record, owner)
				assertReadonlyWarmAttach(t, env, before, session, owner == Dead && repairShapes[tc.name])
			})
		}
	}
}

func readonlyWarmFixture(t *testing.T, record ThreadRecord, owner Liveness) (*testEnv, ThreadRecord, string) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	before, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, inc := range before.Incarnations {
		identities := []ProcessIdentity{{PID: inc.PID, Identity: inc.Identity}}
		if inc.Start != nil {
			identities = append(identities, ProcessIdentity{PID: inc.Start.OwnerPID, Identity: inc.Start.OwnerIdentity})
		}
		for _, id := range identities {
			if id.PID <= 0 {
				continue
			}
			switch owner {
			case Live:
				env.Proc.Set(id.PID, id.Identity)
			case Unknown:
				env.Proc.SetUnknown(id.PID)
			}
		}
	}
	session := recordSessionName(before)
	if session == "" {
		session = "pair-" + string(before.Address.Tag)
	}
	env.Artifacts.SetDetachedSession(before.Address, session)
	env.Runner.AfterAcknowledge = func(string) error { env.Artifacts.SetPairSession(before.Address, session, true); return nil }
	return env, before, session
}

func assertReadonlyWarmAttach(t *testing.T, env *testEnv, before ThreadRecord, session string, expectRepair ...bool) (ActorRecord, Handle) {
	t.Helper()
	actor, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
	if err != nil {
		t.Fatalf("listed terminal attachment refused: %v", err)
	}
	if h == nil {
		t.Fatal("no attach helper")
	}
	t.Cleanup(func() { env.Runner.SetExited(h.ID(), 0) })
	child := env.Runner.Child(h.ID())
	if !slices.Equal(child.Argv, []string{"pair", "--couch-session-v1", "resume", string(before.Address.Tag)}) {
		t.Fatalf("not exact attach: %v", child.Argv)
	}
	if actor.Thread != before.Address || actor.Shape != StartWarmReattach {
		t.Fatalf("wrong helper ownership: %+v", actor)
	}
	if env.Artifacts.BindingResolutions() != 0 {
		t.Fatal("plain attach resolved native binding")
	}
	if len(expectRepair) > 0 && expectRepair[0] {
		assertWarmClientRepair(t, env, before, actor)
	} else {
		assertWarmRecordUnchanged(t, env, before)
	}
	return actor, h
}

func assertWarmRecordUnchanged(t *testing.T, env *testEnv, before ThreadRecord) {
	t.Helper()
	after, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("attachment altered durable lifecycle (including revision):\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestPublicWarmAttachIgnoresReplacementAndContinuationMetadata(t *testing.T) {
	for _, kind := range []string{"pending-replacement", "registered-replacement", "pending-continuation"} {
		t.Run(kind, func(t *testing.T) {
			record := warmDetachedRecord(t)
			if kind == "pending-continuation" {
				request := testContinuationRequest(t, record)
				record.Continuation = &request
			} else {
				record.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 42, Identity: "old-helper", LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{}}}}
				if kind == "pending-replacement" {
					record.Incarnations[0].State = IncarnationCreating
					record.Incarnations[0].LaunchProfile = nil
					record.Incarnations[0].Start = &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor", LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{}}}
				}
			}
			env, before, session := readonlyWarmFixture(t, record, Live)
			env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
				t.Error("plain attach asked for continuation authority")
				return ContinuationSource{}, errors.New("unavailable")
			}
			assertReadonlyWarmAttach(t, env, before, session)
		})
	}
}

func TestPublicWarmAttachRequiresCurrentPositiveListing(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "listing-error"
		if missing {
			name = "missing-session"
		}
		t.Run(name, func(t *testing.T) {
			env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
			if missing {
				env.Artifacts.SetDetachedSession(before.Address, "")
			} else {
				env.Artifacts.ListedSessionHook = func(ThreadAddress) error { return errors.New("listing failed") }
			}
			_, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
			if err == nil || h != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("unproved listing launched: h=%v err=%v ops=%v", h, err, env.Runner.Ops)
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestPublicWarmAttachRechecksListingBeforeHelperExec(t *testing.T) {
	for _, failure := range []string{"missing", "error"} {
		for _, boundary := range []int{2, 3} {
			t.Run(fmt.Sprintf("%s/query=%d", failure, boundary), func(t *testing.T) {
				env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
				queries := 0
				env.Artifacts.ListedSessionHook = func(ThreadAddress) error {
					queries++
					if queries >= boundary {
						if failure == "error" {
							return errors.New("listing became unavailable")
						}
						env.Artifacts.SetDetachedSession(before.Address, "")
					}
					return nil
				}
				_, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
				if err == nil {
					t.Fatal("attached after terminal proof was lost")
				}
				if h != nil {
					if h.Alive() || env.Runner.Child(h.ID()).ExecCount != 0 {
						t.Fatal("unproved helper executed or survived cancellation")
					}
				}
				if len(env.Couch.actorRegistry().Records()) != 0 {
					t.Fatal("failed attach registered actor")
				}
				assertWarmRecordUnchanged(t, env, before)
			})
		}
	}
}

func TestPublicWarmAttachDoesNotAcquireIndependentLifecycleHold(t *testing.T) {
	var record ThreadRecord
	for _, tc := range everyThreadShape(t) {
		if tc.name == "start this couch is driving" {
			record = tc.record
			break
		}
	}
	env, before, session := readonlyWarmFixture(t, record, Live)
	_, release, err := env.Couch.hold(context.Background(), before.Address, "independent-start")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	assertReadonlyWarmAttach(t, env, before, session)
	// Attachment must neither consume nor release the original operation's hold.
	_, unexpectedRelease, err := env.Couch.hold(context.Background(), before.Address, "replacement")
	if err == nil {
		unexpectedRelease()
		t.Fatal("attachment released independently owned hold")
	}
	var busy *ThreadBusyError
	if !errors.As(err, &busy) || busy.Running != "independent-start" {
		t.Fatalf("original hold changed: %v", err)
	}
}

func TestPublicWarmAttachAcceptsListedTerminalWithAttachedClients(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	env.Artifacts.SetDetachedSession(before.Address, "")
	env.Artifacts.SetPairSession(before.Address, session, true)
	assertReadonlyWarmAttach(t, env, before, session)
	if env.Artifacts.DetachedQueries() != 0 {
		t.Fatal("public attach queried client attachment state")
	}
}
