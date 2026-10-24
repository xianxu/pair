---
id: 000866
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
severity: high
card_mirror: '05833eb514c1aae1546dffc8f037df94f84edf05' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:48:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
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

- [ ] Approve the readiness/coordination contour in [the durable plan](../plans/000866-zellij-startup-readiness-plan.md), then enter change-code; no estimate before its gate.
- [ ] Implement one readiness authority and bounded shared admission with deterministic regression controls and normal bypass prevention.
- [ ] Qualify shared disposable native startup, retain exact cleanup/failure evidence, record pre-close verification, then close/land through the gates.

## Log

2026-10-24: Filed unclaimed at TL8029db79's direction, for pair:5; do not wait for #863 to land. Both native failures and source pointer retained above. No production workaround in #863.


### 2026-10-24 — planning investigation and parked #722

Claimed #866 and started its own main-based branch through SDLC; #722 was checkpointed/pushed separately and remains SHIP/codecomplete awaiting its watched morning rerun. start-plan catch-up deferred stale merge-check.yml rather than changing protected config. No production code or gate configuration changed.

Inspected tagged Zellij source efd8fd5a via git show (checkout HEAD differs). listener accepts probe clients before metadata initialization; list-sessions ConnStatus does not certify readiness and removal reaches the None unwrap. Existing Pair create Admit already waits for exact agent readiness, but global ListingPoller and concurrent launcher observations are not excluded by that local wait. The exact historic probing process is still unattributed. First executed pane or actual server Render is a valid witness; no universal passive, current-generation readiness cache/CLI was found. Proposed shared namespace startup/probe exclusion and explicit retained uncertainty in the durable plan; needs TL approval before code.

TL8db7cb72 supplied the third capture at workshop/evidence/000863/branch-required-native-7a71f335.log on #863's remote branch; inspected bytes/hash217ecb09534658cc33ca98bdc08f71ff87141e1305d0ca28b8ba42af424f0f11 and the same pre-initialization1462 panic. No flake-rate claim.

A private direct-server diagnostic started under /Volumes/TestScratch/p866-kmgx1mqo, then its Python ps parser raised ValueError on an unrelated command's unmatched quotation, before the test query and cleanup. Reported gap to TL2d29a1aa (submitted); per TL78d1179c stopped direct socket/process probing. TL5ead059e (submitted) confirms exact private server21479 terminated and no other matches. No standalone reproduction result is claimed and no upstream issue filed. Further evidence comes from documented CLI/output and shared native harness only. Root cause research is source-supported, not proof that a specific old process sent the trigger.

## Revisions

### 2026-10-24 — proposed shared readiness exclusion

The existing local wait alone cannot protect a newly created session from global observers. The proposed plan adds a namespace-level creator/probe lease and bounded retained uncertainty, using exact existing pane readiness and explicit supported retirement. This is a scope/coordination proposal, not implementation authorization; alternatives and state/recovery table are in the durable plan. Await TL decision, then change-code before production edits.


### 2026-10-24 03:20 PDT — TL comparison request; marker-first revision pending

Submitted TL e242a4dc does not approve the initial shared-lease/unfinished-intent proposal. The durable plan now appends a governing marker-first comparison and withdraws the separate unfinished-intent store. Source-only enumeration confirms that global listing AND named helper actions call get_active_session/get_sessions and assert_socket across the whole namespace before honoring -s or ZELLIJ_SESSION_NAME. Public named attach/watch/kill/delete do likewise; only internal Rust primitives bypass enumeration, with no public local CLI switch found. Thus a marker-qualified target does not stop its CLI from probing an unready neighbor. A precheck also races endpoint publication without exclusion.

The revised proposal uses positive exact-launch ready markers, retains conservative starting/unknown projections for unmarked sessions, and keeps a lease only around the native calls that cannot avoid global scanning. It names the still-unresolved crash-before-delayed-endpoint-publication and stale/external-generation cases: inspect existing launch lifecycle authority before code, report a concrete gap rather than invent a replacement journal. No marker, lease, scanner or production change is implemented. Await TL review of the revised comparison; original initial-plan implementation steps are superseded.
