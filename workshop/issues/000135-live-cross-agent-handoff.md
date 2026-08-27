---
id: 000135
status: open
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
---

# Live cross-agent handoff

## Problem

The revived #115 path lets a user switch an exited/recent tag to another agent
through continuation-backed recovery, but it deliberately avoids taking over a
currently live session owned by a different agent. The abandoned live handoff
coordinator proved that the shape is useful, but its production quiescence proof
was unsound: the acceptance fake modeled cleanup behavior real zellij did not
provide, so the source could be destroyed and then time out.

Pair still needs a safe live handoff for the case where a provider is degraded,
quota is exhausted, or the current agent can no longer produce the continuation
document itself. The tag should remain the work identity while the source agent
is quiesced, its Pair scrollback/state is parked, and the target agent starts as
the exclusive driver under the same tag.

## Spec

- `pair <agent>` may select a different-agent live row only through an explicit
  source-quiescent handoff flow; it must never attach a target agent to a live
  foreign-agent session.
- Pair enforces one live driver per tag while preserving tag-scoped draft, log,
  queue, and public session identity.
- Quiescence evidence must be observable by the coordinator itself, not only by
  an acknowledgment emitted from a process that may already be gone. At minimum,
  real zellij session absence and recorded pair-wrap/agent/nvim PID liveness
  must be modeled.
- Non-live different-driver rows skip source stop/quiescence and reuse the
  session-name index instead of assuming `pair-<tag>`.
- Handoff writes the target `config-<tag>-<agent>.json` so stale tag configs do
  not shadow the newly minted native session.
- Any transaction lock or recovery claim must be released or quarantined on
  error so a failed handoff cannot permanently disable a tag.
- The old #115 C4 finding about embedded-runtime queue push-front should be
  split first if it is still reproducible, because it affects released Pair
  independently of live handoff.

## Done when

- A real or process-level fake test proves live Claude-to-Codex handoff through
  the public launcher without fake-only cleanup effects.
- Source-stop failures before quiescence are non-mutating; failures after
  quiescence but before target readiness restore tag input state or report an
  exact manual recovery path.
- Different-agent live rows are selectable only through the confirmed handoff
  flow, and different-agent non-live rows continue through the #115 continuation
  path.
- Atlas and README document the live handoff state ownership and failure model.

## Plan

- [ ] Revalidate the abandoned #115 close-review findings against current main.
- [ ] Design the source-quiescent coordinator and recovery journal from current launcher primitives.
- [ ] Implement with stateful zellij/process fakes that cannot emit impossible cleanup acknowledgments.
- [ ] Add live smoke coverage for same-tag cross-agent handoff.
- [ ] Document the completed handoff model.

## Log

### 2026-08-16
- Created from the deferred historical M5/live-handoff scope in #115. #115 now
  closes on repo-agent defaults plus explicit-agent continuation routing for
  exited/recent work; this issue owns the remaining live takeover design.

## Revisions

### 2026-08-26 — align handoff with composite work-thread identity

**Reason:** #149 replaced tag-only storage identity with the durable composite
`{repo_scope, tag}`, centralized tag-bearing paths in `artifactpath`, and made
ThreadStore the lifecycle authority. The original #135 wording predates that
boundary and could be read as permission to coordinate from a global tag or
rename/move files during handoff.

**Delta:** live handoff must select and mutate one exact composite ThreadStore
record, preserve its immutable tag and starting path, and consume the launcher's
resolved artifact bindings. Source and target are incarnations of that thread,
not independent tag owners. #135 owns the cross-agent transition/recovery
journal; #152 supplies the verified whole-incarnation quiescence and parked
evidence it consumes. #153 alone may provision a worktree or rebind mutable
`working_path`. No tag-only lookup, cross-scope artifact fallback, or handoff-
time path rebinding is permitted (ARCH-DRY, ARCH-PURPOSE).
