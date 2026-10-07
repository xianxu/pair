---
id: 000403
status: open
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '84d797c36992a5ece3ac92558fbaf8556a0da07e' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: resolving repo:N takes ~2 s and runs twice per command

## Problem

Every couch command that takes a `repo:N` reference pays for resolving it. Measured
on 2026-10-06 (pair#362 plan, revision a):

| Command | Wall time |
|---------|-----------|
| `couch --list` (no slot reference) | 0.23 s |
| `couch --peek ariadne:1`, of which rendering is 85 ms | 2.3 s |
| `couch --show ariadne:1` | 3.7 s |

The time goes to `resolveSlotInput` (`couchcore/slotcontext.go`): workspace
resolution of the caller's directory, the repository's primary, and
`Slots.Discover` (git worktree and ref listing). The CLI path then runs it twice:
- `runTypedOperationWithConsole` calls `WorkspaceReferencePath` to derive the
  repository scope;
- the dispatcher's `resolveOperationThread` → `ResolveThreadReference` calls it
  again.
`--show`, `--resume`, `--reboot`, `--peek` and `--reconcile` all pay it.
Observation tools are meant to be fast (cross-slot-work-scheduling goals); a
coordinator that checks several slots pays it once per slot per check.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation.

- **Measure first.** Break the 2 s down (workspace resolution, the primary lookup,
  `Discover`) on the live fleet before choosing a fix. Record the profile in the Log.
- **Resolve once per command.** Carry the resolved slot (path and scope) from the
  CLI's scope derivation into the dispatch, instead of resolving the reference
  again.
- **Then cut the per-resolution cost** where the profile points. For example, skip
  workspace resolution of the caller's directory when the reference names its
  repository explicitly, or have `Discover` read only what a single slot needs.
  Correctness stays as today: the reference resolves to the same slot, and errors
  are reported the same way.

## Done when

- The profile is recorded.
- A `repo:N` command resolves its reference once, with a test that counts
  resolutions.
- `couch --peek repo:N` and `couch --show repo:N` are measurably faster on the live
  fleet, before and after recorded in the Log.

## Plan

- [ ]

## Log

### 2026-10-06

Filed at the operator's request from pair#362's peek measurements. Details left
local for the operator to refine.
