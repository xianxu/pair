---
id: 000698
status: done
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1aa8f6afa6b20814f32a5f5fd64d45a4297b1548' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T07:48:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer]
flow: {kind: quick, provenance: inferred, spec: "1823f796", done: "5b457804"}
actual_hours: 0.11
---

# flake: TestListedDetachedTerminalAttachesWithoutNewServer

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer: failed in attempt adbcdbbf22e41d689a246f916f9dd2d2, passed on the same head in attempt b8cfedebf5403d1c880f0aa2a5e89ccd

## Spec

The registration terminal fixture publishes the agent-ready record before the marker used by its list-sessions implementation. Production admission consumes agent-ready as permission to issue its one listing query, so a valid scheduling interleaving observes no sessions and refuses the cold handoff. Publish listing-ready before agent-ready; keep production admission, leases and refusal policy unchanged.

The permanent guard must observe the real fixture listing between atomic publications, not rely on repeated runs or an elapsed-time failure. Preserve detached attach coverage, process identity, readiness bytes and one-server/one-create assertions.

## Done when

- Listing publication precedes agent-ready publication, including the forced between-write observation.
- The regression fails immediately on the wrong listing or premature readiness when the original order is restored; no timeout is the red oracle.
- Publication errors do not announce agent readiness; the detached attach test and required suite pass, and close/landing unregister the flake.

## Plan

- [x] Add the controlled publication-boundary regression, prove red with the existing order, then publish listing state before final agent readiness (test-only).
- [ ] Pass targeted and required recorded suites, close on SHIP, land and verify removal from the active flake registry.

Durable plan: [publication order](../plans/000698-flake-testlisteddetachedterminalattacheswithoutnewserver-plan.md).

## Log

### 2026-10-19
- 2026-10-19: closed — Test-only fixture order: controlled old-order red 0.82s, corrected order green 0.44s, reverse-order overlay red 0.32s without timeouts. Publication errors preserve absence of agent-ready. Detached attach and registration/environment tests pass (package 1.976s); full artifactpath, testpair and termcmd guards pass. Required recorded suite started on pushed 50b2d2b03dc9bcb432725bbccd51cd35dd04de7c; landing remains gated on its pass. No new production or public surface, hence no atlas change; hub owns project updates; only the combined suite/review/landing task is unchecked. Hub approved private test-only conformance, no live staging owed.; review verdict: SHIP

- Claimed and started on fresh main (8f84dd12619483a9be757c5ba6b2ae49d66e1491). Retained failure log: `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/adbcdbbf22e41d689a246f916f9dd2d2.log`, lines 828–829: cold handoff refused in 0.45s, terminal not listed live. Same-head pass remains recorded in attempt b8cfedebf5403d1c880f0aa2a5e89ccd.
- Source trace: pair-wrap publishes PAIR_AGENT_READY_PATH, then PAIR_TEST_ZELLIJ_READY; the latter alone controls fixture list-sessions. admitTerminalAfterReady permits its listing query after the former. Holding that gap until the query observes it reproduces the refusal without any timeout.
- Investigation overlay (tracked source unchanged): `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair698-ordering.fkxvjqsj/overlay.json`. The listing query writes a receipt only after observing agent-ready while listing-ready is absent, which releases the wrapper. Existing target failed in 0.37s with the original not-listed-live error; receipt states `listing observed agent-ready before listing-ready`. Unmodified targeted control passed in 0.77s. This establishes the bad interleaving rather than estimating its frequency.
- ARCH-ORDER: final readiness must follow every prerequisite its consumer immediately queries. ARCH-DRY: use the existing fixture serializer and atomic filesystem seam. Proposed executable guard is domain-specific publication semantics, not an AST-detectable Go misuse; request hub confirmation instead of adding a brittle fixture-name lint to the upstream catalog.

- Hub approved plan (receipt d292f29f-9f91-45fd-8a81-7a32d833047d, submitted), including executable semantic enforcement instead of lint and private test-only conformance. change-code admitted the quick flow.
- Added one test-only ordered publisher and a real-filesystem observer that invokes the actual fixture's short and full listing commands after the first atomic write. Original order RED: 0.82s, premature agent readiness plus explicit no-active-sessions replies; second-write error also exposed readiness announced before successful prerequisite publication. Listing-first order GREEN: 0.44s. No timeout or sleep decides these cases.
- Reverse-order overlay control: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair698-reverse.7hdfu5bf/overlay.json`; RED in 0.32s with the same semantic assertions. This is an intentional mutation, not a flaky test failure.
- Final targeted suite: publication, empty/listing-form replies, nested environment isolation and unchanged detached reuse all passed (couchcore package 1.976s; original detached test 0.74s). Full artifactpath/testpair/termcmd packages passed in 12.840s/13.726s/5.118s; diff check clean. Production files and all terminal reuse assertions remain unchanged.
