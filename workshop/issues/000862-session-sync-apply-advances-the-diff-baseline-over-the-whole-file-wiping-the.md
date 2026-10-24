---
id: 000862
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'ad29701005c921a39535c1fa45ea756931a7c5e6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T15:58:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# session-sync apply advances the diff baseline over the whole file, wiping the operator's unread edits elsewhere

## Problem

`couch --session-sync apply` is a whole-file write, and it advances the agent's diff baseline to the whole new
document, so it consumes the operator's unread edits everywhere in the file, not only where its records landed.

Evidence (cmd/internal/sessionsync/sync.go):

- `Apply` runs `s.write` with no `Section` and no `Frontmatter` (the records build the body).
- `prepareWrite` then takes `shadow := f.shadowAfter(req.Section, body, next, now)`.
- `shadowAfter` returns `next`, the whole new file, when `section == ""`.

So after an apply, `diff` no longer shows the operator's edits in sections the records never touched. A section
write (`shadowAfter` with a section) and a frontmatter write (`frontmatterShadow`) both advance only their own
scope, so the operator's edits elsewhere stay in `diff` (pair#448, ariadne#426). Apply is the only write that
doesn't keep them.

Found by ariadne#588: `tl-lead instructions link` was going to put the `discussions:` index entry and the answer link
in one apply to make the operation all or none. That would have erased the operator's unread `## Needs you`
edits from the TL's diff, so ariadne#588 kept two scoped writes and an undo instead. ariadne's tl-lead
session-sync fake now models apply advancing the whole-file baseline (TestFakeApplyAdvancesBaseline).

## Spec

A whole-file apply advances the baseline only for the regions its records edited: the shadow takes each edited
range's new bytes, and the operator's unread edits outside them stay in `diff`. The shadow and the file can then
differ at the same offsets they differed before, outside the edited ranges; mapping the records onto the shadow
needs the same resolution against the shadow's text (or a line diff of file before and after, replayed on the
shadow), refusing to advance (keeping the old shadow plus the edited ranges marked) when a record's range doesn't
map cleanly.

## Done when

- An apply whose records touch only one section leaves an operator's unread edit in another section in `diff`
  (test).
- An apply's own edits don't show in `diff` afterwards (test).
- Unmappable/missing-baseline apply succeeds with the previous baseline retained and a visible warning; scoped-read recovery is documented and tested.
- The dependent Ariadne fake update is filed unclaimed against pair#862, with the approved contract and fixture handoff.

## Plan

Draft: [apply shadow plan](../plans/000862-apply-shadow-plan.md).
One atomic close boundary, no milestones. Await TL approval before code.

- [x] Approve the mapped-range contract and conservative retained-baseline warning; separate the dependent Ariadne fake update.
- [ ] Prove pure mapping and bounded growth, then integrate preparation/public warning handling.
- [ ] Verify public registered/legacy/CLI paths, unread preservation and refusal/publication controls; update guidance.
- [ ] File the dependent Ariadne fake update; pass inventory and recorded pre-close, then whole close and land.

## Log

### 2026-10-24

### 2026-10-24 — claim, investigation and plan for TL

Claimed in pair:3 and started from fresh main
f01ccc06b55fd3c5ac383fe172f4a4bb3ab74bcf. Confirmed the issue's Apply ->
prepareWrite -> shadowAfter(empty section) path replaces the entire baseline.
Ariadne's TestFakeApplyAdvancesBaseline explicitly pins that defect; changing
instructions link is not this issue's scope. The fake update needs coordination
with its owning repo/slot, not unlanded runtime imports.

Recommended design reuses bounded changedRuns to map actual changed byte runs
through unchanged old-shadow/current-file intervals. Independent occurrence
resolution against the shadow is rejected because unread edits can change
occurrence counts. Broad records and no-ops must not acknowledge unchanged
text. Unmappable/missing-baseline outcome is proposed as existing-document
apply success, old baseline retained, visible warning and scoped-read recovery;
this policy goes to TL for explicit approval before code. No new persisted
coverage state, parser, timeout or retry. Durable plan records pure/integration
boundaries, failure states, counted growth, fixtures and public verification.

### 2026-10-24 — approved scope and uncertainty policy

Authenticated ops:0 approval 5ef6e854-e86a-48d7-b966-adec8c332ce2 accepts
the plan, including successful document apply with whole old baseline retained
and a visible scoped-read recovery warning when any mapping is uncertain or
the baseline is absent. The hub explicitly directs the fake update to a
separate unclaimed Ariadne issue depending on pair#862 and will assign it
after Pair lands. No code has been changed before approval.

Filed ariadne#615 unclaimed with dependency pair#862; creation published card
and complete details on origin without changing the peer checkout.

## Revisions

### 2026-10-24 — separate dependent fake adoption, TL-approved

Original Done when required “ariadne's tl-lead fake is updated to match”.
Per authenticated hub5ef6e854, Pair's acceptance now covers Pair's range
mapping, conservative warning and filed contract handoff only. The fake update
waits on landed pair#862 in its own unclaimed Ariadne issue; #862 does not wait
on its dependent. This removes a dependency cycle without dropping the fake
adoption work. The approved uncertain outcome is now explicit in Done when.
