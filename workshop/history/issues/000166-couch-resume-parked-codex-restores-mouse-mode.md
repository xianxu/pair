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
