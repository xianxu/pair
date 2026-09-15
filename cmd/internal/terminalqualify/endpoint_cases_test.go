package terminalqualify

import (
	"context"
	"testing"
)

func TestEndpointCasesUseRealIntegrationAndLiteralEvidence(t *testing.T) {
	for _, c := range EndpointCases() {
		t.Run(c.ID, func(t *testing.T) {
			if c.Target != "endpoint" || c.Integration == nil || len(c.Expected) == 0 || c.Uncovered != "" {
				t.Fatalf("invalid endpoint case %+v", c)
			}
			result, err := RunCase(context.Background(), c, executeCandidate)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != Pass {
				t.Fatalf("%s: %s; observed=%+v", result.Status, result.Detail, result.Observed)
			}
			if result.Target != "endpoint" || len(result.Expected) == 0 || len(result.Observed) == 0 {
				t.Fatalf("missing attributed evidence %+v", result)
			}
		})
	}
}

func TestEndpointQualificationPreservesOriginalRequirements(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Cases() {
		if seen[c.ID] {
			t.Fatalf("duplicate %s", c.ID)
		}
		seen[c.ID] = true
	}
	for _, c := range append(append(ScreenCases(), InputCases()...), Coverage()...) {
		if !seen[c.ID] {
			t.Fatalf("removed obligation %s", c.ID)
		}
	}
	for _, id := range []string{"wrapper-composition", "live-display-selection", "terminfo-profile", "clipboard-policy", "notification-origin"} {
		for _, c := range Cases() {
			if c.ID == id && c.Uncovered == "" {
				t.Fatalf("prematurely qualified consumer obligation %s", id)
			}
		}
	}
}

func TestIntegrationDispatchCannotFabricatePass(t *testing.T) {
	called := false
	c := Case{ID: "integration-dispatch", Target: "endpoint", Expected: Observation{"value": "literal"}, Integration: func(context.Context) (Observation, error) { called = true; return Observation{"value": "wrong"}, nil }}
	result, err := RunCase(context.Background(), c, executeCandidate)
	if err != nil || !called || result.Status != Fail || result.Observed["value"] != "wrong" || result.Target != "endpoint" {
		t.Fatalf("dishonest result %+v %v", result, err)
	}
}
