---
id: 000143
status: working
deps: []
github_issue:
created: 2026-08-18
updated: 2026-08-18
estimate_hours:
started: 2026-08-18T22:02:47-07:00
---

# Keep agent session discovery alive after startup timeout

## Problem

The asynchronous session watcher gives up after 60 seconds. Agents such as
Codex can create their transcript only after their first interaction, so an
agent left idle for longer than the startup window never gets a persisted
session ID. The context meter then has no transcript to read and the frame omits
context-window usage for the rest of the session.

## Spec

Treat the existing timeout as the end of fast startup discovery, not the end of
the watcher. When a fresh agent PID is available, continue discovery at a
low-frequency 60-second interval while that process is alive. Stop immediately
when a poll observes that the process has exited (within one slow-poll interval),
and preserve the existing bounded timeout when no fresh PID can be established.

Apply the lifecycle behavior uniformly to every asynchronous agent supported by
`sessionwatch` (Codex, Agy, and Muse). Claude supplies its session ID
synchronously and must remain unaffected.

## Done when

- A live supported agent can acquire and persist a session ID after the initial
  discovery timeout.
- Post-timeout discovery polls no more often than once every 60 seconds by
  default.
- The watcher exits within one slow-poll interval after the bound agent process
  exits, and still times out when no fresh PID exists.
- Automated tests cover delayed discovery for Codex, Agy, and Muse plus both
  exit paths.

## Plan

- [ ] Test `Run` with the existing fake-clock/stateful-runtime seam, scheduling
  transcript and process-state transitions to guard the fast-to-slow cadence,
  every `AgentSpec`, PID-bound exit, and PID-less timeout.
- [ ] Change the watcher loop to transition from fast polling to lifecycle-bound
  slow polling, using the existing injected runtime seam (ARCH-PURE,
  ARCH-MOCK).
- [ ] Run focused and repository-wide verification; confirm the synchronous
  Claude launch path is unchanged (ARCH-PURPOSE, ARCH-DRY).

## Log

### 2026-08-18

- Root cause: the live Codex transcript appeared after the watcher's fixed
  60-second deadline, leaving `config-<tag>-codex.json` absent even though the
  transcript parser supported the current event format.
- Design approved by the operator: retain fast startup polling, then poll every
  60 seconds for the lifetime of the bound agent process. Apply this to all
  asynchronous agent specs rather than special-casing Codex.

## Revisions

### 2026-08-18 — Plan-quality review

- Clarified that process death is observed at the next 60-second slow poll,
  rather than promising impossible immediate detection during a blocking sleep.
- Recast the test plan as a function-level fake-clock strategy for `Run`.
