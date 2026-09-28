---
id: '000045'
status: done
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 5
actual_hours: 7
---

# adaptation flight recorder and pair-doctor

## Problem

`pair` adapts each agent harness (claude/codex/agy) across 7 integration aspects
documented in `atlas/how-to-bring-up-a-new-harness-cli.md`. Harnesses update
constantly and silently break these adaptations — e.g. codex renames its picker
confirm string and the overlay detector goes quiet, leaking a stray newline into
the prompt (the #000042 class of bug). Nothing surfaces that drift today: the unit
tests freeze our *assumptions* about each harness and validate matchers against
frozen strings, so they pass forever even after the live harness moves.
