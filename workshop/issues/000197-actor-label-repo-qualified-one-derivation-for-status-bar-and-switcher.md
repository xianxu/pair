---
id: 000197
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# actor label: repo-qualified, one derivation for status bar and switcher

## Problem

**The request.** The status bar names an actor by its repo, so every thread
started in a subdirectory of the same repo reads identically. It should carry
the subdirectory when there is one: `repo` at a repo root, `repo:<last segment>`
below it. For `~/workspace/kbench/competition/arc-agi-3` → `kbench:arc-agi-3`.

**What checking it found: the status bar and the switcher already disagree.**
Two independent derivations exist for one concept, and neither knows about the
other:

| surface | source | value for the example |
|---|---|---|
| status bar chip | `couchtty/console.go:1817` passes `start.Record.Args.Worktree.Repo()`; `Repo()` is `filepath.Base(git toplevel)` (`couchcore/worktree.go:22`) | `kbench` |
| switcher row | `threadLabel(name, WorkingPath, tag)` → `filepath.Base(WorkingPath)` (`couchcore/actionableinventory.go:148`) | `arc-agi-3` |

So the same actor is `kbench` on the chip and `arc-agi-3` in the switcher, and
the operator cannot match one to the other. That is a defect independent of the
requested format, and it is the reason to fix this at the derivation rather than
at the status bar (`ARCH-DRY`). `threadLabel`'s own doc comment already uses
`arc-agi-3` as its worked example — the two surfaces were reasoned about
separately and landed on different answers.

Only the switcher's rule is documented (`atlas/couch.md:743`). The status bar's
`Worktree.Repo()` is undocumented and appears to be incidental — it is what
happened to be in hand at the attach call site, not a decision with a recorded
reason.

**The inputs already exist; no plumbing is needed.** `StartArgs`
(`couchcore/startargs.go`) carries both halves, and its `WorkingDir()` already
encodes exactly the distinction the request turns on:

```go
Worktree  Worktree // git toplevel, resolved via rev-parse --show-toplevel
Cwd       string   // set only when the operator named a subdirectory

func (a StartArgs) WorkingDir() string { if a.Cwd != "" { return a.Cwd }; return string(a.Worktree) }
```

`Cwd == ""` *is* "started at the repo root". The attach site already holds the
whole `start.Record.Args`, so it passes a narrower value than it has.

## Spec

**One exported derivation, both surfaces call it.** The deliverable is the
shared function plus both call sites converted — a fix that changes only the
status bar leaves the divergence in place and is the easy subset of this issue
(`ARCH-PURPOSE`).

The rule:

- started at the repo root → `repo`
- started in a subdirectory → `repo:<last segment of the working dir>`, one
  segment regardless of nesting depth (`~/w/kbench/a/b/c` → `kbench:c`)
- a human-assigned thread name still wins outright, as `threadLabel` already
  does — this changes only the derived case
- no repo (path outside a git tree) → last segment, then the tag, preserving
  `threadLabel`'s existing fallback chain

Pure function over data already in `StartArgs` / the thread summary: unit-tested
directly, no IO (`ARCH-PURE`).

### To settle in the plan

1. **Where it lives, and whether the switcher can reach it.** `threadLabel`
   receives `(name, WorkingPath, tag)` and has no repo in hand.
   `ActionableThreadSummary` carries `Address.RepoScope`, but that is Pair's
   hidden scope key (`atlas/couch.md:957`) — confirm whether it yields a
   *display* repo name or whether the summary needs the worktree alongside the
   working path. This is the one real unknown; the rest is mechanical.
2. **Interaction with `DisambiguateLabels`.** Repo-qualification is itself a
   disambiguator and should *shrink* the `·<tagtail>` fallback: two threads at
   `competition/arc-agi-3` under different repos collide today and would not
   after. Keep the suffix for genuine same-path collisions (the operator's six
   `brain` rows), and confirm the collision count is computed after
   qualification, not before.
3. **Width.** #172 established that chip spans are display columns, and the
   status row's budget is scarce. `kbench:arc-agi-3` is 16 columns; a git
   worktree directory is worse — `~/workspace/worktree/pair/000140-muse-return-rewrite-only-in-composer`
   yields `pair:000140-muse-return-rewrite-only-in-composer`. Decide the
   truncation rule, and which half survives it (the segment identifies; the repo
   groups). A rule that silently overflows the row is not acceptable.
4. **Separator.** `:` reads as repo-qualification and matches `repo#id`
   convention. Confirm nothing downstream parses labels on `:` — notices
   (`couchtty/notice.go:34`) interpolate the label into text.

Out of scope: the actor *description* (#173). That answers "what is this thread
doing" from the per-turn `pair-slug`; this answers "which thread is this" from
stable launch identity. They compose on the same row and neither substitutes for
the other.

## Done when

- An actor started at a repo root shows `repo`; one started in a subdirectory
  shows `repo:<segment>` — `~/workspace/kbench/competition/arc-agi-3` reads
  `kbench:arc-agi-3`.
- The status bar chip and the switcher row show the **same** string for the same
  actor, and a test pins that equality rather than asserting each surface's
  format separately — the divergence is what let them drift.
- One derivation function; `grep` finds no second place computing a display
  label from a path.
- Unit tests over the rule's cases: repo root, one level down, nested deeper,
  human-named thread, no-repo path, no path at all.
- Collision behavior verified after qualification: cross-repo same-segment
  threads no longer take a `·<tagtail>` suffix; same-path threads still do.
- The chosen truncation rule holds the status row within its column budget for
  a worktree-length label.
- `atlas/couch.md` states the single rule, replacing the switcher-only account
  at `:743`.

## Plan

- [ ] Settle design question 1: whether `RepoScope` yields a display repo name,
      or the summary must carry the worktree.
- [ ] Write the pure derivation + its unit tests (rule cases per Done-when).
- [ ] Convert both call sites: `console.go:1817` and `threadLabel`.
- [ ] Pin chip/row equality in a test; verify `DisambiguateLabels` counts after
      qualification.
- [ ] Apply the truncation rule; check a worktree-length label on the real row.
- [ ] Update `atlas/couch.md:743` to the single rule.

## Log

### 2026-09-06

Filed from an operator feature request. Checked first for an existing record, as
asked: none. #173 (`open`) is the nearest and is a different axis — the actor
*description* from `pair-slug`, i.e. what the thread is doing, versus this,
which is stable launch identity. `atlas/couch.md:743` documents the switcher's
rule only.

The status-bar/switcher divergence was not in the request and is the more
significant half of what was found: the request is a format change, but the two
surfaces disagreeing means a chip cannot be matched to a switcher row today.
Fixing only the status bar's format would leave that in place, so the Spec makes
the shared derivation the deliverable and both call sites the acceptance
criterion.

Nothing in the change needs new data — `StartArgs.Cwd` already distinguishes
"operator named a subdirectory" from "started at the tree root", which is
exactly the condition the requested format turns on.
