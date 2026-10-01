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

## Plan

- [ ] Add rendering regressions for both slot depths, empty messages, clipping, and ownership.
- [ ] Render tree connectors using the row's existing presentation indent; update atlas.
- [ ] Run couchtty tests including race checks and close with the SDLC review.

## Log

### 2026-10-01

Captured the operator's proposed notification grouping. Operator requested task
creation followed by claiming; implementation has not started.

Operator authorized implementation. This is a pure rendering change (ARCH-PURE)
using existing row identity and indentation (ARCH-DRY, ARCH-PURPOSE). Existing
terminal bounds and sanitization remain authoritative (ARCH-CONSTRAINTS,
ARCH-SECURE); the change holds no new state between events (ARCH-ORDER) and
creates no durable runtime artifacts (ARCH-FUNERAL).
