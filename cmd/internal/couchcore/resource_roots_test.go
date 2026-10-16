package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"testing"
)

func TestResourceRootEvidenceSupersedesBeforeProbe(t *testing.T) {
	c := NewResourceRootCache()
	a := ThreadAddress{RepoScope: "scope", Tag: "thread"}
	old := c.Begin(a, "old")
	fresh := c.Begin(a, "new")
	owner := launcher.SessionOwnerObservation{State: launcher.SessionOwnerOwned, Name: "new", Server: launcher.SessionServerIdentity{PID: 12, Identity: "start", Session: "new"}}
	c.Complete(a, fresh, owner, nil)
	owner.Name = "old"
	c.Complete(a, old, owner, nil)
	got := c.Snapshot()[0]
	if !got.Known || !got.Present || got.Session != "new" || got.Revision != fresh {
		t.Fatalf("stale completion replaced proof: %+v", got)
	}
	revision := c.Begin(a, "new")
	c.Complete(a, revision, owner, errors.New("unreadable"))
	got = c.Snapshot()[0]
	if got.Known || got.Present || got.Server.PID != 0 {
		t.Fatalf("failed proof retained authority: %+v", got)
	}
	revision = c.Begin(a, "new")
	c.Complete(a, revision, launcher.SessionOwnerObservation{State: launcher.SessionOwnerAbsent}, nil)
	got = c.Snapshot()[0]
	if !got.Known || got.Present {
		t.Fatalf("absence unresolved: %+v", got)
	}
}
func TestResourceRootsSharedByCheckerCopies(t *testing.T) {
	c := NewScopedThreadArtifactCollisionChecker(t.TempDir())
	if c.ResourceRootEvidenceCache() == nil || c.ResourceRootEvidenceCache() != c.ResourceRoots {
		t.Fatal("cache not shared")
	}
}

func TestResourceRootObserverUsesAuthenticatedNamedProbe(t *testing.T) {
	root := t.TempDir()
	a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "thread"}
	c := NewScopedThreadArtifactCollisionChecker(root)
	c.OwnerProbe = testOwnerProbe(root, "pair-owned", a)
	copy := c
	binding, err := copy.NamedPairSessionContext(context.Background(), a, "pair-owned")
	if err != nil || !binding.Present {
		t.Fatalf("probe: %+v %v", binding, err)
	}
	got := c.ResourceRoots.Snapshot()[0]
	if !got.Known || !got.Present || got.Server != binding.Owner.Server {
		t.Fatalf("lost server proof: %+v", got)
	}
	_, err = c.NamedPairSessionContext(context.Background(), a, "pair-absent")
	if err != nil {
		t.Fatal(err)
	}
	got = c.ResourceRoots.Snapshot()[0]
	if !got.Known || got.Present || got.Server.PID != 0 {
		t.Fatalf("absence left root: %+v", got)
	}
}
func TestResourceRootCacheBoundAndForgottenCompletion(t *testing.T) {
	c := NewResourceRootCache()
	for i := 0; i < resourceRootCacheLimit; i++ {
		if c.Begin(ThreadAddress{RepoScope: "s", Tag: ThreadTag(fmt.Sprint(i))}, "session") == 0 {
			t.Fatal("early bound")
		}
	}
	a := ThreadAddress{RepoScope: "s", Tag: "extra"}
	if c.Begin(a, "session") != 0 {
		t.Fatal("unbounded cache")
	}
	c.Forget(ThreadAddress{RepoScope: "s", Tag: "0"})
	rev := c.Begin(a, "session")
	if rev == 0 {
		t.Fatal("forgotten capacity unavailable")
	}
	c.Forget(a)
	c.Complete(a, rev, launcher.SessionOwnerObservation{State: launcher.SessionOwnerAbsent}, nil)
	if len(c.Snapshot()) != resourceRootCacheLimit-1 {
		t.Fatal("late completion resurrected root")
	}
}
