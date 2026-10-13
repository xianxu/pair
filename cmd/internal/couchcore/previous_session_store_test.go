package couchcore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func parkPreviousSessionSource(t *testing.T, store *ThreadStore, live ThreadRecord, identity ParkIdentity, capture *pairlifecycle.PreservedScrollback) ThreadRecord {
	t.Helper()
	begun, err := store.BeginPark(live.Address, live.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := store.FinalizePark(live.Address, begun.Revision, identity, 1, time.Unix(100, 0).UTC(), capture)
	if err != nil {
		t.Fatal(err)
	}
	return parked
}

func TestPreviousSessionStoreRequiresExactVerifiedSource(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"no verified park", "wrong nonce", "wrong process", "wrong PID", "wrong capture", "missing capture"} {
		t.Run(mode, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			live, identity, _ := createParkableThread(t, store, ns, "park-first")
			binding := &couchidentity.SessionBinding{Name: "pair-source", ScopeKey: live.Address.RepoScope, Tag: string(live.Address.Tag), StartNonce: "source-start", Legacy: true}
			var err error
			live, err = store.updateExistingThread(live.Address, live.Revision, func(r *ThreadRecord) error { r.SessionBinding = binding; return nil })
			if err != nil {
				t.Fatal(err)
			}
			capture := &pairlifecycle.PreservedScrollback{Agent: "codex", Token: "source-capture", Events: true}
			current := live
			if mode != "no verified park" {
				current = parkPreviousSessionSource(t, store, live, identity, capture)
			}
			previous := PreviousSession{Source: toPersistedParkIdentity(identity), Agent: "codex", Capture: pairlifecycle.ClonePreservedScrollback(capture), CapturedAt: time.Unix(100, 0).UTC()}
			switch mode {
			case "wrong nonce":
				previous.Source.Nonce = "other-park"
			case "wrong process":
				previous.Source.ProcessIdentity = "replacement"
			case "wrong PID":
				previous.Source.PID++
			case "wrong capture":
				previous.Capture.Token = "foreign-capture"
			case "missing capture":
				previous.Capture = nil

			}
			if _, err := store.RecordPreviousSession(current.Address, current.Revision, previous); err == nil {
				t.Fatal("accepted evidence unrelated to verified source")
			}
			after, err := NewThreadStore(ns).GetThread(current.Address)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, current) {
				t.Fatal("rejected previous evidence changed persisted thread")
			}
		})
	}
}

func TestPreviousSessionStoreRevisionAndDefensiveCopies(t *testing.T) {
	t.Parallel()
	store, ns := newTestThreadStore(t)
	live, identity, _ := createParkableThread(t, store, ns, "park-first")
	capture := &pairlifecycle.PreservedScrollback{Agent: "codex", Token: "source-capture", Events: true}
	parked := parkPreviousSessionSource(t, store, live, identity, capture)
	previous := previousSessionEvidence(live, parked)
	saved, err := store.RecordPreviousSession(parked.Address, parked.Revision, *previous)
	if err != nil {
		t.Fatal(err)
	}
	want := threadrecord.ClonePreviousSession(previous)
	if saved.Revision != parked.Revision+1 {
		t.Fatal("write did not advance revision")
	}
	previous.Capture.Token = "caller-mutated"
	saved.PreviousSession.Capture.Token = "returned-mutated"
	got, err := NewThreadStore(ns).GetThread(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.PreviousSession, want) {
		t.Fatalf("persisted evidence changed: %+v", got.PreviousSession)
	}
	_, err = store.RecordPreviousSession(parked.Address, parked.Revision, *want)
	var conflict *ThreadRevisionError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale writer error = %v", err)
	}
	// Both conversion directions must own their mutable descriptor fields.
	persisted := toPersistedThreadRecord(got)
	persisted.PreviousSession.Capture.Token = "persisted-mutated"
	if !reflect.DeepEqual(got.PreviousSession, want) {
		t.Fatal("to-persisted conversion aliases evidence")
	}
	conversionSource := toPersistedThreadRecord(got)
	converted := fromPersistedThreadRecord(conversionSource)
	converted.PreviousSession.Capture.Token = "converted-mutated"
	if !reflect.DeepEqual(conversionSource.PreviousSession, want) {
		t.Fatal("from-persisted conversion aliases evidence")
	}
}

