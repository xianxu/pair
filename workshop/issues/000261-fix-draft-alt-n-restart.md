---
id: 000261
status: open
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Fix draft Alt+N restart confirmation

## Problem

During #255 smoke testing, `Alt+N` in the draft Neovim pane opened the restart
confirmation, but confirming it had no visible effect. Restarting from the
switcher menu still works; `Alt+Shift+N` remains the separate new-context
restart command.

## Spec

Trace the draft-pane `Alt+N` path through command-bar confirmation, result
handling, and child relaunch. Fix the missing action or error reporting while
preserving the distinction between restarting the draft process in place and
restarting Pair with new context.

## Done when

- Confirming `Alt+N` relaunches the draft process in place.
- Canceling leaves the existing draft process untouched.
- Restart failures show an actionable error instead of silently returning.
- Regression coverage distinguishes this path from `Alt+Shift+N`.

## Plan

- [ ] Reproduce the confirmation path and capture key/result and lifecycle state.
- [ ] Fix result handling or relaunch, using the switcher path as a reference.
- [ ] Add success, cancel and failure tests; run an isolated smoke session.

## Log

### 2026-09-15

Filed as a follow-up to #255 from operator smoke testing. `Alt+N` reaches the
draft confirmation but confirming with `Y` appears to do nothing.
