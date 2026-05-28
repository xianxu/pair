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

Diagnosis from live process/log inspection: this can happen when the
Codex pair session was launched before the current launcher started
forcing `codex --no-alt-screen`. Codex's default alternate screen
emits `CSI ?1049 h/l`; zellij pane scrollback is intentionally empty
for alternate-screen applications, so mouse wheel has nothing to scroll.

Revision: user uninstalled the Homebrew `pair` and retested with the
workspace version; mouse scroll still does not work in `pair codex`.
The stale-launcher diagnosis was insufficient and the issue must remain
open until live behavior is verified.

## Spec

Mouse wheel in the Codex pane should behave like running Codex
directly in the same terminal: wheel up/down should scroll the Codex
TUI when Codex has enabled mouse tracking, without breaking pair's
copy-on-select behavior or existing key remaps.

## Plan

- [x] Inspect pair-wrap input forwarding and mouse/escape sequence
      handling.
- [x] Inspect zellij config for mouse mode and copy-on-select behavior.
- [ ] Determine whether zellij is forwarding wheel events into Codex
      because Codex enables mouse/protocol modes even under
      `--no-alt-screen`.
- [ ] Determine whether pair-wrap should filter Codex mouse-mode enable
      sequences, translate wheel events, or configure Codex/zellij
      differently.
- [ ] Add a focused regression test if the fix is in pair-wrap stream
      filtering/translation.
- [ ] Implement the narrow fix.
- [ ] Verify with a live `pair codex` scroll test, not just process/log
      inspection.
- [ ] Update atlas notes if the fix changes agent input semantics.

## Log


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

### 2026-05-28 retest

- User uninstalled the Homebrew-installed `pair` and still reproduces
  mouse scroll failure in `pair codex`.
- Reopened issue. Prior close was premature because it relied on
  indirect evidence and did not verify live mouse-scroll behavior.
