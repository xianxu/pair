package couchcore

import (
	"path/filepath"
	"testing"
)

func TestSlotLayoutPaths(t *testing.T) {
	l := NewSlotLayout("/f/pair", "/f/pair/.git", 3)
	for _, c := range []struct{ name, got, want string }{
		{"env", l.Env(), "/f/worktree/pair-slot3"},
		{"host", l.Host(), "/f/worktree/pair-slot3/pair"},
		{"store", l.Store(), "/f/worktree/pair-slot3/.couch"},
		{"saved-work", l.SavedWork(), "/f/worktree/pair-slot3/.couch/saved-work"},
		{"setup lock", l.SetupLock(), "/f/worktree/pair-slot3/.weave-setup.lock"},
		{"intent", l.Intent(), "/f/pair/.git/couch-workspaces/3/creation.json"},
		{"resting branch", l.RestingBranch(), "main-slot3"},
		{"resting ref", l.RestingRef(), "refs/heads/main-slot3"},
		{"marker", SetupMarkerPath("/f/pair/.git/worktrees/pair"), "/f/pair/.git/worktrees/pair/couch-setup-success.json"},
		{"attempt", SetupAttemptPath("/f/pair/.git/worktrees/pair"), "/f/pair/.git/worktrees/pair/couch-setup-attempt.json"},
		{"worktrees root", WorktreesRoot("/f"), "/f/worktree"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if got := LayoutOf(conventionalSlot("/f/pair", 3)); got != l {
		t.Fatalf("LayoutOf(conventionalSlot) = %+v, want %+v", got, l)
	}
	if RestingBranch(0) != "main" || RestingBranch(7) != "main-slot7" {
		t.Fatal("RestingBranch convention changed")
	}
}

func TestParseSlotPathRoundTrip(t *testing.T) {
	for _, n := range []int{1, 12} {
		l := NewSlotLayout("/f/pair", "/f/pair/.git", n)
		for _, p := range []string{l.Env(), l.Host(), l.Store(), l.SavedWork(), filepath.Join(l.Host(), "cmd", "x.go"), filepath.Join(l.Env(), "ariadne")} {
			primary, got, ok := ParseSlotPath(p)
			if !ok || primary != "/f/pair" || got != n {
				t.Errorf("ParseSlotPath(%q) = %q, %d, %v; want /f/pair, %d, true", p, primary, got, ok, n)
			}
		}
	}
	for _, p := range []string{
		"/f/pair",                      // a primary
		"/f/worktree",                  // the container
		"/f/worktree/pair-slot0/pair",  // :0 is never a slot path
		"/f/worktree/pair-slotX/pair",  // not a number
		"/f/worktree/pair-slot01/pair", // not canonical
		"/f/worktree/-slot1/x",         // no repository name
		"/f/other/pair-slot1/pair",     // not under worktree/
		"worktree/pair-slot1/pair",     // relative
	} {
		if primary, n, ok := ParseSlotPath(p); ok {
			t.Errorf("ParseSlotPath(%q) = %q, %d, true; want false", p, primary, n)
		}
	}
}

func TestParseEnvNameAndRestingBranch(t *testing.T) {
	for _, c := range []struct {
		repo, name string
		n          int
		ok         bool
	}{
		{"pair", "pair-slot2", 2, true},
		{"a-slot1", "a-slot1-slot2", 2, true},
		{"a", "a-slot1-slot2", 0, false},
		{"pair", "pair-slot0", 0, false},
		{"pair", "pair-slot+1", 0, false},
		{"pair", "parley-slot1", 0, false},
	} {
		n, ok := ParseEnvName(c.repo, c.name)
		if n != c.n || ok != c.ok {
			t.Errorf("ParseEnvName(%q, %q) = %d, %v; want %d, %v", c.repo, c.name, n, ok, c.n, c.ok)
		}
	}
	for _, c := range []struct {
		branch string
		n      int
		ok     bool
	}{{"main-slot4", 4, true}, {"main-slot04", 0, false}, {"main-slot0", 0, false}, {"main", 0, false}, {"feature", 0, false}} {
		n, ok := ParseRestingBranch(c.branch)
		if n != c.n || ok != c.ok {
			t.Errorf("ParseRestingBranch(%q) = %d, %v; want %d, %v", c.branch, n, ok, c.n, c.ok)
		}
	}
}
