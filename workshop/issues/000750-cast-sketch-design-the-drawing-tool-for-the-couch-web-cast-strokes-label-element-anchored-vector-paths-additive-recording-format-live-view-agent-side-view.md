---
id: 000750
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a8cd0695c921167be4fabb5d012d675eaa3384e9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T19:40:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
---

# cast-sketch: design the ✎ drawing tool for the Couch web cast (strokes + label, element-anchored vector paths, additive recording format, live view, agent-side view)

## Problem

Project cast-sketch (ops `teams/devinfra/projects/cast-sketch/`, created 10-19, started 10-20 by the operator: "I want
to use it as part of the design process for the echo project"). The Couch web cast has a pointer and an annotation
tool (bounding boxes with notes); viewers can't show what a page should look like. A third tool, ✎ drawing, adds
sketches.

## Spec

The design settles these; it is operator-reviewed before code.

Detailed proposal: [cast-sketch plan](../plans/000750-cast-sketch-plan.md). The operator's latest direction is live streaming from the first stroke, Undo and Redo, using the viewer's pointer colour. The proposed label step runs when leaving the tool; unfinished intent stays explicit. Source-moment anchoring, interruption/retry behavior, additive recording, replay and agent image/paths are designed together. All unresolved product choices are marked **Decision needed** in the plan for one operator review.

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
- [ ] Operator reviews the collected Decision needed items and approves the detailed plan.
- [ ] File implementation issues from the approved slices; no code before approval.

## Log

### 2026-10-20

- Claimed in pair:9 and ran start-plan. Read current cast/recording/Board/renderer/query code and the cast-sketch project in ops; no project file edited.
- Operator direction relayed by ops:0 supersedes the TL's initial local-draft/atomic-save choice: stream strokes immediately, Undo/Redo, pointer colour; finish the design without further questions and collect decisions in the plan.
- Read-only delegated recording/query exploration confirmed that unknown kinds belong in events.jsonl for older-reader compatibility; existing typed known records drop unknown fields. Plan avoids claiming universal preservation (ARCH-EVOLVE).
- Proposed a held drawer scene, one immutable source moment, shared anchor projection, author lease, acknowledged ordered chunk writes, pure Undo/Redo transitions and replay-based image export. These are recommendations for operator review, not approval or implementation (ARCH-DRY, ARCH-PURE, ARCH-ORDER).

## Revisions

- 2026-10-20: added the detailed proposed design and operator's streaming/Undo/Redo direction. Earlier Problem/Spec remains as project context; author erasure is proposed through Undo, with post-finish editing explicitly awaiting a decision. Design scope remains design-only until operator approval and implementation issue filing.
