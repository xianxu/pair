---
id: 000680
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '58c4c944e80b82f5ddb976184753e83f94a1c620' # card fields mirrored from issue-cards; edit via sdlc
---

# pair doctor resume: daily read-only check that every slot's resume target resolves on real data

## Problem

pair#676: every Claude resume refused once ~/.claude/projects crossed 10,000 entries (a discovery cap reached resume through #556). Small test fixtures could not show it; only a run against the real tree did (pair:5 proved the fix with a read-only conformance run, ~0.9 s per slot). Nothing checks resume against real data routinely, so the operator found it by a failed relaunch.

## Spec

A read-only command (e.g. `pair doctor resume`) that, for every Couch slot, resolves the slot's recorded native resume target exactly as a relaunch would, without launching an agent or writing anything, and reports per slot: resolves / refuses (with the reason) / no recorded target. Exit non-zero if any slot that has a target refuses. Cheap: bounded IO, about a second per slot. Intended to run daily (by hand from the hub until a Couch schedule exists; operator idea in ops qa 2026-10-18-16 checklist), with a refusal raising a notify.

## Done when

- On the live machine, the command lists every slot with its outcome, in seconds, with no writes (verified by a before/after listing of state dirs).
- A fixture with an unresolvable target makes it exit non-zero and name the slot and reason.
- Re-running main from before #676 against today's tree would have reported the refusal (shown with the retained baseline, or argued from the code path).

## Plan

- [ ]

## Log

### 2026-10-18
