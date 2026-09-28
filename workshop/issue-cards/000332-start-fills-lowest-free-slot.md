---
id: '000332'
status: done
started: 2026-09-25T10:42:40-07:00
created: 2026-09-25
updated: 2026-09-25
actual_hours: 2.20
---

# Starting a thread fills the lowest free slot number, :0 included

## Problem

Starting a thread on a repo leaves "holes" in the slot numbers. Archiving a
thread keeps its slot's worktree on disk, and `SelectNewSlot` treats every
existing slot directory as taken. So an archived `:1` is never reused and the
next start allocates `:N+1` (ariadne went to `ariadne-slot3` while `:0`/`:1`
had no thread). #331 fixed only the `:0` case with a `:0`-specific check.

Separately, #306's admission rule refuses to create any new slot while a
parked thread exists anywhere in the repo.
