package sessioninventory

import "testing"

func TestSelectedPairArtifactScopes(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path, mode, current string
		wantScope, wantName string
		wantOK              bool
	}{
		{"ledger-work.jsonl", "current", "scope-a", "scope-a", "ledger-work.jsonl", true},
		{"repos/scope-a/ledger-work.jsonl", "all", "", "scope-a", "ledger-work.jsonl", true},
		{"repos/scope-b/ledger-work.jsonl", "current", "scope-a", "", "", false},
		{"repos/scope-a/nested/ledger-work.jsonl", "all", "", "", "", false},
		{"../ledger-work.jsonl", "all", "", "", "", false},
	} {
		scope, name, ok := selectedPairArtifact(test.path, test.mode, test.current)
		if scope != test.wantScope || name != test.wantName || ok != test.wantOK {
			t.Errorf("selectedPairArtifact(%q,%q,%q) = %q,%q,%v; want %q,%q,%v", test.path, test.mode, test.current, scope, name, ok, test.wantScope, test.wantName, test.wantOK)
		}
	}
}
