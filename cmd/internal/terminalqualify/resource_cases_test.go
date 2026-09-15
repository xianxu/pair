package terminalqualify

import (
	"context"
	"testing"
)

func TestResourceCasesObserveBoundedBackendStorage(t *testing.T) {
	for _, c := range ResourceCases() {
		result, err := RunCase(context.Background(), c, executeCandidate)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != Pass {
			t.Fatalf("%s: %s %+v", c.ID, result.Detail, result.Observed)
		}
	}
}
