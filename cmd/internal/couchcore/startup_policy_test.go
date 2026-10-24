package couchcore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// startupPolicySnapshot supplies policy inputs directly: classification neither
// reads nor writes the thread store. Store-backed observation is exercised below.
func startupPolicySnapshot(t *testing.T, others int) (ThreadSnapshot, ThreadAddress) {
	t.Helper()
	var snapshot ThreadSnapshot
	for i := 0; i <= others; i++ {
		path, suffix := "/repo", "cwd"
		if i > 0 {
			path, suffix = fmt.Sprintf("/other-%02d", i-1), fmt.Sprintf("o%02d", i-1)
		}
		scope, err := launcher.ResolveRepoScope(path)
		if err != nil {
			t.Fatal(err)
		}
		record := validThreadRecord(t)
		record.Address = ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(fmt.Sprintf("couch-%016x", threadCounter(suffix)))}
		record.StartingPath, record.WorkingPath = path, path
		record.Layout = Layout3
		snapshot.Records = append(snapshot.Records, record)
	}
	return snapshot, snapshot.Records[0].Address
}

func TestStartupClassifiesOnlyPositiveSurvivors(t *testing.T) {
	snapshot, hub := startupPolicySnapshot(t, 1)
	for _, state := range []SessionState{SessionUnresolved, SessionAbsent, SessionOrphaned, SessionPresent} {
		presence := map[ThreadAddress]SessionObservation{}
		for _, record := range snapshot.Records {
			presence[record.Address] = SessionObservation{State: SessionAbsent}
		}
		presence[hub] = SessionObservation{State: state, Name: "selected"}
		observation := ClassifyStartup(snapshot, presence, []string{"/repo", "/missing"})
		want := StartupUnknown
		if state == SessionAbsent {
			want = StartupFullShutdown
		}
		if state == SessionPresent {
			want = StartupNormal
		}
		if observation.Kind != want {
			t.Fatalf("%v: kind %v, want %v", state, observation.Kind, want)
		}
		observedHub := observation.Hubs["/repo"]
		if observedHub.Survived != (state == SessionPresent) || observedHub.Survived && (len(observedHub.Candidates) != 1 || observedHub.Candidates[0].Record.Address != hub) {
			t.Fatalf("%v: wrong hub %+v", state, observation)
		}
		if observation.Hubs["/missing"].Survived || len(observation.Hubs["/missing"].Candidates) != 0 {
			t.Fatal("invented missing hub")
		}
	}
	// A surviving fleet session dominates uncertainty, without granting a missing
	// hub automation or attachment authority.
	observation := ClassifyStartup(ThreadSnapshot{Records: snapshot.Records, Unreadable: []ThreadAddress{{RepoScope: "other", Tag: "broken"}}}, map[ThreadAddress]SessionObservation{hub: {State: SessionPresent, Name: "selected"}}, []string{"/repo", "/missing"})
	if observation.Kind != StartupNormal || !observation.ParkedUnknown || observation.Hubs["/missing"].Survived {
		t.Fatalf("positive survivor: %+v", observation)
	}
}

func TestStartupObservationBulkAndReadOnly(t *testing.T) {
	for _, n := range []int{20, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			env, hub := startupFixture(t, n, "")
			env.Artifacts.SetSessionPresence(hub, SessionObservation{State: SessionPresent, Name: "selected"})
			before, _ := env.Couch.Threads.PreviewSnapshot()
			observation, err := env.Couch.ObserveStartupContext(context.Background(), []string{"/repo", "/missing"})
			if err != nil || observation.Kind != StartupNormal || env.Artifacts.SessionPresenceQueries() != 1 || env.Artifacts.BindingResolutions() != 0 || env.Artifacts.DetachedQueries() != 0 || len(env.Runner.Ops) != 0 {
				t.Fatalf("observation=%+v err=%v presence=%d", observation, err, env.Artifacts.SessionPresenceQueries())
			}
			after, _ := env.Couch.Threads.PreviewSnapshot()
			if !reflect.DeepEqual(before, after) {
				t.Fatal("observation mutated records")
			}
		})
	}
}

