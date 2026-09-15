package terminalqualify

import (
	"context"
	"testing"
)

func TestPresenterCasesUseActualProductionPaths(t *testing.T) {
	for _, c := range PresenterCases() {
		t.Run(c.ID, func(t *testing.T) {
			if c.Target != "presenter" || c.Integration == nil || len(c.Expected) == 0 {
				t.Fatalf("invalid presenter fixture %+v", c)
			}
			result, err := RunCase(context.Background(), c, executeCandidate)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != Pass {
				t.Fatalf("%s: %s; observed=%+v", result.Status, result.Detail, result.Observed)
			}
		})
	}
}