func switchPreviousSession(t *testing.T, env *testEnv, address ThreadAddress, agent string) (SwitchAgentResult, error) {
	t.Helper()
	args := []string{}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), address, agent, &args)
	if err != nil {
		t.Fatal(err)
	}
	return env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: address, Agent: agent, Argv: args, AcceptedFingerprint: prepared.Fingerprint})
}

func TestPreviousSessionFailedLaunchRetryPreservesSource(t *testing.T) {
	t.Parallel()
	env, source := switchEnvWithLiveThread(t)
	env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
		return orientation.OrientationContext{Tag: string(r.Address.Tag), WorkingPath: r.WorkingPath, SourceAgent: "claude", SourceSession: "original-native", NativeTranscripts: []string{"/private/original.jsonl"}}, nil
	})
	env.Runner.FailNextStart(errors.New("target fork failed"))
	result, err := switchPreviousSession(t, env.testEnv, source.Address, "codex")
	if err == nil || result.Outcome != SwitchStartFailed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	failed, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if failed.PreviousSession == nil || failed.PreviousSession.Agent != "claude" || failed.VerifiedPark == nil {
		t.Fatalf("failed target lost source: %+v", failed)
	}
	want := threadrecord.ClonePreviousSession(failed.PreviousSession)
	// Retry sees no native session from the failed target. It must use the
	// evidence already persisted before the first target launch.
	env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
		return orientation.OrientationContext{Tag: string(r.Address.Tag), WorkingPath: r.WorkingPath, SourceAgent: "claude"}, nil
	})
	result, err = switchPreviousSession(t, env.testEnv, source.Address, "codex")
	if err != nil || result.Outcome != SwitchStarted {
		t.Fatalf("retry=%+v err=%v", result, err)
	}
	after, err := NewThreadStore(env.Couch.Threads.namespace).GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.VerifiedPark != nil || !reflect.DeepEqual(after.PreviousSession, want) {
		t.Fatalf("retry changed previous source: %+v", after)
	}
}

func TestPreviousSessionSuccessiveSourceParksChooseImmediatePrevious(t *testing.T) {
	t.Parallel()
	for _, separatePark := range []bool{false, true} {
		t.Run(map[bool]string{false: "park during switch", true: "separate park before switch"}[separatePark], func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
				return orientation.OrientationContext{Tag: string(r.Address.Tag), WorkingPath: r.WorkingPath, SourceAgent: RecordAgent(r), SourceSession: "native-" + RecordAgent(r)}, nil
			})
			first, err := switchPreviousSession(t, env.testEnv, source.Address, "codex")
			if err != nil {
				t.Fatal(err)
			}
			current, err := env.Couch.Threads.GetThread(source.Address)
			if err != nil {
				t.Fatal(err)
			}
			firstPrevious := threadrecord.ClonePreviousSession(current.PreviousSession)
			if firstPrevious == nil || firstPrevious.Agent != "claude" {
				t.Fatal("first switch lost claude source")
			}
			// Wire the second exact process into the same stateful lifecycle and store.
			env = withParkLifecycle(t, env.testEnv, current, first.Record.PID, first.Record.Identity)
			env.Couch.PairLifecycle.Nonce = func() (string, error) { return "park-second", nil }
			env.Artifacts.SetPairSession(current.Address, current.SessionBinding.Name, true)
			env.model.SetSession(current.SessionBinding.Name, true)
			if separatePark {
				parked, err := env.Couch.PairLifecycle.ParkExpected(context.Background(), current.Address, current.Revision)
				if err != nil {
					t.Fatal(err)
				}
				if parked.Thread.VerifiedPark == nil || !reflect.DeepEqual(parked.Thread.PreviousSession, firstPrevious) {
					t.Fatal("separate park must retain the old descriptor until the next switch")
				}
			}
			second, err := switchPreviousSession(t, env.testEnv, source.Address, "claude")
			if err != nil || second.Outcome != SwitchStarted {
				t.Fatalf("second=%+v err=%v", second, err)
			}
			after, err := env.Couch.Threads.GetThread(source.Address)
			if err != nil {
				t.Fatal(err)
			}
			previous := after.PreviousSession
			if previous == nil || previous.Agent != "codex" || previous.Source.PID != first.Record.PID || previous.Source.ProcessIdentity != first.Record.Identity || previous.Source == firstPrevious.Source {
				t.Fatalf("previous did not advance to immediate source: %+v", previous)
			}
		})
	}
}