func TestStartupObservationFailureIsDiagnosticAndCancellationIsError(t *testing.T) {
	env, _ := startupFixture(t, 0, Layout3)
	env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error { return errors.New("listing unavailable") }
	observation, err := env.Couch.ObserveStartupContext(context.Background(), []string{"/repo"})
	if err != nil || observation.Kind != StartupUnknown || observation.Diagnostic == nil {
		t.Fatalf("unknown: %+v %v", observation, err)
	}
	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/missing"})
	if err != nil || result.Handle != nil {
		t.Fatalf("missing: %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = env.Couch.ObserveStartupContext(ctx, []string{"/repo"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation=%v", err)
	}
}

func TestStartupNumberedHubUsesOnlyItsCurrentSlotRecord(t *testing.T) {
	snapshot, hub := startupPolicySnapshot(t, 0)
	slot := conventionalSlot("/primary", 1)
	scope, _ := launcher.ResolveRepoScope(slot.WorktreeRoot)
	hub.RepoScope = scope.Key
	snapshot.Records[0].Address = hub
	snapshot.Slots = []SlotInventoryObservation{{Identity: slot}}
	presence := map[ThreadAddress]SessionObservation{hub: {State: SessionPresent, Name: "historical"}}
	observation := ClassifyStartup(snapshot, presence, []string{slot.WorktreeRoot})
	if observation.Hubs[slot.WorktreeRoot].Survived || len(observation.Hubs[slot.WorktreeRoot].Candidates) != 0 {
		t.Fatal("historical record substituted for empty current numbered slot")
	}
	snapshot.Slots[0].Address = hub
	observation = ClassifyStartup(snapshot, presence, []string{slot.WorktreeRoot, "/rep"})
	observedHub := observation.Hubs[slot.WorktreeRoot]
	if !observedHub.Survived || len(observedHub.Candidates) != 1 || observedHub.Candidates[0].Record.Address != hub || observation.Hubs["/rep"].Survived {
		t.Fatalf("exact numbered hub: %+v", observation)
	}
}

func TestStartupObservationDistinguishesUnreadableFromMissingHub(t *testing.T) {
	snapshot, hub := startupPolicySnapshot(t, 0)
	snapshot.Records = nil
	snapshot.Unreadable = []ThreadAddress{hub}
	observation := ClassifyStartup(snapshot, nil, []string{"/repo", "/missing"})
	if !observation.Hubs["/repo"].Unknown || observation.Hubs["/missing"].Unknown || observation.Diagnostic == nil {
		t.Fatalf("hub uncertainty=%+v", observation)
	}
}

func TestStartupObservationPropagatesResolverCancellation(t *testing.T) {
	env, _ := startupFixture(t, 0, Layout3)
	env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error { return context.Canceled }
	_, err := env.Couch.ObserveStartupContext(context.Background(), []string{"/repo"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolver cancellation=%v", err)
	}
}

func TestStartupDoesNotAttributeSharedSessionToTwoRecords(t *testing.T) {
	snapshot, hub := startupPolicySnapshot(t, 1)
	presence := map[ThreadAddress]SessionObservation{}
	for _, record := range snapshot.Records {
		presence[record.Address] = SessionObservation{State: SessionPresent, Name: "contested"}
	}
	observation := ClassifyStartup(snapshot, presence, []string{"/repo"})
	if observation.Kind != StartupUnknown || observation.Hubs["/repo"].Survived {
		t.Fatalf("contested session attributed to %v: %+v", hub, observation)
	}
}

func TestStartupClassificationIsIndependentOfRecordOrder(t *testing.T) {
	snapshot, _ := startupPolicySnapshot(t, 1)
	states := []SessionState{SessionAbsent, SessionUnresolved, SessionOrphaned, SessionPresent}
	for _, first := range states {
		for _, second := range states {
			presence := map[ThreadAddress]SessionObservation{
				snapshot.Records[0].Address: {State: first, Name: "first"},
				snapshot.Records[1].Address: {State: second, Name: "second"},
			}
			got := ClassifyStartup(snapshot, presence, []string{"/repo", "/other-00"})
			want := StartupUnknown
			if first == SessionPresent || second == SessionPresent {
				want = StartupNormal
			} else if first == SessionAbsent && second == SessionAbsent {
				want = StartupFullShutdown
			}
			if got.Kind != want {
				t.Fatalf("%v/%v: %+v", first, second, got)
			}
			reversed := snapshot
			reversed.Records = []ThreadRecord{snapshot.Records[1], snapshot.Records[0]}
			if again := ClassifyStartup(reversed, presence, []string{"/repo", "/other-00"}); !reflect.DeepEqual(got, again) {
				t.Fatalf("record order changed classification: %+v vs %+v", got, again)
			}
		}
	}
}

func TestStartupInvalidHubAttributionRemainsUnknown(t *testing.T) {
	observation := ClassifyStartup(ThreadSnapshot{}, nil, []string{""})
	if observation.Kind != StartupUnknown || !observation.Hubs[""].Unknown || !observation.ParkedUnknown {
		t.Fatalf("unattributable hub: %+v", observation)
	}
}

func TestStartupUnknownSelectedSessionLeavesCallerUsable(t *testing.T) {
	env, _ := startupFixture(t, 0, Layout3)
	before, _ := env.Couch.Threads.PreviewSnapshot()
	env.Artifacts.ListedSessionHook = func(ThreadAddress) error { return errors.New("listing unavailable") }
	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil || result.Handle != nil || len(env.Runner.Ops) != 0 || env.Artifacts.BindingResolutions() != 0 {
		t.Fatalf("unknown selected session: %+v %v", result, err)
	}
	after, _ := env.Couch.Threads.PreviewSnapshot()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unknown session mutated retained records")
	}
}

