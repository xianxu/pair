package couchcore

import "testing"

func TestContinuationOperationsDeclared(t *testing.T) {
	for _, name := range []string{"request-continuation", "continue-thread", "retry-continuation", "continuation-status"} {
		op, ok := operationByName(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if op.Presentation != PresentationInternal {
			t.Fatalf("%s must be internal", name)
		}
		want := ExecuteLiveOwner
		if name == "request-continuation" {
			want = ExecuteDirectStore
		}
		if op.Execution != want {
			t.Fatalf("%s owner=%v", name, op.Execution)
		}
	}
}
