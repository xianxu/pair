---
id: 000169
status: punt
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-03
estimate_hours:
---

# Persist Couch operation failure diagnostics

## Problem

Couch start and resume failures are reported only through a transient panel
notice. The notice can disappear during refresh/navigation and there is no
durable operation diagnostic to inspect afterward. In the observed Pair
failure, the operator saw the action silently fail, while post-hoc evidence
showed a rolled-back thread reservation and a partially launched Pair/Zellij
session; the original subprocess error was unrecoverable.

## Spec

- A failed Couch start, resume, switch, park, or attach operation must leave a
  clearly visible error until the operator explicitly dismisses it or retries.
- The diagnostic must name the operation and target and preserve the actionable
  root cause rather than only a generic failure.
- Refreshes and inventory updates must not overwrite an unacknowledged error.
- Persist enough bounded diagnostic history to investigate a failure after the
  panel state or Couch process is gone, without recording prompt contents or
  other sensitive terminal data.
- Successful operations should clear or supersede related stale failures
  intentionally, not incidentally.

## Done when

- An integration test injects a start/resume subprocess failure and proves the
  error remains visible across inventory refresh and navigation until explicit
  acknowledgement or retry.
- The same failure can be recovered from a bounded durable diagnostic after
  Couch exits and restarts.
- Diagnostics include operation, exact thread/path target, timestamp, and the
  causal error chain while excluding user prompt and terminal payload data.
- Retention and redaction behavior are documented and tested.

## Plan

- [ ] Define ownership, acknowledgement, redaction, and bounded retention for operation failures.
- [ ] Preserve unacknowledged errors independently of ordinary inventory notices.
- [ ] Persist structured failure diagnostics at the operation-dispatch boundary.
- [ ] Add UI lifecycle and restart-level regression tests.

## Log

### 2026-09-03 — punted: the disappearance is fixed, the error text is not

pair#181 M1 fixed the half that actually bit. The observed failure left "a
rolled-back thread reservation and a partially launched Pair/Zellij session",
and the operator "saw the action silently fail" -- because a record couch could
not prove simply produced no row. The projection is total now: a rolled-back
reservation reads `never started`, a half-launched thread reads its own reason,
and `Enter` on either explains rather than doing nothing. Nothing vanishes.

What is NOT fixed is the second half: the original subprocess error is still
transient, so the operator can see WHAT is wrong but not WHY the launch failed.
That is real and unaddressed. It is punted rather than closed because the
remaining value is much lower -- the state is inspectable, which is what made
the original incident unrecoverable -- and because a durable operation journal
is a larger mechanism than the residue justifies today.

Reopen when a start or resume failure needs its error text after the fact.

### 2026-09-01

Captured alongside #168. Live state proved that startup reached Pair readiness
on one attempt and rolled a later reservation back, but Couch retained no
diagnostic explaining why the Zellij/Pair launch died. The transient notice was
not noticed by the operator and could not support later diagnosis.
