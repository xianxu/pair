---
id: 000866
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
severity: high
card_mirror: '4cefef838a25a9ab738c4ac9b26f2d1d875b2d45' # card fields mirrored from issue-cards; edit via sdlc
---

# Wait for Zellij readiness before startup probe clients

## Problem

During pair#863's few shared disposable native runs, Zellij 0.45.1 twice panicked at zellij-server/src/lib.rs:1462:26 (Option::unwrap on absent session_data). A startup probe can connect/remove a client before session metadata exists. Pair should investigate a readiness-qualified probe so its own startup observation does not trigger that upstream lifecycle window. Exact standalone causation is not yet proven; this is the bounded Pair-side avoidance issue, not an upstream bug report.

First occurrence: pair#863 source25a6c4847 native Claude fresh primary startup, before a wrapper started; full matrix FAIL456.39s, exact cleanup passed. Retained source-branch evidence:
- workshop/evidence/000863/native-first-fixed.log, SHA2562320f9563c197a28f32160e7150bcf7e0d685be60ed1896f28aa760fbde4e38c
- workshop/evidence/000863/fresh-fixed-claude-zellij.log (full server capture).

Second occurrence: pinned main4053f3da995746efcc1c6296cffbde4a67a7f343, baseline explicitly started its primary through the operator form, then initial worker1 startup failed before any switch-agent step. TestDisposableStartupArguments FAIL42.33s (Claude34.11s); Pair exited1 before registering. The retained server log contains the same lib.rs1462:26 panic at02:44:40 PDT on2026-10-24. Exact owned generations absent and direct child/IO joined; Codex skipped by existing test loop. Evidence workshop/evidence/000863/main-startup-arguments-4053f3da.log, SHA256d17dbd817ceb9225e85e8ec6ab48f5c10b465af8f2e8390eb2d9e17570692497.

The evidence is committed on pair#863's pushed branch (latest evidence7a71f3353), not yet main; this issue has no dependency on #863 and can use git show that branch. Do not confuse these panics with #863 BR-2's different post-startup switch-agent adoption failure.

## Spec

Diagnose Pair's startup probing against an owned disposable Zellij session. Investigate a reliable readiness signal before a probe attaches or detaches a client; preserve caller-owned startup deadlines and conservative no-answer behavior. No timeout increase, hot retry loop, general session ownership weakening, live-fleet manipulation or unrelated Zellij patch.

Installed binary /opt/homebrew/Cellar/zellij/0.45.1/bin/zellij. Verified local tagged source /Users/xianxu/workspace/zellij, v0.45.1 commit efd8fd5a89a20c07a111d248ad7fce53848d2c18. In zellij-server/src/lib.rs1462, regular RemoveClient removes the client then unwraps session_data while it can still be None. Inspect the surrounding FirstClientConnected initialization and Pair's startup session probing paths. Pair arguments/environment were not shown malformed; a timing interaction is the hypothesis.

## Done when

- Retained evidence identifies the Pair probe path and the readiness ordering that can trigger or avoid the early client-removal panic.
- A deterministic stateful-fake/control proves no attach/detach probe before the selected readiness witness; uncertainty remains conservative and bounded by the caller's deadline.
- A bounded shared disposable native qualification shows startup without the panic, with exact owned cleanup; preserve failing captures and do not claim a calibrated flake rate from two occurrences.
- Record whether a standalone upstream reproducer is available; do not file an upstream report without one.

## Plan

- [ ]

## Log

2026-10-24: Filed unclaimed at TL8029db79's direction, for pair:5; do not wait for #863 to land. Both native failures and source pointer retained above. No production workaround in #863.
