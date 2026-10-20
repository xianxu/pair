package repolint

import (
	"strings"
	"testing"
)

func TestLifecycleFailureAddressRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, function, body string
		want                      int
	}{
		{"partial nonzero override", "couchcore", "func lifecycleFailureForAttempt()", "if target.Address != (ThreadAddress{}) { f.Address = target.Address }", 1},
		{"observe call override", "couchcore", "func (a *OperationAttempt) ObserveCall()", "a.target.Address = t.Address", 1},
		{"fieldwise scope", "couchcore", "func lifecycleFailureTargetForCall()", "target.Address.RepoScope = scope", 1},
		{"parenthesized fieldwise tag", "couchcore", "func lifecycleFailureForAttempt()", "(f.Address).Tag = tag", 1},
		{"keyed partial construction", "couchcore", "func lifecycleFailureTargetForCall()", "target := LifecycleFailureTarget{Address: ThreadAddress{RepoScope: scope}}; _ = target", 1},
		{"atomic assignment", "couchcore", "func lifecycleFailureForAttempt()", "f.Address = completeFailureAddress(target.Address, f.Address)", 0},
		{"atomic keyed construction", "couchcore", "func lifecycleFailureTargetForCall()", "target := LifecycleFailureTarget{Address: completeFailureAddress(address, ThreadAddress{})}; _ = target", 0},
		{"zero cannot erase captured address", "couchcore", "func lifecycleFailureForAttempt()", "f.Address = ThreadAddress{}", 1},
		{"parenthesized atomic helper", "couchcore", "func lifecycleFailureForAttempt()", "f.Address = (completeFailureAddress)(target.Address, f.Address)", 0},
		{"wrong helper arity", "couchcore", "func lifecycleFailureForAttempt()", "f.Address = completeFailureAddress(target.Address)", 1},
		{"other owner", "couchcore", "func cloneLaunchFailure()", "f.Address = target.Address", 0},
		{"unrelated ObserveCall", "couchcore", "func (a *Other) ObserveCall()", "a.target.Address = t.Address", 0},
		{"other package", "other", "func lifecycleFailureForAttempt()", "f.Address = target.Address", 0},
		{"literal only", "couchcore", "func lifecycleFailureForAttempt()", "_ = \"f.Address = target.Address\"", 0},
		{"reasoned suppression", "couchcore", "func lifecycleFailureForAttempt()", "//lint:allow lifecycle-failure-address -- test controlled legacy evidence\nf.Address = target.Address", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\n" + tc.function + " {\n" + tc.body + "\n}\n")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/address.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("count=%d err=%v findings=%+v want=%d", count, err, findings, tc.want)
			}
			for _, found := range findings {
				if found.Finding.Rule != "lifecycle-failure-address" || !strings.Contains(found.Finding.Message, "pair#714") {
					t.Fatalf("unexpected finding: %+v", found)
				}
			}
		})
	}
}
