package terminalqualify

import (
	"context"
	"errors"
	"fmt"
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

func TestRunCaseDetectsUnassertedSplitStateChanges(t *testing.T) {
	for _, key := range []string{"cell:7,2", "attrs:7,2", "link:7,2", "replies", "new-key"} {
		t.Run(key, func(t *testing.T) {
			tc := Case{ID: "split", Input: "abc", Split: true, Expected: Observation{"cursor": "1,0"}}
			got, err := RunCase(context.Background(), tc, func(_ context.Context, _ Case, chunks []string) (Observation, error) {
				obs := Observation{"cursor": "1,0", "cell:7,2": " ", "attrs:7,2": "0", "link:7,2": "", "replies": ""}
				if len(chunks) > 1 {
					obs[key] = "corrupt"
				}
				return obs, nil
			})
			if err != nil || got.Status != Fail || got.Comparison != "whole-split" || got.Observed[key] != "corrupt" {
				t.Fatalf("undetected %s: %+v %v", key, got, err)
			}
		})
	}
}

func TestRunCaseIncludesStructuredEvidence(t *testing.T) {
	for _, value := range []string{"expected", "different"} {
		got, err := RunCase(context.Background(), Case{ID: "evidence", Expected: Observation{"cell:0,0": "expected"}}, func(context.Context, Case, []string) (Observation, error) { return Observation{"cell:0,0": value}, nil })
		if err != nil || got.Expected["cell:0,0"] != "expected" || got.Observed["cell:0,0"] != value || got.Comparison != "literal" {
			t.Fatalf("%+v %v", got, err)
		}
	}
}

func TestRunCaseEvidenceRetainsRelevantActualFields(t *testing.T) {
	got, err := RunCase(context.Background(), Case{ID: "evidence", Expected: Observation{"underline:0,0": "1"}}, func(context.Context, Case, []string) (Observation, error) {
		obs := Observation{"underline:0,0": "1"}
		for i := 0; i < 100; i++ {
			obs[fmt.Sprintf("attrs:%d,0", i)] = "0"
		}
		return obs, nil
	})
	if err != nil || got.Observed["underline:0,0"] != "1" || !got.ObservedTruncated {
		t.Fatalf("%+v %v", got, err)
	}
}
