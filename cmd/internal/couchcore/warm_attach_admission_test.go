package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

type admissionListedArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	read  func(context.Context, int) error
	reads int
}

func (a *admissionListedArtifacts) ListedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	a.reads++
	if err := a.read(ctx, a.reads); err != nil {
		return PairSessionBinding{}, err
	}
	return a.FakeThreadArtifactCollisionChecker.ListedPairSessionContext(ctx, address, name)
}

type admissionLifetimeRunner struct {
	Runner
	ctx context.Context
}

func (r *admissionLifetimeRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	r.ctx = ctx
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

func TestWarmAdmissionUnknownNeverRetriesIntoAuthority(t *testing.T) {
	for _, failedRead := range []int{1, 2, 3, 4} {
		t.Run(string(rune('0'+failedRead)), func(t *testing.T) {
			env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
			cause := errors.New("unknown terminal observation")
			artifacts := &admissionListedArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, read: func(_ context.Context, n int) error {
				if n == failedRead {
					return cause
				}
				return nil
			}}
			env.Couch.Artifacts = artifacts
			_, _, err := env.Couch.ResumeContext(context.Background(), before.Address)
			if !errors.Is(err, cause) || artifacts.reads != failedRead {
				t.Fatalf("reads=%d want=%d err=%v", artifacts.reads, failedRead, err)
			}
			if len(env.Couch.actorRegistry().Records()) != 0 {
				t.Fatal("unknown terminal admitted an actor")
			}
			if failedRead < 3 && len(env.Runner.Ops) != 0 {
				t.Fatalf("pre-spawn refusal started helper: %v", env.Runner.Ops)
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestWarmAdmissionSharesDeadlineWithoutCappingHelperLifetime(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	var first time.Time
	artifacts := &admissionListedArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, read: func(ctx context.Context, n int) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("terminal observation has no admission deadline")
			return nil
		}
		if n == 1 {
			first = deadline
			if left := time.Until(deadline); left <= 0 || left > 5*time.Second {
				t.Errorf("admission budget=%v", left)
			}
		} else if deadline.After(first) {
			t.Errorf("query %d reset admission deadline", n)
		}
		return nil
	}}
	env.Couch.Artifacts = artifacts
	runner := &admissionLifetimeRunner{Runner: env.Runner}
	env.Couch.Runner = runner
	_, h, err := env.Couch.ResumeContext(context.Background(), before.Address)
	if err != nil || h == nil {
		t.Fatalf("handle=%v err=%v", h, err)
	}
	if artifacts.reads != 4 {
		t.Fatalf("reads=%d want=4", artifacts.reads)
	}
	if runner.ctx == nil {
		t.Fatal("no helper lifetime context")
	}
	if _, capped := runner.ctx.Deadline(); capped {
		t.Fatal("admission deadline capped helper lifetime")
	}
	if runner.ctx.Err() != nil {
		t.Fatalf("helper context canceled after admission: %v", runner.ctx.Err())
	}
}

func TestWarmAdmissionPreservesEarlierCallerDeadline(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	want, _ := ctx.Deadline()
	artifacts := &admissionListedArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, read: func(query context.Context, n int) error {
		got, ok := query.Deadline()
		if !ok || !got.Equal(want) {
			t.Errorf("query %d deadline=%v want=%v", n, got, want)
		}
		return nil
	}}
	env.Couch.Artifacts = artifacts
	runner := &admissionLifetimeRunner{Runner: env.Runner}
	env.Couch.Runner = runner
	_, _, err := env.Couch.ResumeContext(ctx, before.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := runner.ctx.Deadline(); !got.Equal(want) {
		t.Fatalf("helper deadline changed: %v", got)
	}
	if runner.ctx.Err() != nil {
		t.Fatal("admission cancellation canceled caller lifetime")
	}
}

func TestWarmAdmissionNegativeConfirmationDoesNotRetryHelper(t *testing.T) {
	for _, failedRead := range []int{1, 2} {
		t.Run(string(rune('0'+failedRead)), func(t *testing.T) {
			env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
			artifacts := &admissionListedArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, read: func(_ context.Context, n int) error {
				if n == failedRead {
					env.Artifacts.SetPairSession(before.Address, session, false)
				}
				return nil
			}}
			env.Couch.Artifacts = artifacts
			_, _, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
			if err == nil || artifacts.reads != failedRead {
				t.Fatalf("reads=%d err=%v", artifacts.reads, err)
			}
			if len(env.Couch.actorRegistry().Records()) != 0 {
				t.Fatal("negative authority admitted actor")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}
