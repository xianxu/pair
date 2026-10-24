---
id: 000862
status: codecomplete
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'baae92017e13475e309b9c40ca8bb2bc2a83e802' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T15:58:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "8f4253b6", done: "4ffedb0d"}
actual_hours: N/A
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
- Unmappable/missing-baseline apply succeeds with the previous baseline retained and a visible warning; explicit review-then-read-all recovery is documented and tested.
- The dependent Ariadne fake update is filed unclaimed against pair#862, with the approved contract and fixture handoff.

## Plan

Draft: [apply shadow plan](../plans/000862-apply-shadow-plan.md).
One atomic close boundary, no milestones. Await TL approval before code.

- [x] Approve the mapped-range contract and conservative retained-baseline warning; separate the dependent Ariadne fake update.
- [x] Prove pure mapping and bounded growth, then integrate preparation/public warning handling.
- [x] Verify public registered/legacy/CLI paths, unread preservation and refusal/publication controls; update guidance.
- [x] File the dependent Ariadne fake update; complete targeted qualification/inventory and submit recorded pre-close. Final close/landing outcomes remain in Log.

## Log

### 2026-10-24
- 2026-10-24: closed — BR-1 corrected at the receiving authority: ariadne615 details published through SDLC to origin/main91fc60bdc81e1091943c61df2a374669b8d87ac0; active Spec now says inspect diff, review remaining edits, then read --all, and section reads do not advance baseline; receiving Revisions retains the superseded wording. Verified exact main copy. No Pair executable or peer code changed after reviewed95be8dec. Reviewer independently passed session-sync package, focused apply/fuzz seeds and CLI; our targeted race/public/alias/standalone/guard tests and portable14-case/count8-16 tests PASS. Inventory and diff-check PASS immediately before close. Original pre-close lease-2922290125 on6f6e6e216 is retained and must PASS before landing. README/atlas/skill/help and approved plan describe same contract; upstream fake remains unclaimed/dependent until Pair lands. Actual refusal before review: measurement branch proof: measurement sources exceed snapshot limit 128; window2026-10-24T15:58:12-07:00 to refused26fcc83e8 committed16:13:45-07:00, later commits documentation only. Standing hubf0056229/0cd7a589 authorizes precise --no-actual while ariadne603 is working/not-landed; no typed hours.; review verdict: SHIP

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

### 2026-10-24 — implementation and targeted proof

Before the fix, TestApplyPreservesUnreadEditsElsewhere failed for both legacy
and registered documents: the saved shadow included operator unread text
outside the apply target. The pure mapper now reuses changedRuns to map only
actual changed intervals, retaining the whole old shadow on uncertainty.
Preparation carries a nonpersistent warning; publication composes it with
round and late-lock warnings. Existing Write/Read/ApplyFile scope contracts
are unchanged. Core mapping remains free of IO; portable JSON has14 cases.
New pure helper is classified NonArtifactSources in the exhaustive inventory.

Targeted proof, one Go test process at a time under the overload policy:
- New/public baseline tests and fuzz seeds PASS; explicit mapped projection,
  no-op/broad records, same-line/section unread text, duplicate occurrences,
  frontmatter+body, Unicode/CRLF, absent baseline and diff-cap fallback.
- Targeted session-sync race including existing stale/lock/bad-batch and
  source-ownership guards PASS16.342s. Alias/recovery/standalone controls
  PASS3.688s; final new baseline subset race PASS1.466s.
- Public CLI apply --json/diff and frontmatter controls PASS1.218s; warning
  is visible on stderr without a JSON shape change. Artifact inventory
  repaired by classifying the pure helper, then PASS2.941s. Catalog PASS6.904s.
- Counted8/16 mapper span/run visits and actual builder bytes stay linear;
  the existing bounded diff and conservative coarse-hunk fallback are reused.

A legacy alias fixture initially assumed shared canonical metadata; corrected
the fixture to seed its existing independent sidecar. Registered aliases use
canonical metadata. No alias behavior changed. The only new repository read
is testdata/apply-shadow/cases.json, a literal portable pure-test corpus;
audited and recorded its reader as non_go_full_fallback, with no bulk hashes.
Next: commit/push, queued --pre-close, inventory then whole close and merge.
Ariadne#615 is unclaimed and depends on pair#862; its eventual model must use
the final approved review-then-read-all recovery, superseding its early draft's
scoped-read wording. No cross-repo code was changed.

### 2026-10-24 — recorded final-head qualification queued

