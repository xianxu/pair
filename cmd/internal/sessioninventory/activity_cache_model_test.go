package sessioninventory

import (
	"context"
	"errors"
	"testing"
)

func TestAdvanceActivityCacheOnlyLoadCanCommit(t *testing.T) {
	for _, initial := range []activityPhase{activityAbsent, activityReady} {
		if got := advanceActivityCache(initial, activityCommit); got != initial {
			t.Fatal("commit without load")
		}
	}
	s := advanceActivityCache(activityReady, activityBegin)
	if s != activityLoading {
		t.Fatal("not loading")
	}
	if advanceActivityCache(s, activityCommit) != activityReady {
		t.Fatal("stable load not ready")
	}
	if advanceActivityCache(s, activityDiscard) != activityAbsent {
		t.Fatal("canceled load retained success")
	}
}
func TestActivityDependenciesMatchRejectsChangedAndMissingInputs(t *testing.T) {
	a := Artifact{StorageRoot: "native", RelativePath: "root"}
	fingerprint := ArtifactFingerprint{StableFileID: "id", MutationToken: "ctime:1", Size: 10}
	want := activityEvidence{catalogPresent: true, catalog: fingerprint, files: map[Artifact]activityDependency{a: {artifact: a, present: true, fingerprint: fingerprint}}}
	same := activityEvidence{catalogPresent: true, catalog: fingerprint, files: map[Artifact]activityDependency{a: {artifact: a, present: true, fingerprint: fingerprint}}}
	if !activityDependenciesMatch(want, same) {
		t.Fatal("equal observations cannot reuse")
	}
	mutated := fingerprint
	mutated.MutationToken = "ctime:2"
	same.files[a] = activityDependency{artifact: a, present: true, fingerprint: mutated}
	if activityDependenciesMatch(want, same) {
		t.Fatal("same-size mutation reused")
	}
	delete(same.files, a)
	if activityDependenciesMatch(want, same) {
		t.Fatal("missing observation reused")
	}
	same = want
	same.catalogPresent = false
	if activityDependenciesMatch(want, same) {
		t.Fatal("catalog removal reused")
	}
}
func TestActivityCacheBoundedAndWaiterCancellation(t *testing.T) {
	c := NewActivityCache()
	ctx := t.Context()
	for i := 0; i < 129; i++ {
		slot, release, err := c.acquire(ctx, activityKey{tag: string(rune(i))})
		if err != nil || slot == nil {
			t.Fatal("missing slot")
		}
		release()
	}
	held, release, err := c.acquire(ctx, activityKey{tag: "held"})
	if err != nil || held == nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { _, finish, err := c.acquire(canceled, activityKey{tag: "held"}); finish(); done <- err }()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter cancellation=%v", err)
	}
	release()
	if len(c.slots) != 128 {
		t.Fatalf("entries=%d", len(c.slots))
	}
}
