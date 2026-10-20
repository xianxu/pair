---
id: 000741
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd70240280b41d15f688b8c731cd7d109b883f3b6' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: live concurrent resume conformance with an owned hub caller

## Problem

Follow-up to pair#726. Tier A covers the real broker socket, CLI and console recovery/launch/adoption with three parked slots and stateful fake native dependencies. It does not qualify the same concurrent batch against a running native Couch supervisor with a fixture-owned live hub caller identity. The operator's recovery batch exposed client exchange timeouts that narrower tests missed.

## Spec

Add gated live conformance for concurrent lifecycle admission/resume, using the production CLI and supervisor. The fixture owns its supervisor, disposable repository/worktrees, parked conversations and live hub caller registration; it must establish that caller itself rather than borrow an operator session or inherited binding. Exercise startup-prewarmed and cold repository batches. Preserve fresh authority proofs before every effect. Keep the test opt-in and isolate all effects to fixture-owned resources, with bounded startup, operation polling and teardown.

## Done when

- One opt-in command starts the owned fixture and concurrently resumes at least three parked slots through their real CLI route; every request returns a typed result inside its exchange budget and every intended resume succeeds, without final warming failures, uncertain outcomes or dropped requests. Internal typed warming responses are expected when held cold: they must guarantee nothing queued, carry a retry-after, and lead to bounded automatic CLI retry with the same request identity.
- Startup-prewarmed and held-cold batch variants run, and the fixture proves it is exercising its own live hub caller identity. The held-cold fixture waits for actual warming responses before releasing discovery; all three CLIs then succeed, with no synchronous native authority fallback at admission.
- Native effect outcomes and ownership are observed directly; teardown removes only fixture-owned resources even after a failed or interrupted run.
- The documented live command is run successfully, and the repository-warming regression is demonstrated to fail when batching/prewarm is removed where applicable.

## Plan

- [ ]

## Log

- Filed unclaimed per the ops:0-approved split recorded in #726's CHECKPOINT. Tier A lands with #726 (PR #487); this item owns Tier B live conformance and its independent design/approval.
