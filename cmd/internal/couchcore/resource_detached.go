package couchcore

import (
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// DetachedSlotRoots adds a root for each launchd child whose inherited
// environment names a live slot's thread (pair#606). An agent's backgrounded
// command (Claude Code's background shells reparent to launchd, and a timed-out
// tool call leaves its command behind) keeps the slot's COUCH_THREAD_SCOPE and
// COUCH_THREAD_TAG but loses its ancestry, so its subtree (a recorded suite and
// its test binaries) would otherwise read as orphans. macOS withholds a platform
// binary's environment, so when the launchd child itself (typically /bin/zsh)
// names no thread, its direct children are asked instead; the root is then the
// child that answered. Zellij servers, agents and embedded nvim are never
// claimed this way: a relaunched thread keeps its address, so its leftover old
// server or agent carries the same environment and must stay visible to the
// unattributed-orphan rule (#583). Attribution only; never permission for
// effects. threadOf answers from the process's environment.
// pair:m5-concept pure
func DetachedSlotRoots(table []procutil.Process, slots []ResourceSlot, threadOf func(procutil.Process) (ThreadAddress, bool)) []ResourceSlot {
	out := make([]ResourceSlot, len(slots))
	index := make(map[ThreadAddress]int, len(slots))
	for i, s := range slots {
		out[i] = s
		out[i].Roots = append([]ResourceRoot(nil), s.Roots...)
		index[s.Thread] = i
	}
	if len(slots) == 0 {
		return out
	}
	infrastructure := map[string]bool{"zellij": true, "nvim": true}
	for _, n := range launcher.AgentInventory() {
		infrastructure[n] = true
	}
	candidate := func(p procutil.Process) bool {
		return !p.Zombie && p.PID > 0 && p.Identity != "" && !infrastructure[filepath.Base(p.Name)]
	}
	children := map[int][]procutil.Process{}
	for _, p := range table {
		children[p.PPID] = append(children[p.PPID], p)
	}
	claim := func(p procutil.Process) bool {
		thread, ok := threadOf(p)
		if !ok {
			return false
		}
		i, live := index[thread]
		if !live {
			return false
		}
		s := &out[i]
		s.Roots = append(s.Roots, ResourceRoot{Key: resourceKey(p), Slot: s.Slot, Thread: s.Thread, Workspace: s.Workspace})
		return true
	}
	for _, p := range children[1] {
		if !candidate(p) || claim(p) {
			continue
		}
		for _, c := range children[p.PID] {
			if candidate(c) {
				claim(c)
			}
		}
	}
	return out
}
