package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestDetachedSlotRootsClaimsBackgroundedCommandsNotInfrastructure(t *testing.T) {
	thread := ThreadAddress{RepoScope: "s", Tag: "t"}
	ws := WorkspaceIdentity{RepoIdentity: "/r/.git", WorktreeRoot: "/r"}
	slots := []ResourceSlot{{Slot: "pair:6", Thread: thread, Workspace: ws, OwnershipKnown: true}}
	table := []procutil.Process{
		{PID: 10, PPID: 1, Identity: "zsh-1", Name: "zsh"},     // detached background shell
		{PID: 11, PPID: 10, Identity: "sdlc-1", Name: "sdlc"},  // its local-check run
		{PID: 12, PPID: 1, Identity: "zj-1", Name: "zellij"},   // a relaunched thread's leftover server
		{PID: 13, PPID: 1, Identity: "other-1", Name: "zsh"},   // another slot's, or nobody's
		{PID: 14, PPID: 99, Identity: "child-1", Name: "zsh"},  // still has ancestry
		{PID: 15, PPID: 1, Identity: "codex-1", Name: "codex"}, // a detached agent stays an orphan candidate
		{PID: 20, PPID: 1, Identity: "zsh-2", Name: "zsh"},     // platform shell: environment withheld
		{PID: 21, PPID: 20, Identity: "sdlc-2", Name: "sdlc"},  // its readable child
		{PID: 22, PPID: 21, Identity: "test-2", Name: "couchcore.test"},
	}
	inherited := map[int]ThreadAddress{10: thread, 12: thread, 14: thread, 15: thread, 21: thread, 22: thread, 13: {RepoScope: "s", Tag: "gone"}}
	got := DetachedSlotRoots(table, slots, func(p procutil.Process) (ThreadAddress, bool) {
		a, ok := inherited[p.PID]
		return a, ok
	})
	roots := got[0].Roots
	if len(roots) != 2 || roots[0].Key.PID != 10 || roots[1].Key.PID != 21 || roots[0].Slot != "pair:6" || roots[0].Workspace != ws {
		t.Fatalf("roots %+v, want the detached shell and the readable child of the withheld one", roots)
	}
	if len(slots[0].Roots) != 0 {
		t.Fatal("input slots were mutated")
	}
	// Its subtree is then the slot's, not an orphan.
	sel := SelectResourceProcesses(table, got, nil, true, true)
	owned := map[int]string{}
	for _, p := range sel.Processes {
		owned[p.Key.PID] = p.Slot
	}
	if owned[11] != "pair:6" || owned[10] != "pair:6" || owned[22] != "pair:6" {
		t.Fatalf("selection %+v, want the detached shell and its suite attributed to pair:6", owned)
	}
	if _, ok := owned[12]; ok {
		t.Fatal("a leftover zellij server was claimed")
	}
	if _, ok := owned[15]; ok {
		t.Fatal("a detached agent was claimed")
	}
}

// A suite already attributed while it still had ancestry keeps its slot once
// reparented, instead of flipping to an orphan alert.
func TestDetachedSuiteIsNotAnOrphanAfterReparenting(t *testing.T) {
	thread := ThreadAddress{RepoScope: "s", Tag: "t"}
	ws := WorkspaceIdentity{RepoIdentity: "/r/.git", WorktreeRoot: "/r"}
	agent := ResourceRoot{Key: ResourceProcessKey{PID: 5, Identity: "agent-1"}}
	slots := []ResourceSlot{{Slot: "pair:6", Thread: thread, Workspace: ws, OwnershipKnown: true, Roots: []ResourceRoot{agent}}}
	before := []procutil.Process{
		{PID: 5, PPID: 1, Identity: "agent-1", Name: "claude"},
		{PID: 30, PPID: 5, Identity: "sdlc-3", Name: "sdlc"},
		{PID: 31, PPID: 30, Identity: "test-3", Name: "couchcore.test"},
	}
	first := SelectResourceProcesses(before, slots, nil, true, true)
	after := append([]procutil.Process(nil), before...)
	after[1].PPID = 1 // the tool call ended; the suite reparented to launchd
	threadOf := func(p procutil.Process) (ThreadAddress, bool) { return thread, p.PID == 30 || p.PID == 31 }
	for name, s := range map[string][]ResourceSlot{"unattributed": slots, "attributed": DetachedSlotRoots(after, slots, threadOf)} {
		sel := SelectResourceProcesses(after, s, first.Processes, true, true)
		orphan := false
		for _, p := range sel.Processes {
			if p.Key.PID == 31 {
				orphan = p.Orphan
			}
		}
		if want := name == "unattributed"; orphan != want {
			t.Fatalf("%s: test binary orphan=%v, want %v", name, orphan, want)
		}
	}
}
