---
id: 000026
status: done
deps: [000010]
created: 2026-05-28
updated: 2026-05-28
related: [cmd/pair-wrap/main.go, zellij/config.kdl]
actual_hours: 0.3
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

## Spec

Mouse wheel in the Codex pane should behave like running Codex
directly in the same terminal: wheel up/down should scroll the Codex
TUI when Codex has enabled mouse tracking, without breaking pair's
copy-on-select behavior or existing key remaps.

## Plan

- [x] Inspect pair-wrap input forwarding and mouse/escape sequence
      handling.
- [x] Inspect zellij config for mouse mode and copy-on-select behavior.
- [x] Decide whether a pair-wrap regression test is needed: no, because
      the failure is not stdin translation; it is a stale Codex process
      running in alternate-screen mode.
- [x] Implement the narrow fix: already present in current `bin/pair`
      before this issue; no code delta needed here.
- [x] Run verification appropriate to this diagnosis: inspect live
      process args and raw scrollback logs for alt-screen toggles.
- [x] Update atlas notes if the fix changes agent input semantics: no
      new atlas change needed because the launcher behavior is already
      documented in `bin/pair`.

## Log


- 2026-05-28: closed — Inspected live process args and raw scrollback logs: pair-codex has --no-alt-screen and no CSI ?1049; stale pair-brain-politics1 codex lacks --no-alt-screen and raw log contains CSI ?1049 h/l.
### 2026-05-28

- Started investigation from user report: mouse scroll does not work
  inside pair with Codex.
- Confirmed `bin/pair` in the repo already forces `--no-alt-screen` for
  Codex unless `PAIR_CODEX_ALT_SCREEN=1` is set.
- Confirmed Homebrew `pair` 1.19 does not contain the
  `--no-alt-screen` Codex launcher block.
- Inspected live processes: `pair-codex` is running
  `codex resume ... --no-alt-screen`, but the older
  `pair-brain-politics1` Codex session is running `codex resume ...`
  without `--no-alt-screen`.
- Inspected raw scrollback logs: `scrollback-codex-codex.raw` has no
  alt-screen toggles, while `scrollback-brain-politics1-codex.raw`
  contains `CSI ?1049 h/l` and `CSI ?1007 h/l`. That is the concrete
  reason mouse wheel cannot scroll zellij history in the older session.
- No new code change was needed in this pass; source already has the
  fix. Remedy is to restart the affected Codex pair session so it is
  relaunched by the current `bin/pair` with `--no-alt-screen`.
