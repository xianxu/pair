---
id: 000638
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours: 2.452
collaboration-mode: collaboration
card_mirror: 'a5dbbb568f9ef3a71cecb43bab4d68a59ba13327' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T02:01:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 4.00
---

# Couch Ctrl+Y copy still fails live after #604; the error vanishes and isn't logged

## Problem

## Spec

pair#604 (PR #381) landed Ctrl+Y "copy last notice" in the Couch switcher using the local system clipboard. The operator's live check on 2026-10-16 at about 23:15, on Couch built at 712a6478 and running since 22:22, failed:
- pressing Ctrl+Y showed an error;
- nothing reached the clipboard;
- by the time they went back to screenshot it, the error was gone.

Nothing about it is in supervision.jsonl, so the error text is lost too.

Two defects:
1. The copy fails in the operator's real environment, even though the tests pass with a fake clipboard. Find which tool Couch picks (pbcopy?), and how it fails from the Couch process: PATH, sandboxing or the launch environment, a timeout, stdin handling.
2. A failure has to stay visible and be recorded: the notice persists until it is dismissed, and the error, with the tool and its exit status and stderr, goes to the supervision log.

The original OSC 52 loss layer is still undiagnosed (#604's close review).

## Done when

- Typed native tool/exit/bounded-stderr evidence survives through the production Console path; one content-free clipboard_copy event logs tool/exit/timeout and stderr length/hash only (never free-form stderr).
- Failed or unconfirmed CopyFeedback persists until explicit dismissal or successful retry; stateful fake tests prove retry, payload retention and menu transitions.
- An opt-in real Console clipboard conformance harness is available; the normal suite never touches the real clipboard.
- Recorded full suite and close review pass. No pair:0 staging. The historical copy failure is not yet fixed: after landing, the next natural Couch restart carries the operator's Ctrl+Y check (real paste, or measured logged cause filed as a follow-up), owned by the hub.

## Plan

- [x] Deliver typed clipboard diagnostics and a pure retained-copy feedback model, with red/green tests.
- [x] Wire bounded copy outcomes to supervision.jsonl; exercise production Console wiring with stateful clipboard doubles and retained-error sequence tests.
- [x] ~~Obtain the actual Couch-path failure evidence through live conformance, correct the measured cause, and verify a real Ctrl+Y paste with the operator.~~ Superseded by the approved acceptance Revisions: the opt-in harness is delivered; the hub holds actual Ctrl+Y acceptance for the next restart. No historical cause is claimed fixed.
- [x] Run the complete recorded suite and update README/atlas; close through SDLC is the final gate for this checklist.

Durable design: workshop/plans/000638-couch-clipboard-copy-plan.md.

## Log

### 2026-10-16

### 2026-10-17 — initial investigation and authorized real-tool probe
- 2026-10-17: closed — Corrected unified completion: unread inherited stdin with 1 MiB fails in both clipboard entry points; exit 0 after full input with held stderr succeeds and flags diagnostic truncation. Red/green and focused race pass. Recorded full suite on 0e892021865cda5cf2d2b3dbb6e9ec99ab74ee9f PASS exit 0 in 13m38s, log sha256 26e8ff759e12; all CI green. No state, timeout or OSC policy added. Existing typed evidence, persistent CopyFeedback and metadata-only supervision remain tested. TL-approved rollout defers historical real Ctrl+Y cause/paste to next natural restart; not claimed fixed. No pair:0 staging or real clipboard probe. (fixed-then-ship); review verdict: SHIP
- 2026-10-17: closed — Recorded full suite on 35219ec723e70a7c033bf0801bd7b9482d206c55 PASS exit 0 in 13m14s, log sha256 prefix 8b5e337d5e9a; CI green; behavioral red/green, focused race and artifact classification pass. Typed tool evidence, persistent retry/dismiss state and metadata-only supervision wiring proven through Console keys with private executables; opt-in real conformance harness delivered but not run. TL-approved acceptance Revisions explicitly defer actual Ctrl+Y paste or measured-cause follow-up to next natural restart; historical failure unknown, not claimed fixed. No pair:0 staging or additional real clipboard probe.; review verdict: SHIP

#624 landed as PR390 before this claim. start-plan refreshed the slot's ariadne copy from 585ae4f7 to 0a1f6a45 and required a rerun; the rerun created this issue branch from main. Brew reported existing go/cue/uv; no new machine packages were chosen.

Source proves three evidence defects: copyLastNotice discards the ClipboardCopyLocal error and shadows it with Presenter.Copy's error; copyClipboard discards stderr and returns no tool identity; copy feedback bypasses the supervision writer. The menu copy message has no expiry timer: ordinary operator events clear it, and inventory/bookkeeping can replace its ownerless notice. The historical event that removed the operator's error remains unknown. Presenter.Copy only emits OSC52, with no clipboard acknowledgment; a stalled-parent drop may return nil. Neither fallback success nor a new fake-backed pass proves actual clipboard delivery.

TL approved exactly one real pbcopy/pbpaste probe in receipt e82fa054-d8cb-4cf5-8b21-970e0c91884e: original bytes only in memory, no clipboard contents printed/logged, restore only while the clipboard still contains the marker. From this Couch-launched worker's non-interactive environment, /usr/bin/pbcopy and /usr/bin/pbpaste resolve and SSH markers are unset. The exit-only probe completed exit 0 in 0.077s, verified its marker and restored the previous bytes. This confirms that one worker-launched path works; it does not reproduce or explain the operator's original Couch failure. No further clipboard write, primary build, visible window or slot drive was performed.

A bounded read-only lsof check found no outer terminal capture for PID99807. A fresh-context read-only agent independently traced the lost-error and notice-clearing paths; no implementation edits were delegated. Plan approval and real Couch-path evidence are required before inventing a timeout/PATH/bootstrap correction.

### 2026-10-17 — implementation entry and native evidence

Plan-quality rounds 2/3 CLEAN disposed PQ-1/PQ-2; estimate-quality INFO accepted the derivation, noting harness and verification allocation. The issue-spec primitive includes incurred investigation; the Go/TUI implementation items cover harness construction, and the review allowance includes recorded-suite triage and landing integration. change-code completed normally with full flow and no bypass.

TestClipboardCopyLocalRetainsToolEvidence first failed with only `exit status 7` despite the fake pbcopy writing a diagnostic. The typed executor now preserves tool, process status, bounded sanitized stderr and wrapped errors; a separate metadata value exposes only byte count/full-stream hash, never free-form stderr. Whole clipcmd package passed unchanged legacy tests plus metadata/cancellation/start-failure/oversized-stream controls. One test's expected stderr length was corrected from a hand-count to literal byte length. The 1s Console policy is unchanged; no real clipboard probe was repeated.

### 2026-10-17 — retained state, Console integration and conformance harness

The second behavioral red reproduced native-error loss: Console showed only the unconfirmed OSC outcome. A fresh-context subagent's pure reducer reds reproduced lost feedback, Escape clearing ordinary menu state, copy results erasing ordinary notices and missing banners. ReduceMenu now owns independent CopyFeedback; held failures retain one immutable original payload, explicit retry replaces the outcome, and Escape consumes only dismissal. Both banners fit at 40x10 with extents/cursor re-based.

Production Console-key tests prove original-payload retry after a new feed item, one content-free observation per attempt, both failure causes, cancellation ordered by a fake child's start handshake, and no late banner after shutdown. The observer is wired to the existing bounded supervision log; executable doubles prove real stdin/exit/stderr collection without touching the host clipboard. The log seam first failed red with no clipboard row. An initial screen assertion incorrectly assumed terminal diffs contained contiguous words; corrected it to inspect a stable rendered token while metadata assertions verify exact byte count/hash. Combined focused clipcmd/couchtty/couchcmd/artifactpath tests passed; the whole artifactpath package passed in 10.984s.

The Darwin/Couch-context real conformance test is explicit opt-in and remains unrun under the approved next-restart rollout. It backs up only bounded clipboard bytes in memory and conditionally restores only if its marker remains current. The original actual-path diagnosis/paste Plan row is superseded by the acceptance Revisions; the historical failure remains unknown, not fixed.

### 2026-10-17 — pre-close frame-budget regression and superseded queued run

CI on b9a5b907 passed all checks, including zellij conformance. Final source inspection found that only root/start frames reserved copy-banner rows; action and coding-agent menus still windowed at full height and then lost selected rows during final clipping. A new behavioral red at 40x10 reproduced hidden selected resume-candidates and grok entries. The fix moves banner reservation to the single RenderMenuView owner before every frame windows its items, replacing local deductions.

The first recorded suite attempt on b9a5b907 was still waiting for a machine slot (inflight 17f8af91f6156202288c7f67aff01c3c, queued about 15 minutes). It was canceled intentionally as superseded, before any suite test ran, via SIGINT to this session's verified sdlc PID23102. Its owned cat/lock helper exited; no peer process was touched. The recorded cancellation is not a test flake. A new recorded suite will run the corrected code head.

The central-budget correction passed the expanded menu/Console selection in the main session (0.661s), following the agent's behavioral red and green. Current pushed code head is 35219ec723e70a7c033bf0801bd7b9482d206c55 in PR394. Its recorded suite is running/queued through the normal machine gate, inflight fbbb0a170bb6e5481c03573880bfc45f, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/fbbb0a170bb6e5481c03573880bfc45f/run.log (not yet a passing result). Complete that run, handle failures under the worker rule, then close and land; no primary staging and no real clipboard rerun.

### 2026-10-17 — recorded verification before close

Corrected-head recorded suite PASS: 35219ec723e70a7c033bf0801bd7b9482d206c55, exit 0 in 13m14s after 1668s of machine-queue waiting, log SHA256 prefix 8b5e337d5e9a. Full retained log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/35219ec723e70a7c033bf0801bd7b9482d206c55/suite.log. All shell/editor checks and the complete Go run finished; no flake rerun or early stop. CI conformance, merge-check and local-gates guard passed on this code head. Focused race checks passed for clipboard execution, feedback and supervision wiring; complete artifactpath classification passed. The prior b9a5b907 queued attempt was intentionally superseded before execution, not a test failure.

Both rollout/privacy decisions were additionally verified against canonical submitted Couch receipts 00b6f3ff-12f2-4f3e-8407-9da4c629a989 and 8429745d-ecb6-40e5-9325-a6282e0141c0 (ops:0 to pair:7). Close acceptance is the tested instrumentation, persistence and available opt-in harness. The harness was not run against the host clipboard; no further real probe or primary staging was performed. The historical live copy failure remains unknown; the hub owns next-restart real Ctrl+Y paste or measured-cause follow-up.

### 2026-10-17 — post-close inherited-stderr regression

A private executable probe found a branch regression before publication: the old Cmd boundary (stderr to the null device) returned nil for a zero-exit tool with a detached child holding stderr, while the new bounded stderr capture returned exec.ErrWaitDelay after its unchanged 100ms drain cutoff. The tool had completed successfully; only diagnostic EOF was unavailable. This matters for daemonizing clipboard tools and legacy ClipboardCopy callers.

TestClipboardInheritedStderrKeepsSuccessfulExit failed red for both legacy/local entry points, then passed after preserving zero-exit success only for exec.ErrWaitDelay and marking stderr capture truncated. Existing finish still enforces context cancellation/deadlines, and nonzero exits remain errors. The fixture holds stderr behind an explicit release handshake, with a hard child lifetime; no real clipboard is touched. A preliminary 20ms-deadline fixture could kill the tool before it spawned a holder, so its cleanup wrongly waited for a nonexistent child; removed that invalid fixture branch and retained the existing independent deadline/cancellation controls. Full clipcmd package PASS after the fix (1.741s).

The first close had returned SHIP with no findings. Main was then merged cleanly; the PR-head mismatch was repaired through the gate's named sdlc pr/retry path. CI passed on integrated 8ee7973c5ee00a2b24c109598fdda075e71d6f21. Its required suite (inflight b66166661a7ca42e485f737a77708b0a) was still queued when this regression was proven. SIGINT to this session's verified merge PID24391 canceled that superseded run and released the landing before any test executed; no peer process was touched. This is not a test flake. Hub report b5db9799-03b2-4f94-9019-baa1f88730d8 is canonically submitted.

Fresh corrected-head suite and fix-only re-close are required before landing. The 1s/OSC policy, persistent feedback, metadata-only log boundary and deferred live acceptance are unchanged.

### 2026-10-17 — unified completion red/green

TestClipboardInheritedUnreadStdinFails failed red in both legacy/local modes: an exit-0 tool with a child holding 1 MiB stdin unread was falsely reported copied. Retaining the existing strings.Reader and requiring Len()==0 before demoting ErrWaitDelay fixes both, while TestClipboardInheritedStderrKeepsSuccessfulExit retains successful full-read behavior and the truncation flag. Full clipcmd package PASS (3.064s); focused race tests for both descriptor cases plus cancellation/deadline controls PASS (2.560s). No real clipboard touched. The prior queued 8c suite was canceled before execution because this negative control exposed the branch defect; no passing retry is claimed.

### 2026-10-17 — corrected-head full suite

Recorded suite on 0e892021865cda5cf2d2b3dbb6e9ec99ab74ee9f passed exit 0 in 13m38s after an 827s machine-gate wait; log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/0e892021865cda5cf2d2b3dbb6e9ec99ab74ee9f/suite.log, sha256 prefix 26e8ff759e12. All shell/editor and Go packages completed, no flakes. clipcmd 1.261s, couchcmd 79.817s, couchtty 56.296s, couchcore 452.130s. All PR CI checks green, including conformance (2m27s). This is the fresh full-suite evidence for the unified completion fix; the historical live clipboard failure remains unmeasured and the operator check stays with the next natural restart.

## Revisions

### 2026-10-17 — TL-approved instrumentation-first acceptance

Receipts dd9c7097-9be3-4fca-9cc3-c7046cde424a and 00b6f3ff-12f2-4f3e-8407-9da4c629a989 approve the full-flow single-pass plan and explicitly change acceptance sequencing. Close and land on tested instrumentation, persistent CopyFeedback and the opt-in conformance harness. Do not stage on pair:0. The historical clipboard failure remains unknown and is not claimed fixed. The hub holds the actual Ctrl+Y result for the next natural restart: either a real paste, or the logged measured cause filed as a follow-up. This supersedes the pre-landing live-check steps above without guessing a mechanism or changing the 1s/OSC policy.

### 2026-10-17 — strict log boundary (PQ-1/PQ-2)

TL receipt 8429745d-ecb6-40e5-9325-a6282e0141c0 replaces proposed echo detection with a strict projection: supervision never carries free-form stderr. Record only tool, exit status, timeout flag, total stderr byte length and a short hash; the operator's screen may show bounded sanitized stderr. No overlap heuristics. The plan now names tested functions and adversarial strategies, including deterministic cancellation ordering, rather than enumerating case prose.

### 2026-10-17 — implementation correction within the approved executor contract

The inherited-stderr correction preserves the existing successful-process contract rather than adding a new recovery state: successful exit plus diagnostic-drain cutoff is copied with truncated capture metadata. Context expiry still refuses confirmation; failed process exits still fail. No retry, timeout increase, stderr logging or operator-environment action is added.

### 2026-10-17 — unified clipboard completion rule (TL approved)

A negative control proved that exit 0 plus ErrWaitDelay can also mean an inherited stdin was never consumed. TL receipt 62f5141b-44e8-40dc-bcf3-9505d0fa95f2 approves one rule: retain the existing strings.Reader and require Len()==0 after Run, successful process exit, and a valid context before treating ErrWaitDelay as diagnostic-only stderr truncation. No new state or timeout. Add the unread-stdin negative control alongside the full-read/held-stderr positive control, rerun the suite and fix-only close review before landing.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; estimate-source flags calibration stale, so provisional. The plan cleared plan-quality in round 2 before deriving this estimate. One issue/spec primitive budgets the investigation and acceptance discussion (0.5 design, 0.3 impl); two smaller Go extensions budget executor and log wiring (each 0.3 design x0.2 thorough-spec discount, 0.5 impl); the TUI reducer/rendering primitive takes 1.0 design x0.2 and 1.0 impl. Real-API discovery uses the 0.6 impl upper bound for the opt-in harness; docs 0.1 design x0.2 and 0.2 impl; one review 0.2 design x0.2 and 0.5 impl. All implementation values below apply v3.1's 0.4 multiplier. Existing os/exec, rowtext, reducer and supervision writer cover the stack; no greenfield library or cross-repo work. Familiarity 1.0; thorough-plan design buffer 15%. Design 0.88 x1.15 + implementation 1.44 = 2.452 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 2.452
```
