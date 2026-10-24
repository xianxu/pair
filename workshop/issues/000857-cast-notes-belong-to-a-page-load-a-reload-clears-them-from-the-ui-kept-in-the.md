---
id: 000857
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '742e73954c53259a1f749c33f0dee6192b1b4c19' # card fields mirrored from issue-cards; edit via sdlc
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

Dependent implementation section in [Cast feedback workspace](../plans/000856-cast-feedback-workspace-plan.md); its own branch and change-code gate follow #856 landing.

- [ ] Approve page-load identity, pending-scope and compatibility policies in the joint design.
- [ ] Persist/derive load membership and fence top-navigation snapshots.
- [ ] Filter UI and atomic press selection, qualify native lifecycle and preserve history query.
- [ ] Close, obtain operator acceptance and land independently.

## Log

### 2026-10-23

### 2026-10-23 — Joint plan with #856

Claimed alongside #856 at the hub's request. Recording already computes loader/root identity but does not persist it; top frame navigation currently updates loader without directly forcing resnapshot. DocSeg is an activation fence, so tab return cannot use it as the page-load identity. Plan derives note/sketch membership from additive segment metadata, preserving sketch request digests and unfiltered historical query. Pending or legacy-unknown scope excludes page feedback rather than borrowing a prior load. This policy and codec extension handling are explicitly proposed for approval, not implemented. #849 owns board/submission.go until landing; coordinate before edits. Next: joint design approval, then own main-based branch after #856 lands.

## Revisions

### 2026-10-23 — Dependent branch and submit behavior

TL approved one joint plan with #856 and a separate #857 branch from landed main. No earlier-load UI drawer. #849's new atomic press selection is the membership seam; preview does not gate submission and an already admitted selection stays pinned through later navigation. Unknown current load contributes no page feedback while ordinary text and the unsaved-note guard retain #849 behavior; this conservative edge policy awaits joint plan approval.
