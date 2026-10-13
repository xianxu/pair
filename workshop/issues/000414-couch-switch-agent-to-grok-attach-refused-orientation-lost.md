---
id: 000414
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '817b3074837475caba97dc10bd0e222365c75f92' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch switch-agent to grok: attach refused, orientation lost

## Problem

Switching a live Couch thread from Claude to Grok (pair#410's harness) started
the Grok incarnation correctly, but the console's follow-up attach was refused
and the switch's orientation prompt was never delivered.

Observed 2026-10-08 on pair:5 (thread `23e53ad8c4dfb803/1-pair-20`, Couch built
from #410 at `a37b9bee`):

- Status line: `cleanup after a failed attach of 1-pair-20: attach
  record/handle process identity mismatch` then `abort started: start handle is
  unavailable`. The attach check is `couchtty/console.go` ("attach record/handle
  process identity mismatch"); the cleanup is `couchcore.AbortStarted` reached
  from `couchcmd/run.go` after a failed attach.
- The thread is healthy afterwards: the registry record (pid 10777, identity
  `1791480803.22827`, shape `cold-resume`, extra args `--minimal --permission-mode
  bypassPermissions`) matches the live `pair --couch-session-v1 resume 1-pair-20
  --layout3`, and grok runs under it with a minted `--session-id`.
  `couch --show pair:5` reports live, nothing to do.
- Orientation was lost: the new grok session's first user message is the
  operator's own `hello`; the wrapper trace has no orientation events.
- The console attaches only after switch-agent reports success, passing that
  switch's own `StartResult` (record + handle). So at attach time the handle the
  launch returned was nil or disagreed with its record; the follow-up "start
  handle is unavailable" points to nil.
- Not reproduced on a Codex → Claude switch (that one hit only the benign
  "operator input interrupted automatic orientation"). One sample each, so it may
  be grok-triggered rather than grok-only.
- No start-sequence evidence survives: `threadstore/events` stops in September
  and the thread has no file under `threadstore/records`.

## Spec

Reproduce with a Claude → Grok switch on a scratch slot while tracing Couch's
tracked launch (`launchTrackedThread` → `SwitchAgentResult.Started()` → console
`attach`). Find which handle reaches the attach and why it is nil or mismatched:
for example, whether the grok profile's pair child acknowledges readiness before
the handle is bound, or re-execs. Fix the cause rather than loosening the
identity check. If orientation is skipped as a consequence, make that failure
visible as an orientation notice.

## Done when

- A Claude → Grok switch-agent attaches the new incarnation without the identity
  notice and delivers orientation (the grok session's first user message is the
  orientation prompt).
- A regression test pins the root cause in couchcore/couchtty.
- If the cause is not grok-specific, the same switch to another agent is covered.
- Carried from #410's Done-when: a Couch-hosted grok thread parks and
  cold-resumes, and `doctor/doctor.sh` on a live grok session shows
  `return-remap`, `session-id` and `slug-parse` firing (the #410 smoke sessions'
  adapt logs were absent, so no tally exists yet).

## Plan

- [ ] Reproduce a Claude → Grok switch-agent on a scratch slot with Couch tracing; capture which `StartResult` reaches the console attach.
- [ ] Find why the handle is nil or mismatched; fix the cause with a couchcore/couchtty regression test.
- [ ] Verify orientation delivery, then Couch park/cold-resume and the `doctor/doctor.sh` tally on a live grok thread.

## Log

### 2026-10-08
