---
collaboration-mode: discussion
---

# Native exit fixture SIGINT ordering plan

> **For agentic workers:** Follow AGENTS.md Section3. Execute this small fixture change in the current session; SDLC owns the close review.

**Goal:** Make the exit-diagnostics test prove SIGINT termination under a controlled ordering, with a deterministic regression and useful failure evidence.

**Architecture:** Keep the existing Python instrumentation unchanged. A shared test-only fixture blocks SIGINT through setup, readiness and the release gate, verifies a pending signal, then delivers it at top level. Compare baseline/instrumented outcomes only after checking the independent expected outcome.

**Tech Stack:** Go os/exec, installed Python signal/weakref, owned pipes and temporary logs. Quick-flow-sized test-only work; no milestones or new production API.

## Evidence and limits

- Original attempt a57d6744a65388be7e757875ca3aa520 on48cf4264 failed in the plain baseline, line117; the assertion discarded stderr, wait error and native status. Its exact cause cannot be reconstructed.
- #674's retained startup evidence established the ignored-KeyboardInterrupt mechanism elsewhere. A generic analogy alone is insufficient.
- Our deterministic probe uses this test's existing nativeAgentExitDiagnostics constant: synchronous SIGINT inside a real weakref callback is swallowed, and both specimens exit0. The instrumented specimen still records EXIT=signal:SIGINT, so the old parity/receipt oracle accepts both being wrong.
- With SIGINT blocked, the callback returns with the signal pending; explicit top-level unblock terminates both specimens by SIGINT. All four real-Python outcomes are retained in the issue Log's scratch evidence. No sleeps or elapsed-time failure.
- 100 ordinary samples all passed; they do not disprove the ordering hole or establish the old timeout's stack.

## Core concepts

### Pure values and rules

| Name | Lives in | Status |
| --- | --- | --- |
| Exit fixture case / expected termination | cmd/internal/couchtty/native_agent_exit_diagnostics_test.go | modified |
| Fixture outcome (wait status, error, stderr, receipt) | cmd/internal/couchtty/native_agent_exit_diagnostics_test.go | new |

One case describes the requested exit, signal source and receipt. One outcome retains evidence from one owned child. Expected termination is independent of baseline/instrumented parity: EOF0, exceptions1, SystemExit7, native HUP/TERM, and SIGINT-derived termination (SIGINT or exit130 where Python reports that form). This is a small private test predicate, with literal value checks if extracted.

### Integration

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Shared exit fixture runner / Python script builder | cmd/internal/couchtty/native_agent_exit_diagnostics_test.go | modified | real Python, signal delivery, stdin/stdout/stderr, owned process cleanup |

This is direct native conformance, not a production dependency or a new service needing a fake. The controlled Python callback is a stateful adversarial specimen through the same runner. The existing instrumentation and nativeAgentFixture consumers remain unchanged (ARCH-DRY). No generic fixture framework.

## Ordering contract

Actions are Send signal, Release EOF, Inspect outcome, and cleanup the directly owned child.

| Phase / event | Effect and next observation | Failure / recovery |
| --- | --- | --- |
| SIGINT specimen setup | Import signal; block SIGINT before hooks and ready | Unexpected inherited ignored disposition is reported, not silently treated as valid baseline. |
| Ordinary SIGINT ready | Go sends SIGINT, then closes stdin | Child passes EOF gate, requires pending SIGINT, then unblocks at top level. |
| Adversarial SIGINT ready | Go closes stdin only | Child's weakref callback raises SIGINT synchronously; it must remain pending until top-level unblock. No competing parent signal. |
| Missing-signal negative control | Release EOF without sending or self-signaling | Pending assertion fails immediately with diagnostic evidence; no timeout verdict. |
| EOF / exception / SystemExit / HUP / TERM | Preserve current case behavior | Independently assert expected baseline result, then parity and bounded receipt. |
| Child ends or assertion fails | Wait/reap the same owned handle and close pipes | Report wait error/status/stderr/receipt. Existing bounded cleanup remains a safety guard; it is never the regression oracle. |

