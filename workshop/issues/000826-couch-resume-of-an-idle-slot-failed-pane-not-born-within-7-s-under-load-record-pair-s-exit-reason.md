---
id: 000826
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'df950c402a3bf4e24bfc24e2e16d996e64b5d746' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --resume of an idle slot failed: pane not born within 7 s under load; record pair's exit reason

## Problem

## Spec

From the 10-22 full-shutdown recovery: resuming idle slot ariadne:1 (`couch --resume ariadne:1`) failed with "Pair
exited before registering (exit 1)": `waiting for pane birth after 7s: agent pane not born` / `context canceled
[pane-birth wait consumed the budget: pane birth 6.957s ... 0 ownership polls]` / `recovery terminal pane birth is not
confirmed [remote job: queued 2ms, prepare 3.681s, operation 14.423s]` (attempt start-b9265e1e5d80b85d, recorded in
`~/.local/share/pair/couch/launch-diagnostics/failures.json`, failed_at 21:56:54). The machine was loaded (15 slots
resuming and rebuilding on a new Go). 15 other resumes the same evening succeeded.

Diagnose: is the 7 s pane-birth budget too tight under load (prepare alone took 3.7 s), or did pair exit for another
reason (its exit 1 cause isn't in the record)? Record pair's own exit reason in the failure record either way.

## Done when

- The failure record includes pair's exit reason (test).
- The cause is classified; if it's the budget, it's derived from measured load with a re-measure note, not raised
  blindly.

## Plan

- [ ]

## Log

### 2026-10-22
