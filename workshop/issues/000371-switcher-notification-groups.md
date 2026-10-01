---
id: 000371
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'd4cad4f0ba6bb871f77fe6621792fa53755c59f0' # card fields mirrored from issue-cards; edit via sdlc
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


## Log

### 2026-10-01

Captured the operator's proposed notification grouping. Operator requested task
creation followed by claiming; implementation has not started.