Pushed implementation 6f6e6e216390727a39408ab588a2e1f078eaeeba; PR599
is open. SDLC --pre-close selected affected, then the declared full fallback
for cmd/internal/sessionsync/testdata/apply-shadow/cases.json. Original runner
terminal39827, attempt a7d4b27fdd25c235d58c12a6a549d612,
lease-2922290125 is queued normally (last observed FIFO4). Preserve it;
no unqueued full suite or native matrix. The final issue Plan row records
submission rather than falsely claiming the pending recorded result/landing.
A close review may overlap this queued check under the worker contract, but
landing requires its pass on the final code. Next: inventory, close review,
retain the check to its terminal result, then authoritative merge on SHIP.

### 2026-10-24 — actuals snapshot cap before review

The first close runner34729 exited1 before review with exact refusal:
“measurement branch proof: measurement sources exceed snapshot limit 128”.
Affected issue window starts 2026-10-24T15:58:12-07:00; refused candidate
26fcc83e8b450df2fb03efa45d78c8078ed4754d committed16:13:45-07:00.
No hours were typed. Authenticated standing hubf0056229, reaffirmed by
hub0cd7a589 (64 KiB exception revoked, 128-source stays until ariadne#603),
authorizes the precise --no-actual retry with this refusal/window retained.
Inventory rerun passed; the original queued suite39827 is unchanged.
GitHub conformance, merge-check and both local-gates checks were observed green.
Next: retry whole close under that measurement-only exception.

### 2026-10-24 — BR-1 receiving handoff corrected on Ariadne main

Close round1 found no runtime defect and returned FIX-THEN-SHIP, but its
Important BR-1 blocked finalization: the receiving ariadne#615 Spec still
prescribed the obsolete scoped-read recovery despite Pair's corrected plan.
Corrected the unclaimed ticket through sdlc issue publish --issue615 from
the clean resting dependency checkout, preserving status/dependency and
adding its own Revisions. Publication exited0; verified origin/main's ticket
at commit91fc60bdc81e1091943c61df2a374669b8d87ac0 states inspect diff,
review remaining edits, then read --all, with section reads non-acknowledging.
No peer code, runtime state or Pair executable changed. The issue stays
unclaimed, depends on pair#862, and waits for the hub's post-land assignment.

The review independently passed the whole session-sync package, focused
apply/fuzz seeds, public CLI and diff check. Runtime proof is unchanged;
original recorded suite39827/lease-2922290125 remains queued. Resubmit close
to dispose BR-1, retaining the precise authorized actuals exception.

### 2026-10-24 — SHIP; retain original queued verification before boarding

Whole close round2 finalized SHIP with BR-1 addressed and no remaining
findings. Reviewer verified the receiving Ariadne ticket's before/after at
91fc60bd and independently passed the full session-sync package25.573s and
public CLI0.343s. Codecomplete is bound to evidencee865e2db32b7e70fd21b07f883407bd0138b49ca;
the mirror is b5932f36b339a4681f385960a2e494039dc2856e.

The first merge runner42330 exited1 before boarding: GitHub conformance and
merge-check were pending on the close's pushed head, and PR599 still needs
its recorded local check. Its named recovery explicitly says retain a queued
or running affected/full-fallback check and retry merge after PASS. Retain
original terminal39827, attempta7d4b27fdd25c235d58c12a6a549d612,
lease-2922290125 on implementation6f6e6e216390727a39408ab588a2e1f078eaeeba
(last observed normal FIFO3). No live merge runner or duplicate verification.
All descendants of that implementation are prose/review/close evidence only.
Next: read the original suite to its terminal result. On PASS verify CI and
re-enter one sdlc merge --yes; on failure classify before any repeat. Preserve
queue place and do not checkpoint/restart a continuation while it is active.
No operator stage, native matrix or out-of-queue full run was started.

## Revisions

### 2026-10-24 — separate dependent fake adoption, TL-approved

Original Done when required “ariadne's tl-lead fake is updated to match”.
Per authenticated hub5ef6e854, Pair's acceptance now covers Pair's range
mapping, conservative warning and filed contract handoff only. The fake update
waits on landed pair#862 in its own unclaimed Ariadne issue; #862 does not wait
on its dependent. This removes a dependency cycle without dropping the fake
adoption work. The approved uncertain outcome is now explicit in Done when.

### 2026-10-24 — recovery corrected after executable counterexample

Authenticated hub4f69397b approves recovery: inspect diff, review remaining
edits, then explicitly read --all to acknowledge them. The earlier proposed
scoped-read recovery was false: Read(section) never saves the shadow today
(sync.go:295-320); four recovery tests exposed it. Do not change Read(section),
introduce implicit acknowledgment or add state. This entry supersedes earlier
scoped-read recovery wording; frontmatter/scoped writes retain their contracts.