// Fresh is a proof of no retained history, rather than a guess from absence.
func TestStartupFreshRequiresReadableZeroHistory(t *testing.T) {
	snapshot, _ := startupPolicySnapshot(t, 1)
	for n := 0; n <= len(snapshot.Records); n++ {
		for _, unreadable := range []bool{false, true} {
			for _, slotUnknown := range []bool{false, true} {
				for _, state := range []SessionState{SessionUnresolved, SessionAbsent, SessionOrphaned, SessionPresent} {
					for _, blankName := range []bool{false, true} {
						for _, sharedName := range []bool{false, true} {
							name := fmt.Sprintf("records=%d/unreadable=%t/slot=%t/state=%v/blank=%t/shared=%t", n, unreadable, slotUnknown, state, blankName, sharedName)
							t.Run(name, func(t *testing.T) {
								input := ThreadSnapshot{Records: append([]ThreadRecord(nil), snapshot.Records[:n]...)}
								if unreadable {
									input.Unreadable = []ThreadAddress{{RepoScope: "unknown", Tag: "unreadable"}}
								}
								if slotUnknown {
									input.Slots = []SlotInventoryObservation{{Err: errors.New("slot unavailable")}}
								}
								presence := map[ThreadAddress]SessionObservation{}
								for i, record := range input.Records {
									session := fmt.Sprintf("session-%d", i)
									if sharedName {
										session = "shared"
									}
									if blankName {
										session = ""
									}
									presence[record.Address] = SessionObservation{State: state, Name: session}
								}
								got := ClassifyStartup(input, presence, []string{"/repo", "/other-00"})
								// This independent authority predicate uses only absence of history
								// and read failures: session names and order cannot create fresh proof.
								fresh := n == 0 && !unreadable && !slotUnknown
								if (got.Kind == StartupKind("fresh")) != fresh {
									t.Fatalf("kind %q, fresh permitted=%t", got.Kind, fresh)
								}
								if fresh && (got.ParkedUnknown || got.ParkedCount != 0 || got.Diagnostic != nil) {
									t.Fatalf("fresh carried uncertainty/history: %+v", got)
								}
								// Reordering retained evidence and the requested hubs may change only
								// the display order of MissingHubs, never attribution or classification.
								for i, j := 0, len(input.Records)-1; i < j; i, j = i+1, j-1 {
									input.Records[i], input.Records[j] = input.Records[j], input.Records[i]
								}
								reversed := ClassifyStartup(input, presence, []string{"/other-00", "/repo"})
								if got.Kind != reversed.Kind || got.ParkedUnknown != reversed.ParkedUnknown || got.ParkedCount != reversed.ParkedCount || !reflect.DeepEqual(got.Hubs, reversed.Hubs) {
									t.Fatalf("input order changed classification: %+v vs %+v", got, reversed)
								}
							})
						}
					}
				}
			}
		}
	}
}
