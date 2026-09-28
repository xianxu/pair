---
id: '000026'
status: done
created: 2026-05-28
updated: 2026-05-28
actual_hours: 0.4
---

# Mouse scroll does not work inside pair with Codex

## Problem

Inside `pair codex`, mouse wheel scroll does not work in the Codex
agent pane. The likely failure mode is in the wrapper / zellij input
path: Codex either does not receive mouse wheel sequences, receives a
different protocol than it expects, or pair-wrap consumes/translates
the sequences incorrectly.

Diagnosis from live process/log inspection: this can happen when the
Codex pair session was launched before the current launcher started
forcing `codex --no-alt-screen`. Codex's default alternate screen
emits `CSI ?1049 h/l`; zellij pane scrollback is intentionally empty
for alternate-screen applications, so mouse wheel has nothing to scroll.

Revision: user initially reported that uninstalling Homebrew `pair` did
not fix the issue, so the issue was reopened. On live retest, mouse
scroll does work in `pair codex`; no code change is needed.
