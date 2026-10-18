package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type retryAttachRunner struct {
	Runner
	attempts  int
	failures  int
	onFailure func()
}

func (r *retryAttachRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	r.attempts++
	if r.attempts <= r.failures {
		if r.onFailure != nil {
			r.onFailure()
		}
		return nil, errors.New("temporary attachment failure")
	}
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

func TestWarmAttachmentRetriesOnceAcrossExplicitEntryPoints(t *testing.T) {
	for _, entry := range []string{"hub", "resume", "selection"} {
		t.Run(entry, func(t *testing.T) {
			env, address := startupFixture(t, 0, "")
			before, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				t.Fatal(err)
			}
			runner := &retryAttachRunner{Runner: env.Runner, failures: 1}
			env.Couch.Runner = runner
			var handle Handle
			switch entry {
			case "hub":
				result, e := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
				handle, err = result.Handle, e
			case "resume":
				_, handle, err = env.Couch.ResumeContextWith(context.Background(), address, ResumeOptions{})
			default:
				result, e := env.Couch.ResumeTarget(context.Background(), ResumeTarget{Address: address})
				err = e
				if start, ok := result.(StartResult); ok {
					handle = start.Handle
				}
			}
			if err != nil || handle == nil || runner.attempts != 2 {
				t.Fatalf("attempts=%d handle=%v err=%v", runner.attempts, handle, err)
			}
			if len(env.Couch.actorRegistry().Records()) != 1 {
				t.Fatal("retry registered duplicate clients")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestWarmAttachmentStopsAfterSecondFailureWithReason(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &retryAttachRunner{Runner: env.Runner, failures: 3}
	env.Couch.Runner = runner
	_, handle, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if runner.attempts != 2 || err == nil || !strings.Contains(err.Error(), "temporary attachment failure") || !strings.Contains(err.Error(), "2 attempts") || handle != nil {
		t.Fatalf("attempts=%d handle=%v err=%v", runner.attempts, handle, err)
	}
	if len(env.Runner.Ops) != 0 || len(env.Couch.actorRegistry().Records()) != 0 {
		t.Fatal("failed retry created an actor")
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentCancellationDuringBackoffDoesNotRetry(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &retryAttachRunner{Runner: env.Runner, failures: 1}
	env.Couch.Runner = runner
	ctx, cancel := context.WithCancel(context.Background())
	runner.onFailure = func() { time.AfterFunc(10*time.Millisecond, cancel) }
	_, _, err := env.Couch.attachListedTerminal(ctx, before, session, ResumeOptions{})
	if !errors.Is(err, context.Canceled) || runner.attempts != 1 {
		t.Fatalf("attempts=%d err=%v", runner.attempts, err)
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentRetryRechecksSameLiveTerminal(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &retryAttachRunner{Runner: env.Runner, failures: 1}
	env.Couch.Runner = runner
	runner.onFailure = func() {
		env.Artifacts.SetDetachedSession(before.Address, "")
		env.Artifacts.SetPairSession(before.Address, session, false)
	}
	_, _, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err == nil || runner.attempts != 1 || !strings.Contains(err.Error(), "no longer uniquely listed live") {
		t.Fatalf("attempts=%d err=%v", runner.attempts, err)
	}
	if len(env.Runner.Ops) != 0 || len(env.Couch.actorRegistry().Records()) != 0 {
		t.Fatal("retry created helper without live proof")
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentRetryCleansFailedClientBeforeStartingAnother(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &retryAttachRunner{Runner: env.Runner}
	env.Couch.Runner = runner
	var first string
	env.Runner.AfterAcknowledge = func(id string) error {
		if first == "" {
			first = id
			env.Runner.SetDiesOn(id, TermSignal, 1)
			return errors.New("attachment failed after acknowledgment")
		}
		if env.Runner.Child(first).alive {
			t.Fatal("retry overlaps failed client")
		}
		return nil
	}
	_, handle, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err != nil || handle == nil || runner.attempts != 2 || env.Runner.Child(first).alive {
		t.Fatalf("attempts=%d handle=%v err=%v", runner.attempts, handle, err)
	}
	if len(env.Couch.actorRegistry().Records()) != 1 {
		t.Fatal("failed client retained registration")
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentCancellationAfterAcknowledgmentDoesNotRetry(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &retryAttachRunner{Runner: env.Runner}
	env.Couch.Runner = runner
	ctx, cancel := context.WithCancel(context.Background())
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetDiesOn(id, TermSignal, 1)
		cancel()
		return context.Canceled
	}
	_, handle, err := env.Couch.attachListedTerminal(ctx, before, session, ResumeOptions{})
	if !errors.Is(err, context.Canceled) || runner.attempts != 1 || handle == nil || handle.Alive() {
		t.Fatalf("attempts=%d handle=%v err=%v", runner.attempts, handle, err)
	}
	if len(env.Couch.actorRegistry().Records()) != 0 {
		t.Fatal("canceled attachment registered a client")
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentRetriesManagedSlotWithDeadIncarnationAndListedTerminal(t *testing.T) {
	for _, entry := range []string{"selection", "named-hub"} {
		t.Run(entry, func(t *testing.T) {
			env, _, before := liveSlotFixture(t)
			env.Proc.Kill(before.Incarnations[0].PID)
			env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
			runner := &retryAttachRunner{Runner: env.Runner, failures: 1}
			env.Couch.Runner = runner
			var handle Handle
			var actor ActorRecord
			var err error
			if entry == "named-hub" {
				start, e := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: before.StartingPath})
				handle, err, actor = start.Handle, e, start.Record
			} else {
				result, e := env.Couch.ResumeTarget(context.Background(), ResumeTarget{Path: before.StartingPath})
				err = e
				if start, ok := result.(StartResult); ok {
					handle, actor = start.Handle, start.Record
				}
			}
			if err != nil || handle == nil || runner.attempts != 2 {
				t.Fatalf("attempts=%d handle=%v err=%v", runner.attempts, handle, err)
			}
			if env.Artifacts.BindingResolutions() != 0 {
				t.Fatal("listed slot attachment required native proof")
			}
			assertWarmClientRepair(t, env, before, actor)
		})
	}
}

func TestWarmListingUnknownIsFinalAndHonorsCancellation(t *testing.T) {
	for _, mode := range []string{"transient", "persistent", "absent", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads := 0
			env.Artifacts.ListedSessionHook = func(ThreadAddress) error {
				reads++
				if mode == "cancel" {
					cancel()
					return context.Canceled
				}
				if mode == "persistent" || mode == "transient" && reads == 1 {
					return errors.New("listing temporarily unavailable")
				}
				return nil
			}
			if mode == "absent" {
				env.Artifacts.SetDetachedSession(before.Address, "")
			}
			_, listed, err := env.Couch.listedTerminal(ctx, before)
			switch mode {
			case "transient":
				if err == nil || listed || reads != 1 {
					t.Fatalf("reads=%d listed=%v err=%v", reads, listed, err)
				}
			case "persistent":
				if err == nil || reads != 1 {
					t.Fatalf("reads=%d err=%v", reads, err)
				}
			case "absent":
				if err != nil || listed || reads != 1 {
					t.Fatalf("reads=%d listed=%v err=%v", reads, listed, err)
				}
			default:
				if !errors.Is(err, context.Canceled) || reads != 1 {
					t.Fatalf("reads=%d err=%v", reads, err)
				}
			}
			if len(env.Runner.Ops) != 0 {
				t.Fatal("unknown listing launched a helper")
			}
		})
	}
}
