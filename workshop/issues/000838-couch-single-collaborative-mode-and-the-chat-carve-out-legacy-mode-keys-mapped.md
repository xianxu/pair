---
id: 000838
status: working
deps: [ariadne#575]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bee8b954bca9a0701859c5bd0550e2180427ff21' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T14:02:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch: single collaborative mode and the @@chat@@ carve-out; legacy mode keys mapped or ignored

## Problem

## Spec

Implements ariadne's single-collaborative-mode spec (filed alongside, from the operator's 10-23 pivot) in Couch,
simplifying pair#823's per-section mode machinery:
- Rounds follow the one collaborative mode; presentation is always highlight for agent edits.
- `@@chat@@` above a heading marks a chat section (numbered inputs always get side-doc replies); Couch recognises it,
  protects the tag as operator-owned, and enforces the chat rule there.
- Legacy keys: `collaboration-mode: chat` (document-wide or per-section map) maps to the chat carve-out; the
  presentation and processing keys and instructions are ignored with one deprecation notice per round; nothing new
  writes them. pair#823's migration command converts old maps to `@@chat@@` tags (operator-run).
- Remove the per-section mode compiler paths that only served the retired keys, keeping the legacy reads above.

## Done when

- A round on a document with a `@@chat@@` section follows the chat rule there and the collaborative rule elsewhere
  (test against the spec's fixtures).
- Documents with legacy keys resolve as mapped, with one notice and no refusal (test; plus a read-only audit of the
  real ops and brain session docs like pair#823's).
- Brain's chat document keeps working unchanged.

## Plan

Draft: [single collaborative mode and chat plan](../plans/000838-collaborative-chat-plan.md).
One atomic full-flow implementation, one issue-close review (no Mx tags).

- [ ] Approve design and verify ariadne#575 landing; qualify final scanner/notices against ariadne#576.
- [ ] Pin shared fixtures; simplify the editorial compiler/guards around sidedoc.ChatSections.
- [ ] Wire new highlight rounds, legacy notices and lossless operator map-to-tag migration.
- [ ] Prove public seams, old rounds, real-doc read-only audit and disposable native acceptance; update atlas/help.
- [ ] Inventory, recorded affected check, close review, operator acceptance as agreed with TL, then land.

## Log

### 2026-10-23

### 2026-10-23 — claimed and draft plan

#811 completed before claim: its merge runner exited0, tracker/archive and
main reachability verified, final proof Log published. Claimed #838 in pair:3,
then start-plan created its fresh-main branch and refreshed the ordinary
Ariadne dependency. No code, live store or :0 changes.

Read the operator's ops side doc and ariadne#575's issue and committed candidate
6ae80e4e69aacb72d8745e479791ada7c4454c17. Its shared sidedoc.ChatSections
already owns tag/legacy grammar and notices (ARCH-DRY). Current Pair source
still has independent three-axis resolution, runtime briefs and mode filling;
the durable plan removes those paths while preserving transaction/marker/chat
ownership and old saved-round decoding.

Cost inspection found shared #575 prefix counting and heading-end scans can
cost O(B*H+H²). Authenticated ariadne:2 receipt
c8bda0b0-3e7f-43f4-9301-6a5521d89dc6 confirms it, with no current bound/test;
the owner takes the linear scanner/count tests and two notice corrections in
ariadne#576 and is informing the TL. Plan proposes #576 before final fixture
pin/performance qualification/close, with no Pair scanner fork (ARCH-CONSTRAINTS).

Draft also names a migration compatibility trap: new tags accept numbered
items, old chat accepts bullets and whole-document/preamble asks. Proposed
operator transform refuses conversions that alter the eligible-item projection,
leaving legacy reads fully functional and source untouched. This is explicit
in the approval request, not a silently chosen product refusal.

Next: commit/push this design and send it to ops:0; wait for plan approval and
#575 landing before change-code, and #576 for final qualification. No estimate
yet and no implementation started.

## Revisions

### 2026-10-23 — initial implementation design

Expanded the empty Plan after the TL's dispatch, preserving the operator's
Spec and Done when. Recorded the approved ariadne#575 dependency, shared
scanner ownership and one atomic review boundary. Proposed (not yet approved)
lossless-only migration refusals and ariadne#576 final qualification, with the
reasons and alternatives in the durable plan.
