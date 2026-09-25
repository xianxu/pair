---
id: 000332
status: working
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
started: 2026-09-25T10:42:40-07:00
flow: {kind: quick, provenance: inferred, spec: "400bb6e0", done: "a9524a2b"}
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

## Spec

Operator decisions (2026-09-25):

- **One rule for every number, `:0` included — no `:0` special case.** A slot
  number is *free* when it has no live or parked thread: never created, or
  its thread archived. Replaces #331's `:0`-only occupancy check.
- **Start fills the lowest free number.** An existing directory with no thread
  (e.g. archived `:1`) is reused as-is: the new thread inherits whatever
  branch and uncommitted files the old one left. Only when every number
  `0..N` is taken is a new slot `:N+1` created.
- **Parked threads occupy their number but no longer block creation.** When
  there is no hole and some threads are parked, create the next slot, and the
  start preview names the parked threads as a reminder. This reverses #306's
  "parked blocks new slot" rule to a notice.
- Unchanged: an unreadable record anywhere in the repo still refuses (can't
  tell whether its number is taken); the preview/fingerprint contract still
  refuses a changed selection instead of renumbering.
- Consequence: the thread menu's **Add slot** uses the same path, so it fills
  a hole before adding a number.
- The repository action is rendered as `add slot`.
- Add-slot previews render actionable reuse notices without changing the
  selected allocation: `consider reuse parked <repo>:<n> with open-slot` for
  parked threads, and `consider reuse lost <repo>:<n> with fresh-slot` for
  lost-binding slot rows. When both kinds exist, show both in slot-number
  order.

## Done when

- Tests: holes at `:0`, at `:1` with `:0`/`:2` live, and at `:1` with `:2`
  parked each resolve to the hole; the reused `:1` keeps its leftover files;
  no hole + a parked thread creates `:N+1` and the preview lists the parked
  thread; unreadable record still refuses.
- #306's "parked blocks create" tests updated to the notice behavior.
- Atlas (`workspace-provisioning.md`, `couch.md` slot section) states the rule.
- Operator smoke test in couch: archive a slot thread, start on the repo, land
  on the freed number.

## Plan

Design (from code reading + a probe test, 2026-09-25):

- Occupancy source = the thread snapshot's records mapped to repository slot
  scopes. A number is *occupied* iff a record carries that number. The probe
  showed an archived `:1` stays as a slot row with empty address
  (`unusable`/`never-started`) — that is a hole. Unreadable thread records and
  unreadable slot-current records still refuse (can't know their number's
  state).
- One pure function `lowestFreeSlot(numbers occupied, existing dirs) → (n,
  exists)` (ARCH-PURE), unit-tested; `resolveManagedStart`'s StartCreate
  branch maps its answer to an action, replacing #331's `:0`-only check and
  the "any slot exists ⇒ occupied" rule:
  - `0` → ordinary primary start (as today when free);
  - `n` with an existing directory → `StartCreate` + `ReuseSlot` on that slot
    (the reuse path requires the current slot record to remain empty);
  - `n` without a directory → `StartCreate` of slot `n` (provision as today).
- `checkSlotCreation` drops the parked blocker (keeps the unreadable refusal);
  `StartResolution` gains an ordered typed `ReuseNotices` list (labels and
  kinds, omitted from the fingerprint so a park elsewhere does not invalidate
  a preview), rendered as actionable `open-slot`/`fresh-slot` suggestions.
- ARCH-FUNERAL: creates nothing durable beyond what slot create already made;
  reuse *reduces* slot-directory growth. No new files.

Steps:
- [x] Unit tests for the pure selector; integration tests for holes at :0,
  :1 (dir exists, thread archived — reused with leftover file), :1 with :2
  parked, no-hole+parked → new slot with notice, unreadable refuses
- [x] Implement selector + resolveManagedStart mapping; drop parked blocker
- [x] Preview notice in start form (+ render test)
- [x] Update #306 parked-blocks tests to the notice behavior; atlas
- [x] Full `couchcore` and `couchtty` package tests; operator smoke test in
  couch.
- [x] Add-slot wording and parked/lost reuse notices have focused render/menu
  coverage.
- [x] Unreadable slot-current state refuses allocation, hole reuse refuses an
  occupied current record, and mixed reuse notices retain numeric order.
- [x] README documents lowest-free reuse and the add-slot reuse guidance.
- [x] Reused starts preserve the accepted launch profile through workspace
  readiness and refuse drift before launching.
- [x] Reused-slot launch payloads are sourced from that accepted resolution;
  later profile reads can only refuse drift, never substitute a profile.

## Log

### 2026-09-25

- Implemented (fadd5927). Probe test confirmed an archived `:1` stays a slot
  row with empty address (`unusable`/`never-started`) and old code picked `:3`.
  Occupancy uses snapshot records mapped scope→number (cheap, no session
  probes); parked labels use the switcher inventory only when adding a slot.
- Caught before review: resolving a hole-fill to action `fresh` broke the
  menu's CommitArgs round trip ("fresh action requires an existing numbered
  slot" on the primary path). Kept action `create` + `ReuseSlot` in the
  fingerprint; the reuse test now commits via CommitArgs/SpawnPrepared, and
  removing the reuse routing fails it (mutation-checked).
- `TestManagedCreateParkAppearingDuringSetup…` and
  `TestManagedLaunchThenPark…` flipped from "parked blocks" to "launch
  proceeds / preview names parked". Full `make test` green.
- 2026-09-25 — Add-slot presentation now uses lowercase `add slot`, renders
  parked reuse suggestions as `consider reuse parked <repo>:<n> with
  open-slot`, and renders lost-binding slot suggestions as `consider reuse
  lost <repo>:<n> with fresh-slot`. Focused tests plus full `couchcore` and
  `couchtty` package tests pass.
- 2026-09-25 — Operator smoke test passed after moving the #332 branch into
  the primary workspace and restoring `ariadne:0`.
- 2026-09-25 — Boundary-review fixes: unreadable slot-current records now
  block allocation, hole reuse requires an empty current record at commit,
  and typed reuse notices preserve numeric order. Added regressions and
  updated the README. Full `couchcore` and `couchtty` package tests pass.
- 2026-09-25 — Second boundary-review fixes: reuse now revalidates the
  accepted launch profile after workspace preparation, checks physical
  current-file absence rather than decoded-record absence, and covers the
  `:1` hole plus parked `:2` integration case.
- 2026-09-25 — Full package verification caught and fixed reuse fingerprint
  reconstruction dropping `ReuseSlot`; the archived-slot commit path and the
  complete `couchcore`/`couchtty` suites now pass.
- 2026-09-25 — Final boundary-review fix: reused-slot launches now clone the
  accepted launch profile instead of resolving a replacement profile after
  preparation. Focused couchcore/couchtty regressions pass.

## Revisions

- 2026-09-25 — Boundary review found that slot-current read errors were not
  included in admission, hole reuse lacked an empty-current commit guard, and
  separately stored notices could lose mixed numeric ordering. The design now
  blocks uncertain slot inventory, uses an ordered typed notice list, and
  requires `StartCreate` hole reuse to observe an empty current record.
- 2026-09-25 — Revisions from the next review: hole reuse also carries the
  accepted launch resolution through readiness and revalidates it before
  launch; the empty condition is physical current-file absence, not merely a
  nil decoded record.
- 2026-09-25 — Revisions from the final review: the reused launch now derives
  its payload and profile provenance from the accepted resolution; later reads
  are validation-only and cannot substitute launch authority.
