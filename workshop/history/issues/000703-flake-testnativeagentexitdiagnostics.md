---
id: 000703
status: done
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '62da1908990b9c9f5585a0906c4c74ec1c0b6ee5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T06:30:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchtty.TestNativeAgentExitDiagnostics]
flow: {kind: quick, provenance: inferred, spec: "ce5713e2", done: "a9e3afb2"}
actual_hours: 0.33
---

# flake: TestNativeAgentExitDiagnostics

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchtty.TestNativeAgentExitDiagnostics: Same exact integrated head 48cf4264e56a67f6737db77997e2e3cec9f42468: recorded suite attempt a57d6744a65388be7e757875ca3aa520 failed TestNativeAgentExitDiagnostics/int at native_agent_exit_diagnostics_test.go:117, actor exceeded cleanup bound after 10s in the uninstrumented Python baseline run before Couch instrumentation. Direct go test ./cmd/internal/couchtty -run ^TestNativeAgentExitDiagnostics$ -count=1 -v on that unchanged HEAD then passed all seven subtests in 0.53s, package 0.827s (owned exec session 80249). Test introduced by existing main commit 9cb5fc7ed; no fixture or exit instrumentation changes in #683. Failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/a57d6744a65388be7e757875ca3aa520/run.log. Root cause of the signal timeout remains unproved; retain the native baseline boundary rather than widening timeout or weakening assertion.

## Spec

The diagnostic parity test must control where Python delivers SIGINT. Its current readiness banner precedes an uncontrolled asynchronous exception; a weakref callback can swallow KeyboardInterrupt and continue. A controlled real-Python probe proves that both baseline and the existing instrumentation can then exit0, while the instrumentation still writes the expected signal receipt. Parity alone therefore accepts a shared incorrect outcome.

Block SIGINT before instrumentation/readiness, use stdin EOF as the release after the parent sends SIGINT, assert the signal is pending, then unblock at an explicit top-level site. Add a deterministic callback self-signal case through the same fixture; its parent releases EOF but does not also signal, so scheduling cannot hide the adversarial ordering. Preserve native handler behavior and all existing cases. Independently validate expected termination, not only equality between specimens. Retain wait errors, process status, stderr and receipt in failures. No sleeps, retries, larger deadline or production changes.

The recorded timeout discarded stderr/status, so its exact delivery site is unproved. The fix defends the demonstrated ordering class; do not relabel the old timeout as proven weakref delivery. See the [implementation plan](../plans/000703-native-exit-signal-ordering-plan.md).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- The callback SIGINT ordering fails deterministically without the mask and passes with it, using termination/receipt evidence rather than a deadline.
- All original EOF, exception, SystemExit, HUP, TERM and SIGINT expectations remain checked, including an independent expected baseline outcome.
- An omitted SIGINT fails the pending-signal assertion after EOF; failure output retains the actual wait error, status and stderr.

## Plan

- [x] Implement the controlled SIGINT fixture and deterministic callback regression, with retained diagnostics, after hub approval and change-code.
- [ ] Verify focused/mutation and full couchtty coverage, recorded full suite, close review and landing.

## Log

### 2026-10-19
- 2026-10-19: closed — Test-only controlled SIGINT delivery: callback regression failed semantically in 0.05s before the mask; mask-only Go overlay failed in 0.12s with ignored KeyboardInterrupt and wrong exit, never deadline. Final focused eight exit cases, missing-signal controls, independent oracle and retained-error rendering passed 1.256s. Full couchtty passed 50.300s before rendering-only addition; artifactpath/testpair/termcmd passed 11.106s/10.470s/5.460s. Required recorded suite started on e2dec2854 and final-head pass precedes landing. no-atlas: no production or public surface changed; no-project: hub owns ticks; no-plan-check: only combined recorded-suite/review/landing workflow remains. Hub approved private Python proof with no :0 staging.; review verdict: SHIP

Claimed following verified hub dispatch3d5e75a8. start-plan created the canonical branch from fresh main; sibling catch-up completed. No implementation edits before the plan. #674 commit902e3b20a proved a startup KeyboardInterrupt swallowed by Python ignored-exception contexts; this is prior art, not proof of #703's missing original stderr. A read-only peer audit independently distinguished default delivery, ignored disposition and callback swallowing.

100 ordinary ready→SIGINT→EOF probes all exited by SIGINT; these passing samples do not diagnose the timeout. Deterministic scratch proof using the unchanged nativeAgentExitDiagnostics constant: unmasked callback self-SIGINT yielded exit0 for both baseline/instrumented, with ignored KeyboardInterrupt stderr and the expected instrumented signal receipt. Masking before readiness left SIGINT pending through the callback; top-level unblock yielded SIGINT termination for both specimens. No sleep or timeout oracle. Retained four-case evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair703-ordering.drZpu9/results.json. Actual original suite failure log is now /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/a57d6744a65388be7e757875ca3aa520.log (the creation text's inflight path was temporary).

Implementation: eight baseline/instrumented cases now share the controlled runner; SIGINT is masked before hooks/readiness, EOF releases the child, pending state is required, and top-level unblock delivers it. The callback case alone owns its synchronous self-signal; the parent only releases EOF. Independent expected-termination checks reject shared wrong parity across all original case kinds. Missing-signal baseline/instrumented controls exit1 at the pending assertion. Hooks and the production native fixture are unchanged; no timeout increase, retry or sleep.

RED before the mask: int_callback failed in0.05s with both statuses0, ignored KeyboardInterrupt stderr and a seemingly correct instrumented receipt. GREEN focused coverage passed1.169s. A Go overlay removing only mask establishment then failed in0.12s from the pending assertion/incorrect exit1, retaining ignored-callback stderr, never a deadline; tracked source was unchanged. Overlay/log evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair703-mutation.sZBjUX. That proof exposed waitErr formatted as an address; a pure diagnostic sentinel test failed before an explicit String renderer, then passed. Final focused cases, missing-signal controls, literal outcome oracle and diagnostic renderer passed1.256s. Full couchtty passed50.300s before that rendering-only addition; full artifactpath/testpair/termcmd passed11.106s/10.470s/5.460s. Required final-head recorded suite and SDLC close review remain outstanding.

## Revisions

2026-10-19 — Hub delivery519e7955 approved the test-only plan and runtime adversarial guard instead of a Go lint; default cleanup bound is unchanged and never the red oracle. Receipt and exact body verified submitted before change-code.

2026-10-19 — Replace the undiagnosed flake placeholder with a controlled delivery contract and adversarial proof. Original timeout mechanism remains uncertain because its evidence was discarded; the concrete defect established here is uncontrolled SIGINT delivery plus parity-only acceptance of a shared wrong result. Test-only scope; no native hook or production helper behavior changes.
