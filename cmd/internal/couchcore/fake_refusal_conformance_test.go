package couchcore

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The family this closes is `production-seam-only-tested-through-fake`, third
// occurrence in #256.
//
// The instance that bit, in M2: production's PairSession wrapped
// ErrPairSessionBindingAbsent with %w, and the FAKE returned a plain error with
// similar words. Every caller that branches on `errors.Is(err,
// ErrPairSessionBindingAbsent)` — archive's binding-absent hatch among them —
// was therefore untestable through the fake, and the branch that mattered most
// could never be entered. Nothing about fixing that one call site prevents the
// next, which is why this is derived rather than a list.
//
// The pairing is a filesystem convention, not a table: `X.go` next to
// `X_fake.go` is a seam and its double, so a new fake joins this guard by being
// named the way every existing one already is. If the convention changes, the
// count check below fails rather than silently covering nothing.
func TestFakesWrapEverySentinelTheirProductionSeamWraps(t *testing.T) {
	dir := filepath.Join(repoRootFrom(t), "cmd", "internal", "couchcore")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// `fmt.Errorf("%w ...", ErrSomething` — the wrap that makes errors.Is work.
	wrapped := regexp.MustCompile(`%w[^"]*",\s*(Err[A-Za-z0-9_]+)`)
	sentinelsIn := func(name string) map[string]bool {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, match := range wrapped.FindAllStringSubmatch(string(body), -1) {
			found[match[1]] = true
		}
		return found
	}

	pairs := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_fake.go") {
			continue
		}
		production := strings.TrimSuffix(name, "_fake.go") + ".go"
		if _, err := os.Stat(filepath.Join(dir, production)); err != nil {
			continue
		}
		pairs++
		fake := sentinelsIn(name)
		var missing []string
		for sentinel := range sentinelsIn(production) {
			if !fake[sentinel] {
				missing = append(missing, sentinel)
			}
		}
		sort.Strings(missing)
		if len(missing) != 0 {
			t.Errorf("%s wraps %v and %s does not; a caller that branches on errors.Is "+
				"cannot reach that branch through the fake, so the branch is untested",
				production, missing, name)
		}
	}
	if pairs < 3 {
		t.Fatalf("found %d seam/fake pairs; the naming convention changed and this guard stopped guarding", pairs)
	}
}

// The behavioural half, for the one sentinel that actually has consumers today.
// The source guard above proves the fake WRAPS it; this proves the fake can be
// driven into the state that returns it, which is what a caller's test needs.
func TestTheFakeCanProduceAnAbsentBindingErrorsIsCanSee(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0102030405060708"}
	fake := NewFakeThreadArtifactCollisionChecker()

	_, err := fake.PairSession(address)
	if err == nil {
		t.Fatal("a thread with no binding at all returned no error")
	}
	if !errors.Is(err, ErrPairSessionBindingAbsent) {
		t.Fatalf("errors.Is cannot see the sentinel in %v (%T); every caller's hatch is unreachable through the fake", err, err)
	}
}
