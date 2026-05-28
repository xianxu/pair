---
id: 000026
status: working
deps: [000010]
created: 2026-05-28
updated: 2026-05-28
related: [cmd/pair-wrap/main.go, zellij/config.kdl]
---

# Mouse scroll does not work inside pair with Codex

## Problem

Inside `pair codex`, mouse wheel scroll does not work in the Codex
agent pane. The likely failure mode is in the wrapper / zellij input
path: Codex either does not receive mouse wheel sequences, receives a
different protocol than it expects, or pair-wrap consumes/translates
the sequences incorrectly.

## Spec

Mouse wheel in the Codex pane should behave like running Codex
directly in the same terminal: wheel up/down should scroll the Codex
TUI when Codex has enabled mouse tracking, without breaking pair's
copy-on-select behavior or existing key remaps.

## Plan

- [ ] Inspect pair-wrap input forwarding and mouse/escape sequence
      handling.
- [ ] Inspect zellij config for mouse mode and copy-on-select behavior.
- [ ] Add a focused regression test for mouse wheel bytes through
      pair-wrap if the bug is in translation.
- [ ] Implement the narrow fix.
- [ ] Run pair-wrap tests and rebuild `bin/pair-wrap` if needed.
- [ ] Update atlas notes if the fix changes agent input semantics.

## Log

### 2026-05-28

- Started investigation from user report: mouse scroll does not work
  inside pair with Codex.
