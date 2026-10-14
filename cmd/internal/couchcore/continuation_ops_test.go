package couchcore

import "testing"

func TestContinuationOperationsDeclared(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"request-continuation", "continue-thread"} {
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
	// #522 retired the exits from a failed request and its delivery receipt:
	// a compaction that doesn't finish leaves one line and no operation.
	for _, name := range []string{"retry-continuation", "dismiss-continuation", "continuation-status"} {
		if _, ok := operationByName(name); ok {
			t.Errorf("%s is still declared", name)
		}
	}
}