The mask is owned by the disposable child until it exits, not shared Go/global signal state. No retries or later execution. No production persistence, worker, queue, visibility or lifecycle policy changes. Each run owns one child; no descendants or operator terminal/window. The process and t.TempDir logs end with the test.

## Implementation and verification

Only modify native_agent_exit_diagnostics_test.go, this issue/plan, and lessons.md for the learned rule. No production hook edits.

- [ ] Extract the current local runner only as needed so existing cases and the callback specimen use the same script/transport/cleanup. Keep helpers private in the same test file.
- [ ] Add the independent expected-termination oracle for all seven existing case kinds. Add an int_callback specimen that self-signals inside a weakref callback after EOF; its parent must not send a competing signal.
- [ ] Run the adversarial case before the mask fix. It must fail from wrong termination/pending evidence with ignored KeyboardInterrupt stderr, before any cleanup deadline.
- [ ] Block SIGINT before hook installation/ready; after EOF and optional callback delivery, assert SIGINT is pending, then unblock at top level. Keep default handlers and the unchanged instrumentation's previous-handler chaining.
- [ ] Retain cmd.Wait's error, native ProcessState/status, captured stderr and receipt in every helper failure. Check inherited disposition explicitly so ignored input cannot masquerade as successful delivery.
- [ ] Add a missing-signal negative control proving immediate pending-assertion failure. Use the same real runner; do not inject a fake signal function or a clock.
- [ ] Run go test ./cmd/internal/couchtty -run '^TestNativeAgentExitDiagnostics' -count=1 -v. All baseline expectations and bounded receipts must pass.
- [ ] Temporarily remove only mask establishment via a Go overlay, rerun int_callback, and require deterministic semantic failure; restore and rerun the target. No timeout increase or sleep.
- [ ] Run full go test ./cmd/internal/couchtty -count=1 plus full artifactpath/testpair/termcmd guards. Record evidence, push and run the required recorded suite; close with SDLC's fresh review and land on SHIP. No atlas surface changed; any precise no-atlas acknowledgement explains test-only scope. Hub owns project ticks.

## Alternatives and boundaries

- More retries, sleeps or longer cleanup bounds hide the ordering and are rejected.
- Replacing the default handler with immediate os._exit would change the behavior under test and is rejected.
- Keeping parity alone accepts both specimens swallowing SIGINT and is rejected by the independent oracle.
- The shared Go lint catalog lives upstream in ariadne/pkg/golint. This defect depends on Python ignored-exception delivery context, rather than a mechanically invalid Go call. Proposed guard is the executable callback regression; ask the hub to confirm that classification or assign any useful upstream catalog work separately, rather than inventing a brittle string-matching lint rule here.

## Architectural check

ARCH-DRY: one existing runner and instrumentation constant. ARCH-PURE: expected outcomes are values; process/signal effects stay in the test runner. ARCH-PURPOSE: control delivery and reject shared wrong parity, preserve every original case. ARCH-MOCK: actual Python conformance with a controlled callback specimen; no new production integration. ARCH-CONSTRAINTS: fixed eight cases/two children per case, no runtime hot path or growing scan; existing cleanup bound unchanged. ARCH-SECURE: own child handles and private temporary logs only. ARCH-ORDER: EOF and kernel pending state establish order, not timing. ARCH-FUNERAL: reap every child, close every pipe, t.TempDir removes logs. ARCH-EVOLVE: no persisted product format changes.

## Approval

Pending hub review. No implementation changes before approval and change-code.

## Revisions

2026-10-19 — Approval supersedes the initial pending status above. Verified hub delivery519e7955 approved the complete test-only plan, including the controlled gate, callback self-signal, independent outcomes, missing-signal control and retained diagnostics. Hub explicitly agreed this is Python delivery-context semantics: executable adversarial guard, not a Go lint. Proceed through change-code; no scope change.
