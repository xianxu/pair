package testcouch

import "testing"

func TestFleetOptionsRefuseBeforeAllocation(t *testing.T) {
	for _, workers := range []int{-1, 0, 9, 1000000} {
		if err := validateOptions(Options{Workers: workers}); err == nil {
			t.Fatalf("accepted workers=%d", workers)
		}
	}
	if err := validateOptions(Options{Workers: 3, AgentKind: "claude"}); err != nil {
		t.Fatal(err)
	}
	if err := validateOptions(Options{Workers: 3, AgentKind: "installed"}); err == nil {
		t.Fatal("allowed installed-agent fallback")
	}
}
