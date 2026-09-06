---
id: 000166
status: punt
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T17:08:04-07:00
---

# Mouse scroll stops after resuming parked Codex agent

## Problem

After parking and resuming a Codex thread through Couch, Zellij can continue to
report a growing scrollback count while mouse and programmatic `scroll-up`
remain at offset zero. A parked/resumed Claude thread did not reproduce the
failure. The exact Codex native session binding was preserved; restoring the
precise pre-park viewport is explicitly out of scope.

## Spec

## Done when

- A repeatable Codex park/resume reproduction identifies the exact terminal
  state transition that leaves Zellij unable to navigate retained history.
- Mouse-equivalent and programmatic Zellij scrolling work after the resumed
  Codex redraw without regressing the Couch status surface.

## Plan

- [ ] Reconfirm the operator-visible failure before changing terminal policy;
  treat a non-recurrence as evidence that the original event was transient.
- [ ] Pin the confirmed transition through a disposable Zellij conformance
  test, then design and implement the smallest root-cause fix.

## Log

### 2026-09-01

Investigation only; no production code changed. The resumed process used the
exact saved Codex session ID, whose transcript contained the apparently missing
assistant response. Pair's existing split-safe visible-output filter still
removes Codex synchronized-output (`CSI ? 2026 h/l`) and focus-event
(`CSI ? 1004 h/l`) controls; KKP filtering remains opt-in. Live Codex raw
streams contained complete scrolling-region (`DECSTBM`) set/reset sequences,
not a visibly bisected escape. A disposable Zellij 0.44.3 probe retained 200
lines in its full dump but left `scroll-up` fixed at the bottom after a reduced
scrolling region, whereas ordinary output advanced from line 181 to 179. Claude
resume working and the operator's uncertainty about recurrence mean the
Codex/Couch interaction is suggestive, not yet sufficient authority for a
production change. Parked pending another natural reproduction (`ARCH-PURPOSE`).

### 2026-09-06 — re-evaluated against `pair#172`, and it stays punted

`pair#172`'s Done-when required this issue to be re-evaluated once mouse
ownership landed, rather than left punted by default. The answer is that #172
does not touch it, and the reason is worth recording so the next reader does not
re-open the question.

**What #172 changed:** couch enables click reporting (`?1000;?1006`) for ITSELF,
re-asserts it on every paint while no child holds tracking, and stands back the
moment a child enables `?1000`/`?1002`/`?1003`. It never writes the child's
modes — `ptychild` replay already owns that across a park/resume.

**Why that is orthogonal to this failure.** The symptom here is that Zellij's
`scroll-up` stays pinned at the bottom after a resumed Codex redraw, with the
scrollback count still growing. The 2026-09-01 probe reproduced it with a
disposable Zellij 0.44.3 and a **reduced scrolling region** (`DECSTBM`) — no
mouse mode involved, and `scroll-up` is a Zellij action rather than a terminal
mouse report. #172's tracking policy cannot reach it.

The connection assumed when this was punted — "explicit mode ownership should
subsume it" — was wrong: mouse MODE and scrolling REGION are different terminal
state, and only the first is now owned.

**Still punted**, unchanged: the reproduction is Codex-specific, the operator has
a working alternative, and the remaining suspect (`DECSTBM` interacting with
Zellij's history navigation) is a Zellij-behaviour investigation rather than a
couch one.

