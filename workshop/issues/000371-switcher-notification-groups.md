---
id: 000371
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'b12c48bdd8d39bceeb6e1e464abb4b5ed89d8023' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T13:27:30-07:00
flow: {kind: quick, provenance: inferred, spec: "70635be1", done: "698d336d"}
---

# Group switcher notifications under their originating slot

## Problem

Slots :1+ are indented two spaces in the switcher, placing their labels at
the same indentation as notification text. This makes notifications hard to
distinguish from slots and obscures which row owns each message.

## Spec

Group notifications immediately beneath their originating slot. The proposed
layout uses tree connectors aligned with the owning slot label, retaining the
two-space indentation for slots :1+:

```text
pair:0
├─ notif 1
└─ notif 2
  pair:1
  pair:2
  ├─ notif 1
  └─ notif 2
```

Use `├─` for messages before the last and `└─` for the last or only message.
Derive indentation from the owning row rather than a fixed notification
prefix (ARCH-DRY). Preserve message order and existing navigation/click ownership.

## Done when

- Notifications are visually grouped under the correct slot at both indentation levels.
- Zero, one, and multiple notifications render correctly, ignoring empty messages.
- Keyboard navigation skips notification rows; clicking one targets its owning slot.
- Rendering tests cover the grouping, narrow-width clipping, and viewport behavior.
- A selected group taller than the viewport keeps its owning slot label visible.

## Plan

- [x] Add rendering regressions for both slot depths, empty messages, clipping, and ownership.
- [x] Render tree connectors using the row's existing presentation indent; update atlas.
- [x] Run couchtty tests including race checks; submit to the SDLC close review.

## Log

### 2026-10-01

Captured the operator's proposed notification grouping. Operator requested task
creation followed by claiming; implementation has not started.

Operator authorized implementation. This is a pure rendering change (ARCH-PURE)
using existing row identity and indentation (ARCH-DRY, ARCH-PURPOSE). Existing
terminal bounds and sanitization remain authoritative (ARCH-CONSTRAINTS,
ARCH-SECURE); the change holds no new state between events (ARCH-ORDER) and
creates no durable runtime artifacts (ARCH-FUNERAL).

Spec review: no blockers. Connectors include the existing two-cell selection
gutter and align at column `2 + entry.Indent`. Keep matching-description rows
before notifications. Determine the last nonempty message before viewport
clipping; test normal and focus views. These clarify the proposed layout.

Rendering regressions failed on the old fixed prefix, then passed with tree
connectors. An oversized selected-group regression also failed before bounding
the scroll start by the owner row. Full couchtty tests and `make build` passed.
The full race check exposed a pre-existing grouped-tab fixture race: it mutates
pane identity after Console.Run starts. Move configuration before startup and
rerun the race suite; production pane identity remains unchanged.

Verification after fixture repair: full couchtty tests and race suite pass;
focused grouped-click race regression passes ten runs. `make build` and
`git diff --check` pass. Updated grouped-rendering fixtures and atlas. The
SDLC close gate owns the final review.

## Revisions

### 2026-10-01 — Preserve ownership for oversized groups

Regression exploration reproduced an existing scroll-window bug: anchoring the
window at the selected group's end can hide the selected slot label when its
messages exceed the viewport. Bound the start by the selected row's start so
grouping remains intelligible; add a regression and retain the existing
viewport budget. This extends the viewport acceptance criterion.

### 2026-10-01 — Repair verification fixture startup ordering

The race-suite failure is in the grouped-tab test setup, which writes pane
identity concurrently with console startup. Configure the fixture before
starting its goroutine to make the required race verification reliable.
