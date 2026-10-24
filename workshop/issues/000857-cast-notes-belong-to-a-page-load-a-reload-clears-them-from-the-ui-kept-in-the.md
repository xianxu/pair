---
id: 000857
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
card_mirror: 'b7e679930fb167469cde5b0368b2e033661b77b4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T21:35:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Cast notes belong to a page load; a reload clears them from the UI (kept in the recording)

## Problem

## Spec

**Cast notes belong to a page load; a reload clears them from the UI (operator, 10-23).** Today the live Notes panel
shows every unsent note in the whole cast, regardless of page, and a reload may not even be detected (pair:4's
read-only findings: one document recorded through a refresh; the panel projects all open notes; detection waits for a
later DOM update; load ids aren't saved). Operator decision: "after a reload, user would expect all old annotations
gone"; "if the same page load id (e.g. switch tab and back), keep the note; page refresh, notes are gone, at least
from the UI; agents can still check the disk; not in the main user UI."

Change:
- Give every note and sketch a **page-load identity**: the tab (CDP target id, already recorded) plus the page load
  (loader id plus root node), saved in the recording.
- Detect a real top-document replacement directly from Chrome's navigation event (fenced resnapshot), not only via a
  later DOM update.
- The live panel and the submit selection show **only the current page load's** notes and sketches. Notes from earlier
  loads are not shown anywhere in the UI (no "earlier pages" section); they stay in the recording and remain readable
  through `couch --broadcast-web-query`.
- Same load id (pause/resume, switching tabs and back) keeps the notes. An app refreshing its own content within one
  document is the same page.

## Done when

- After a real same-URL reload, the panel is empty of earlier notes, and they're still queryable from the recording
  (native test).
- Pause/resume and A→B→A tab switching keep notes; an app-only refresh keeps them; a second tab on the same URL has its
  own notes (native tests).

## Plan

- [ ] Implement persisted load identity and replay/catch-up preservation per [the durable plan](../plans/000857-page-load-feedback-plan.md).
- [ ] Fence recorder navigation at ingress and filter Board's atomic press selection from confirmed scope.
- [ ] After #856 lands, integrate the current-load UI projection and unfiltered historical provenance.
- [ ] Qualify fake/native reload, pause/resume, tab return, app update and same-URL new-tab behavior; close, stage through the hub and merge.

## Log

### 2026-10-23

### 2026-10-24 — Independent data and recorder work

TL receipt 8b5caf7d-587c-44ab-9f60-a0832c521a5a authorizes a separate branch from current main now, copying no #856 commits. Started from main 9c81052a1; #849 press-time selection is present, #856 UI integration waits for its landing. The approved joint design is carried into this issue's own durable plan before change-code. Read-only surveys show synchronous recorder snapshot IO requires the approved generation fence at ingress, not only when the actor later dequeues navigation. They also establish no individual CDP identifier cap; retain that contract and positive int64 backend roots, with exact string projection to JS.

#856 remains SHIP on its retained branch with PR583, its exact-source recorded check in terminal5418 / lease-306801268 / run2d195b5c900c7a2bf81b53c80416abe4, source087631062be649f7578cab38fe75ddc6c3044ed1, detached snapshot /Volumes/TestScratch/local-checks/sdlc-local-1248088162. It is still queued, now position4 behind the active ariadne:2 suite. Do not duplicate/cancel it. When it passes, checkpoint and push this branch and return immediately to #856's merge; then resume #857. No production edits before this issue's change-code gate.

## Revisions

### 2026-10-24 — Parallel independent preparation

The approved joint scope is unchanged. TL8b5caf7d permits data/recorder work before #856 lands, on a main-based branch with no copied #856 commits; its UI integration still waits for landed main. Pending/unknown scope contributes no page feedback, without adding a global text-submit refusal; the ordinary unsaved-note guard stays. Earlier-load feedback remains queryable history, entirely absent from live UI.
