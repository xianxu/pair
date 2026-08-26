package couchcore

import "testing"

func registryRecord(id string, tree Worktree) ActorRecord {
	return ActorRecord{ID: ActorID(id), Args: StartArgs{Worktree: tree}}
}

func TestRegistryInsertKeepsEveryLegacyCoTenantEnumerable(t *testing.T) {
	base := NewRegistry()
	next := base.Insert(registryRecord("couch-a", "/repo"))
	next = next.Insert(registryRecord("couch-b", "/repo"))
	if len(base.Get("/repo")) != 0 {
		t.Fatal("insert mutated its receiver")
	}
	if got := next.Get("/repo"); len(got) != 2 {
		t.Fatalf("co-tenants = %+v", got)
	}
}

func TestRegistryKeysFoldCaseAccordingToPlatform(t *testing.T) {
	reg := NewRegistry().Insert(registryRecord("couch-a", "/Users/x/repo"))
	reg = reg.Insert(registryRecord("couch-b", "/users/x/repo"))
	got := reg.Get("/Users/x/repo")
	if caseInsensitiveFS() && len(got) != 2 {
		t.Fatalf("case aliases did not share a display-cache key: %+v", got)
	}
	if !caseInsensitiveFS() && len(got) != 1 {
		t.Fatalf("distinct case-sensitive trees were conflated: %+v", got)
	}
}

func TestRegistryRemoveActorLeavesCoTenant(t *testing.T) {
	reg := NewRegistry().Insert(registryRecord("couch-a", "/repo"))
	reg = reg.Insert(registryRecord("couch-b", "/repo"))
	reg = reg.RemoveActor("/repo", "couch-a")
	got := reg.Get("/repo")
	if len(got) != 1 || got[0].ID != "couch-b" {
		t.Fatalf("remaining records = %+v", got)
	}
}

func TestRegistryUnregisterFreesTheDisplayKey(t *testing.T) {
	reg := NewRegistry().Insert(registryRecord("couch-a", "/repo"))
	if got := reg.Unregister("/repo").Get("/repo"); len(got) != 0 {
		t.Fatalf("unregistered records = %+v", got)
	}
}

func TestRecordsCarryOriginalCaseWorktree(t *testing.T) {
	w := Worktree("/Users/x/KBench")
	got := NewRegistry().Insert(registryRecord("couch-a", w)).Records()
	if len(got) != 1 || got[0].Args.Worktree != w {
		t.Fatalf("Records = %+v, want the unfolded path %q", got, w)
	}
}
