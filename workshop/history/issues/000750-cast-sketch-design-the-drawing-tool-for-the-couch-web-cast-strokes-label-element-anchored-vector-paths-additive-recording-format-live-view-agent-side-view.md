---
id: 000750
status: done
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1e3919d5d263433a51d32251c6ccf91dce708c53' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T19:40:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 0.42
---

# cast-sketch: design the ✎ drawing tool for the Couch web cast (strokes + label, element-anchored vector paths, additive recording format, live view, agent-side view)

## Problem

Project cast-sketch (ops `teams/devinfra/projects/cast-sketch/`, created 10-19, started 10-20 by the operator: "I want
to use it as part of the design process for the echo project"). The Couch web cast has a pointer and an annotation
tool (bounding boxes with notes); viewers can't show what a page should look like. A third tool, ✎ drawing, adds
sketches.

## Spec

The design below was approved by the operator via ops:0 on 2026-10-20, with the drawing-privilege and pressure revision recorded below.

Approved design: [cast-sketch plan](../plans/000750-cast-sketch-plan.md). Drawing follows driving: console user inside the casting Chrome while control is local, or the current granted drive-link holder, exactly one drawer cast-wide. Pointer/view links watch; pointer links still point. Include pen pressure and Apple Pencil, streaming strokes, current-sketch Undo/Redo, label on leaving, held source moment, fixed colour, one anchor per stroke and no old-sketch takeover after expired link. Agent handoff is operator-only recording plus local CLI export; no viewer-side submission.

- The ✎ tool and stroke capture: a sketch is strokes plus a **required label** (the label carries intent; no shape
  recognition).
- Strokes as vector paths in page coordinates, anchored to page elements at a moment of the DOM recording, so they
  survive scroll and resize in the replay.
- The recording format extended additively (older viewers ignore sketches).
- Live rendering for all viewers in the drawer's colour, erasable by its author, listed beside notes.
- The agent-side view: label, anchor elements, a rendered image with the sketch, and the vector paths, readable from
  the recording.
- Out of scope: audio; freehand recognition.

## Done when

- The design (plan) is approved by the operator; implementation issues are filed from it.
- Ultimately (project done-when): on a real cast of xianxu.dev, a viewer sketches and labels, everyone sees it live, it
  survives scroll and resize in the replay, and a coding agent reading the recording builds from it.

## Plan

- [x] Claim and start planning; read broadcast/webcast atlas and implementation before designing.
- [x] Write the durable design with streaming strokes, Undo/Redo, required label, coordinates/anchors, additive records, live/replay rendering and agent evidence.
- [x] Operator reviewed and approved the detailed plan, with drawing privilege/console/pressure revision.
- [x] Filed unclaimed dependent implementation slices: #751 recording → #752 drawing → #753 agent export → #754 acceptance; #751 depends on #750. No implementation started.

## Log

### 2026-10-20
- 2026-10-20: closed — Operator approved detailed design via ops:0; revision includes driving-derived single drawer, casting-Chrome controls and pressure. Published unclaimed dependent slices #751-#754 verified against origin/main; issue schema and git diff checks pass. Docs-only design; no feature implementation claimed.; review verdict: SHIP

- Claimed in pair:9 and ran start-plan. Read current cast/recording/Board/renderer/query code and the cast-sketch project in ops; no project file edited.
- Operator direction relayed by ops:0 supersedes the TL's initial local-draft/atomic-save choice: stream strokes immediately, Undo/Redo, pointer colour; finish the design without further questions and collect decisions in the plan.
- Read-only delegated recording/query exploration confirmed that unknown kinds belong in events.jsonl for older-reader compatibility; existing typed known records drop unknown fields. Plan avoids claiming universal preservation (ARCH-EVOLVE).
- Proposed a held drawer scene, one immutable source moment, shared anchor projection, author lease, acknowledged ordered chunk writes, pure Undo/Redo transitions and replay-based image export. These are recommendations for operator review, not approval or implementation (ARCH-DRY, ARCH-PURE, ARCH-ORDER).
- One advisory fresh-eyes pass led to explicit initial snapshot synchronization, recorded interruption/expiry, finalization-capacity reserves and interrupted-label recovery. Retained Decision needed sections per the operator's explicit request; no automated plan-quality gate or implementation started. Added the general prevention rule to workshop/lessons.md.

## Revisions

- 2026-10-20: added the detailed proposed design and operator's streaming/Undo/Redo direction. Earlier Problem/Spec remains as project context; author erasure is proposed through Undo, with post-finish editing explicitly awaiting a decision. Design scope remains design-only until operator approval and implementation issue filing.

### 2026-10-20 — approved scope revision

- Operator approval relayed by ops:0: replace viewer-first access with driving-derived single-drawer authority. Casting-Chrome drawing and pen pressure/Apple Pencil are required; pointer/view spectators cannot draw, pointer spectators retain pointing. All other recommendations approved.
- Updated current plan sections and appended the approval Revision, including ordered handoff fencing, independent pending-label completion and pressure-faithful live/replay/export. Filed #751, #752, #753 and #754 on origin, unclaimed for TL dispatch, with dependencies #750 → #751 → #752 → #753 → #754. No implementation work started.
