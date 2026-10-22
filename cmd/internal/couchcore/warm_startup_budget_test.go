package couchcore

import (
	"context"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

type warmBudgetRunner struct {
	Runner
	lifetime context.Context
	env      []string
	timeout  time.Duration
}

func (r *warmBudgetRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	r.lifetime, r.env, r.timeout = ctx, append([]string(nil), env...), timeout
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

func TestPublicWarmAttachmentTransportsCallerStartupBudget(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Live)
	runner := &warmBudgetRunner{Runner: env.Runner}
	env.Couch.Runner = runner
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		want, _ := ctx.Deadline()
		var confirmationDeadline time.Time
		env.Couch.registrationDeadline = func(parent context.Context, remaining time.Duration) (context.Context, context.CancelFunc) {
			confirmationDeadline, _ = parent.Deadline()
			if remaining != 27*time.Second {
				t.Errorf("post-ack remaining budget=%v want 27s", remaining)
			}
			return context.WithTimeout(parent, remaining)
		}
		env.Runner.BeforeAcknowledge = func(string) error {
			// A delayed helper must consume the same absolute deadline.
			<-time.After(time.Second)
			return nil
		}
		artifacts := &admissionListedArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, read: func(_ context.Context, n int) error {
			if n == 1 {
				<-time.After(2 * time.Second)
			}
			return nil
		}}
		env.Couch.Artifacts = artifacts
		_, handle, err := env.Couch.ResumeContextWith(ctx, before.Address, ResumeOptions{})
		if err != nil || handle == nil {
			t.Fatalf("warm attachment: handle=%v error=%v", handle, err)
		}
		defer func() {
			env.Runner.SetExited(handle.ID(), 0)
			_ = env.Runner.Terminal(handle.ID()).Close()
			_ = handle.Wait()
		}()
		var deadline string
		for _, entry := range runner.env {
			if raw, ok := strings.CutPrefix(entry, startupbudget.DeadlineEnv+"="); ok {
				if deadline != "" {
					t.Fatal("duplicate startup deadline")
				}
				deadline = raw
			}
		}
		if deadline != startupbudget.Encode(want) {
			t.Errorf("warm helper deadline=%q want=%q", deadline, startupbudget.Encode(want))
		}
		if runner.timeout != 28*time.Second {
			t.Errorf("helper ack budget=%v want remaining 28s", runner.timeout)
		}
		if !confirmationDeadline.Equal(want) {
			t.Errorf("attachment renewed startup deadline: got=%v want=%v", confirmationDeadline, want)
		}
		if runner.lifetime != ctx || runner.lifetime.Err() != nil {
			t.Fatal("startup completion changed or canceled interactive lifetime")
		}
		assertWarmRecordUnchanged(t, env, before)
	})
}
