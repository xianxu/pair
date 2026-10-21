---
id: 000750
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f761356cc2fd006a5419f050ab3390aef065730e' # card fields mirrored from issue-cards; edit via sdlc
---

# cast-sketch: design the ✎ drawing tool for the Couch web cast (strokes + label, element-anchored vector paths, additive recording format, live view, agent-side view)

## Problem

Project cast-sketch (ops `teams/devinfra/projects/cast-sketch/`, created 10-19, started 10-20 by the operator: "I want
to use it as part of the design process for the echo project"). The Couch web cast has a pointer and an annotation
tool (bounding boxes with notes); viewers can't show what a page should look like. A third tool, ✎ drawing, adds
sketches.

## Spec

The design settles these; it is operator-reviewed before code.

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

- [ ]

## Log

### 2026-10-20
