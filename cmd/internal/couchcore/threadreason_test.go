package couchcore

import "testing"

// The vocabulary guards itself where the wording now lives, so all three
// renderers are covered by one check. Go cannot verify a switch is exhaustive;
// iterating the enumeration is what stands in for that.
func TestEveryReasonHasADistinctOperatorLabel(t *testing.T) {
	seen := map[string]ThreadReason{}
	for _, reason := range AllThreadReasons() {
		label := reason.Label()
		if label == "" || label == string(reason) {
			t.Errorf("reason %q has no operator label of its own (got %q)", reason, label)
			continue
		}
		if other, clash := seen[label]; clash {
			t.Errorf("reasons %q and %q both render %q", reason, other, label)
		}
		seen[label] = reason
	}
}
