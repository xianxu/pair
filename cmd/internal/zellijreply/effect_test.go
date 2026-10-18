package zellijreply

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEffectChecksExistingTokenBeforeDispatch(t *testing.T) {
	for _, kind := range []string{"canceled", "expired"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if kind == "expired" {
				ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
				defer cancel()
			}
			called := false
			err := Effect(ctx, func() error { called = true; return nil })
			if !errors.Is(err, ctx.Err()) || called {
				t.Fatalf("effect dispatched after %s: called=%v err=%v", kind, called, err)
			}
		})
	}
}

func TestEffectPreservesIssuedResultAndStopsNextEffect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	issuedErr := errors.New("issued mutation result")
	calls := 0
	if err := Effect(ctx, func() error { calls++; cancel(); return issuedErr }); err != issuedErr {
		t.Fatalf("issued result=%v", err)
	}
	if err := Effect(ctx, func() error { calls++; return nil }); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("later effect dispatched: calls=%d error=%v", calls, err)
	}
}

func TestEffectDispatchesOneSuccessfulEffectSynchronously(t *testing.T) {
	calls := 0
	if err := Effect(context.Background(), func() error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}
