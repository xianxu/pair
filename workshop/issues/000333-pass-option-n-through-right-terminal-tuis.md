---
id: 000333
status: working
deps: []
github_issue:
created: 2026-09-26
updated: 2026-09-26
estimate_hours:
started: 2026-09-26T16:51:02-07:00
flow: {kind: quick, provenance: inferred, spec: "a7856128", done: "82d008d4"}
---

# Pass Option+n through to right-terminal TUIs

## Problem

When the right terminal pane is focused on a TUI program, Pair currently
intercepts `Option+n` as its own reload action. This prevents TUIs that bind
that key from receiving it.

## Spec

- In the focused right-terminal path, pass `Option+n` through unchanged to the
  program, including shell and full-screen TUI programs.
- Keep current Pair handling for the right terminal's shell/non-TUI state and
  for every other pane.
- Reuse the existing TUI detection and input-routing boundary; do not add a
  second independent classification.

## Done when

- [x] Right-terminal TUIs receive `Option+n` through both Pair and Couch input
  routing layers.
- [x] Right-terminal shells/non-TUI states retain Pair's existing `Option+n`
  behavior.
- [x] Other panes retain their existing `Option+n` behavior.
- [x] Routing regressions cover the TUI passthrough and neighboring retained
  behaviors.
- [x] User-facing keybinding documentation reflects the focused right-terminal
  exception.

## Plan

- [x] Locate the existing focused-pane and TUI input-routing decision.
- [x] Route `Option+n` through whenever the right terminal is focused.
- [x] Add routing regressions for TUI passthrough and non-TUI/other-pane
  behavior.
- [x] Update the keybinding documentation.
- [x] Add the Couch interceptor exception for a focused full-screen child.

## Revisions

### 2026-09-26 — Inner-pane focus evidence

Standalone Pair smoke passed. Replace Couch's outer-screen inference with a
bounded exact-session client query plus live pane registry. Forward right-pane
restart chords to Pair, retain Couch confirmation for known other panes, and
show a notice on uncertain focus. Multi-client ambiguity uses the switcher.

### 2026-09-26 — Couch routing boundary

The smoke test showed that Couch intercepts `Alt+n` before Pair's terminal
router. Extend the implementation and regression coverage to Couch's existing
focused-child interceptor, passing only `Alt+n` through for a full-screen child;
keep `Ctrl+Alt+n` and shell/non-TUI relaunch behavior unchanged.

### 2026-09-26 — Shared full-screen policy

Pair and Couch must use the same focused-right-terminal rule: a full-screen
program receives every recognized chord except the explicit layout/tab controls
(`Alt+Shift+Enter`, `Alt+Shift+t`, `Alt+Shift+Left/Right`, and `Alt+k`).
`Alt+n` is draft-scoped in Pair and passes through the right terminal even when
the child is a shell.

## Log

### 2026-09-26 — Couch inner-focus correction

- Standalone Pair smoke passed, reported by the operator. Couch smoke exposed
  that the outer alternate screen belongs to Zellij, not the right-pane TUI.
- Replace that inference with an exact-session, one-second client-focus query
  for restart candidates, matched against the live terminal pane registry.
  Right-terminal candidates reach Pair; other panes keep Couch confirmation.
  Ambiguous/unavailable focus consumes the candidate with a notice; the
  switcher remains usable for relaunch. No query for ordinary keys (ARCH-ORDER,
  ARCH-DRY). Multi-client focus is explicitly unsupported for this shortcut.
- Focus regression first reproduced the draft failure. Couch package suites
  and focused race checks passed. Fresh review found absence in the pane
  registry was not negative proof; added missing/malformed registry regressions
  and now require positive draft/agent identification before relaunch fallback.

### 2026-09-26

- Approved design: right-terminal TUI programs receive `Option+n`; shell and
  all other pane paths retain their existing Pair behavior. Reuse the existing
  TUI routing boundary and cover it with focused regressions.
- Implemented the exception in the existing alternate-screen routing path.
  Added workbenchshortcut and termcmd regressions, updated key help and README,
  and passed the full `workbenchshortcut`, `termcmd`, and `wrapcmd` package
  suites.
- Debugging the smoke test found that Couch intercepts `Alt+n` before Pair's
  `pair term` router. The Couch interceptor is the earlier routing boundary;
  the fix must pass Alt+n through when its focused child owns the alternate
  screen, while retaining relaunch for shells/non-TUI children (ARCH-DRY).
- Added the Couch-layer exception and a real input-loop regression with a fake
  alternate-screen child. The Couch, Couch command, terminal command, and
  shortcut package tests pass.
- Broadened the policy so Pair and Couch share the same full-screen allowlist;
  generated draft keymaps now mark `Alt+n` as draft-scoped.
