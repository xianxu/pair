package terminalqualify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunCaseDetectsMismatchAndPartitions(t *testing.T) {
	tc := Case{ID: "test", Capability: "test", Input: "abc", Expected: Observation{"value": "ok"}, Split: true}
	calls := 0
	execute := func(ctx context.Context, c Case, chunks []string) (Observation, error) {
		calls++
		if len(chunks) > 1 {
			return Observation{"value": "bad"}, nil
		}
		return Observation{"value": "ok"}, nil
	}
	got, err := RunCase(context.Background(), tc, execute)
	if err != nil || got.Status != Fail || calls != 2 {
		t.Fatalf("%+v %v calls=%d", got, err, calls)
	}
	calls = 0
	got, err = RunCase(context.Background(), tc, func(ctx context.Context, c Case, chunks []string) (Observation, error) {
		calls++
		return Observation{"value": "ok"}, nil
	})
	if err != nil || got.Status != Pass || calls != 4 {
		t.Fatalf("%+v %v calls=%d", got, err, calls)
	}
}
func TestRunCaseDoesNotConfuseInfrastructureWithConformance(t *testing.T) {
	tc := Case{ID: "test", Expected: Observation{"x": "y"}}
	_, err := RunCase(context.Background(), tc, func(context.Context, Case, []string) (Observation, error) { return nil, errors.New("broken pipe") })
	if err == nil {
		t.Fatal("IO failure swallowed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RunCase(ctx, tc, func(context.Context, Case, []string) (Observation, error) {
		t.Fatal("called after cancellation")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}
func TestRunCaseUncoveredNeverExecutes(t *testing.T) {
	got, err := RunCase(context.Background(), Case{ID: "pending", Uncovered: "adapter required"}, func(context.Context, Case, []string) (Observation, error) { t.Fatal("executed"); return nil, nil })
	if err != nil || got.Status != NotCovered {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRunCaseRejectsOversizeBeforePartitioning(t *testing.T) {
	_, err := RunCase(context.Background(), Case{ID: "oversize", Input: strings.Repeat("x", 65537), Expected: Observation{"x": "y"}, Split: true}, func(context.Context, Case, []string) (Observation, error) {
		t.Fatal("oversize fixture executed")
		return nil, nil
	})
	if err == nil {
		t.Fatal("oversize fixture accepted")
	}
}
