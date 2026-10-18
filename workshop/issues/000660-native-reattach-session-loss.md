---
id: 000660
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3c67d04e4d56e9d7d8342fdb57d6120d2c4e7b9b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T06:05:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "9ae9c647", done: "9a5fe6cc"}
actual_hours: 0.86
---

# Native direct Zellij session disappears after detach

## Problem

MacOS conformance intermittently loses the private direct-Zellij session at persistent reattach. First retained failure: PR422, run39105463917 attempt1, head7de3dec6684644b5114278ac92c53106d45345f2, 2026-10-18 06:57UTC. TestNativeConsoleWrapperZellij/direct-zellij-baseline waited285.86s at terminal_native_test.go:312 for the persistent frame after reattach. child.raw contained only "No session with the name ... found!"; receipts had BOOT/CPR/input and STATE=<same boot>:1 before detach. The wrapped subtest subsequently retained PID:nonce and advanced counter1 to2, before the package's five-minute alarm exhausted the remaining budget. Treat that alarm as a cascade from the first wait, not a separate causal finding.

Run: https://github.com/xianxu/pair/actions/runs/39105463917/job/117610220213

Same issue code passed conformance in runs39105228904 (d9c9bab5bb60d3cf68b753bbe079606e8139ac0f) and39105295728 (eeedca6731051353d4314ab056da6ba2b200d0f9); only documentation differs through the failed head. The terminal fixture is unchanged by #658. This establishes intermittency, not its cause.

This differs from #603's wrapped real-shell-actor discovery wait, and #361's initial layout/fixture mismatch. Here initial launch/input/state succeeded and reattach reported an absent session.

## Spec

Land the approved diagnostic stage so the next native reattach discovery failure explains its boundary facts. Preserve private-session ownership, original PID:nonce/counter/frame assertions, and native deadlines. Record detach result, original client exit, owned actor liveness, socket metadata, and bounded wire/server-log evidence before cleanup; report an exited replacement client promptly. Keep the underlying discovery cause unresolved and park this issue after diagnostic landing. Do not build the upstream-race reproduction or choose a production accommodation without a failure pointing at it.

## Done when

- Diagnostic output distinguishes detach/client exit, actor liveness, socket state and server history without adding session probes; byte tails and agent exit receipts are bounded and owned scratch is removed after cleanup.
- Deterministic regressions exercise early dead-client reporting, live/unknown preservation, bounded latest evidence and missing-file errors; instrumented actor termination preserves the original raw wait status. Original native identity/state/frame assertions remain.
- One granted native direct/wrapped family attempt passes with counts recorded; the recorded full suite and normal close/merge gates pass for this diagnostic stage. The Log records the unresolved cause and next-failure handoff.

## Plan

- [x] Classify the original failed job from its complete retained log and inspect the exact upstream 0.45.1 source before selecting a fix.
- [x] Add bounded, fixture-owned detach/reattach observations and deterministic tests of diagnostic collection and early child-exit reporting; preserve all same-process/frame assertions.
- [x] Run the one granted private direct/wrapped conformance attempt, record both results and release the lane.
- [x] Record the corrected head's full suite and native evidence for normal close review; leave the unresolved-cause handoff ready for parking after diagnostic landing.

## Log

### 2026-10-18
- 2026-10-18: closed — Hub-approved diagnostic-only landing, then reopen/unclaim unresolved cause. BR-1 complete consumer sweep: actual five-stream collector retains bounded latest child/parent/detach/wrapper/receipt evidence; regression failed all four old prefix streams before fix, now targeted tests pass normally (1.208s) and -race (3.569s). Corrected pushed e2f8bca989ecd6fa6b1c22f3cd82b3e9f0333018 full recorded make PAIR_GO_TRIMPATH=-trimpath test passed exit 0 in 11m46s, log b60cf9203f95; all server CI green including native macOS conformance. Original single granted local native family passed two subtests with same PID:nonce and counter 1 to 2, private scratch removed, lane released; no additional local native run. Atlas and lessons updated. No production fix, deadline bump, attach retry or reproduction claimed. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Filed unclaimed while verifying #658. Failed CI log and job metadata retained locally under pair658-profile.yqKPI1/ci-39105463917*; durable evidence is the public run and summary above. Two earlier greens on unchanged issue code establish the flake. #658 may make one recorded unchanged failed-job rerun under the fleet rule; any repeat is reported rather than retried indefinitely.

### 2026-10-18 — claimed investigation and corrected classification

Hub dispatch 5c39bbe0-8a21-4953-9949-255be98a93e4 verified submitted. Card was unowned; claimed pair:3 and ran start-plan from clean main. The first start-plan refreshed the ariadne tool layer and refused to branch with the old binary; the requested second invocation created this issue branch. No native lane used.

Fetched the complete original job log with `gh api repos/xianxu/pair/actions/jobs/117610220213/logs --allow-escape-sequences` into invocation-owned TMPDIR `pair660-evidence.iHo4cL/job-attempt1.log`. The older retained failed-step excerpt omitted the successful "Dump Zellij logs on failure" step. That full log materially changes the interpretation: the original server starts at 06:52:51.199, the client says Bye at 06:52:52.188, and the server emits its periodic PTY warnings at 06:53:51.312, 06:54:51.311, 06:55:51.314, and 06:56:51.312. Its plugin shutdown is 06:57:36.826, during direct-subtest cleanup after the failed wait. There is only one server start before that cleanup. Thus the attach error does not establish original-server death. The socket and agent were not independently sampled, so their exact state remains unknown. Attempt 2 of the same head passed; its retained metadata already exists. No additional retries have been run.

Read-only audit of the Pair fixture: detach command success is checked, then oldChild.Exited (after drain/reap) and endpoint disposal are awaited. Console disposal kills only the exited client's PID, not its process group. Explicit private-session kill is only in cleanup. Missing observations are the old client's wait code and exit-wire tail, the socket's state, and the original actor's exit reason. The fixture uses on_force_close=quit, but normal client Bye plus continued server activity does not support blaming that policy yet.

Exact local upstream tag `v0.45.1` (tag ref bebf50a75198fe6c434cbf34cb2ef981a1d959ca) supplies a concrete hypothesis: src/commands.rs performs session_exists and then attach_with_session_name, each probing the session socket. zellij-utils/src/sessions.rs:148-170 treats a missing/unexpected ConnStatus reply as absence. zellij-server/src/lib.rs reuses the lowest free ClientId, removes ConnStatus clients immediately, and route.rs queues another RemoveClient when its router exits. A late cleanup for probe/client N could therefore remove a replacement connection with the reused numeric ID. This resembles #640/#647's upstream client-ID family; it is source-supported, not a reproduction or proof of this incident. Do not weaken identity or retry attach based on it.

Proposed first stage is diagnostic, not a speculative behavioral fix (ARCH-PURPOSE, ARCH-ORDER): observe the owned boundary and retain unknown outcomes. Reuse Child.Wait/Exited and existing testpair environment/evidence helpers (ARCH-DRY); use deterministic fake child/process outcomes for the diagnostic policy (ARCH-PURE, ARCH-MOCK). Limit each diagnostic stream to a bounded tail, and all new logs to fixture scratch removed only after owned children join (ARCH-CONSTRAINTS, ARCH-FUNERAL). No user sessions or global process sweep, credentials, new runtime configuration, or production lifecycle changes (ARCH-SECURE). The native grant request will name exactly one sequential direct/wrapped family attempt, its five-minute test cap, and cleanup of only its private session. No native use before that grant.

Hub approved diagnostic stage in submitted receipt c94e69cd-f88a-4a57-8fdb-da82ecf4f129 and linked the upstream family: zellij issues 5270/5382 and merged, unreleased fix PR 5630. No reproduction is claimed. Instrument four finite phase boundaries, without adding ConnStatus/list probes that could disturb the suspected race: socket-file metadata only, owned actor liveness, old client Wait code, bounded old/new wire tails, and bounded Zellij log tail. Agent EOF/error/signal receipts preserve original termination behavior. A dead replacement child fails the frame wait promptly with its retained exit evidence instead of consuming the package deadline. Unit tests force already-exited and live-but-not-ready outcomes plus oversized tails; native tests retain the exact PID:nonce/counter/frame checks. Existing five-minute family cap remains unchanged. Native grant will be requested after targeted tests pass.

Diagnostic TDD: fake Child exit=1 plus missing-session wire first failed with nil evidence; bounded log-tail regression first returned zero bytes and false success for a missing file. Both now pass. The helper reads at most 8192 latest file bytes, reports read errors, and uses actual Child.Done/Wait and its existing stateful fake. A separate latest-wire bound test retains the final exit reason. Terminal fixture now logs four file/handle observations, old client exit/wire, and the capped private Zellij log before failure cleanup. Explicit private TMPDIR puts Zellij's existing log inside the fixture cleanup owner. There are no extra native session queries. Replacement-child exit ends only the impossible frame wait; the original PID:nonce/counter/frame assertions remain.

Bounded agent instrumentation was delegated as isolated test-only work. Empty instrumentation first failed its missing-receipt assertions; an adversarial exception type with a newline first failed the one-line bound. Seven controlled Python cases now preserve baseline raw wait status for EOF, exception, bounded hostile exception type, SystemExit(7), HUP, TERM, INT. Each has readiness gating, unconditional owned cleanup/reap, and a ten-second hang bound. A receipt is at most one printable line, at most 102 bytes. SIGKILL remains unobservable; EXIT=atexit does not claim status zero. Targeted combined diagnostics/evidence tests passed. No Zellij session, native lane, or operator UI has been used; installed version query reports 0.45.1.

### 2026-10-18 — native sample, release and approved diagnostic landing

Grant 7344ec27-1d46-465f-97a2-f8c87023fa3e authorized exactly one native family invocation. Ran the fresh candidate from diagnostic commit 9cb5fc7ed6a3425f2e0a384f7462a4ec7726a077 with `go test -race -json ./cmd/internal/couchtty -run '^TestNativeConsoleWrapperZellij$' -count=1 -timeout=5m`. It exited 0: direct 2.44s, wrapped 24.61s, family 27.05s/package 28.42s; two subtests passed, zero failures. Both detach commands succeeded and old clients exited 0. Same boot identities advanced counter 1 to 2 in both routes. Private pcn-1188022532 and pcn-3438811432 directories were absent after the joined test. The 54,921-byte owned JSON is pair660-evidence.iHo4cL/native-family-attempt1.jsonl, SHA256 d0306cde33c4637f2f87613005423d7bc3c05ba633e0c2807fef6cf0fc02687d. Lane released with result receipt 8c1afbf8-8be3-4a5c-8ef2-1b518ff207f7. This green sample is not a causal reproduction.

Early draft PR429 opened under the standing CI-overlap authorization. Submitted hub directive 75b0dbdf-a6a2-4d97-b7d4-013501c3c4fc now explicitly authorizes landing these diagnostics through the normal gates, then parking #660. Do not build a controlled upstream reproduction before a failure points at it. Next CI failure: read the retained detach result, old/new client exits and wire, socket metadata, actor receipts/liveness, and private server-log tail before choosing a fix. Underlying causal acceptance remains pending. Normal close/merge finalizes the diagnostic scope; immediately reopen with this rationale and release the claim to keep the unresolved issue parked, rather than claim the flake fixed.

### 2026-10-18 — recorded suite passed, close review next

Recorded pre-close suite on pushed 892d4d74271c1d7281c9f3c68f8a343d3443ef70 passed exit 0 in 13m59s, log SHA256 prefix b0f5f3215d84, attempt 3c238d9a21bbcf478fa82ee4c080d288. Queue admission waited 256s; all Make prerequisites and final Go packages completed. Saved log: `.git/sdlc/local-checks/892d4d74271c1d7281c9f3c68f8a343d3443ef70/suite.log` in the common Git directory. No suite failure or retry. PR429 server checks on that head are green: macOS conformance 3m13s (run39131116219/job117686307893), merge-check 1m20s and local-gates-guard 11s. Code remains unchanged since the tested diagnostic stage; subsequent changes are this evidence and close preparation. Normal close review has not run yet. The final Plan checkbox records verification preparation, not a premature review/landing claim.

### 2026-10-18 — close BR-1, bounded latest diagnostic consumers

First close review returned FIX-THEN-SHIP with one Important finding, BR-1 / bounded-latest-diagnostic-evidence, so the close was not finalized. Verified the finding: the cleanup reporter still used the old 4096-byte prefix for child.raw, parent.raw, wrapper.log and receipts.txt, which could omit appended EXIT receipts. Detach output was also mislabeled as a tail and printed unbounded again on failure. No dispute or bypass.

Swept that entire family through one failure collector: child wire, parent wire, detach output, wrapper log and actor receipts now retain at most 8192 latest bytes, preserving file read failures. The actual cleanup reporter calls this collector. Both detach success and failure reporting use the same byte-tail rule; the old full-body duplicate is removed. Old/dead-client and private-server diagnostics reuse the same tail/file helpers. Original boot identity is logged separately before detach so it survives a long receipt stream. The sole remaining prefix helper in terminal_native_test.go formats the first differing serialized viewport cells for a frame-comparison failure; it is not an event-history consumer.

New regression through the failure collector first failed for all four old prefix streams (each lost its distinct final marker), then passed after the sweep, including fifth-stream detach output and missing-receipt errors. Targeted combined tests pass in 1.208s; the corresponding race run passes in 3.569s. The original native sample is not rerun: this correction changes diagnostic collection/reporting only, and the one granted native lane use is already spent. A fresh required full suite and updated macOS CI will cover the corrected code before close round 2. The first full-suite pass remains retained; it does not stand in for this corrected head.

### 2026-10-18 — corrected full suite passed, close round 2 next

Required suite on corrected e2f8bca989ecd6fa6b1c22f3cd82b3e9f0333018 passed exit 0 in 11m46s, log SHA256 prefix b60cf9203f95, attempt a04ca8ecf85a7a50a0c7e12d2d109056. It acquired its slot after 1062s and completed all Make prerequisites and Go packages, including couchtty 46.290s. Retained log is `.git/sdlc/local-checks/e2f8bca989ecd6fa6b1c22f3cd82b3e9f0333018/suite.log` in the common Git directory. This fresh verification was required by the BR-1 code correction, not a flake retry; both full-suite attempts passed. Corrected-head server CI is green, including macOS conformance 2m35s (run39132943893/job117691644844), merge-check 1m0s and local-gates-guard 9s. No additional local native invocation. Ready for normal close round 2 to settle BR-1; causal investigation remains parked after diagnostic landing.

### 2026-10-18 — diagnostic-stage SHIP

Normal close round 2 returned SHIP, high confidence, no Critical/Important/Minor findings remaining. BR-1 was explicitly disposed addressed after the reviewer ran the actual collector regression with a temporary prefix mutation; all five streams failed under that mutation. The reviewer also reran targeted tests normally (1.190s) and with race detection (3.565s). Close published codecomplete bound to b8fe3d0451f156fca0275100835a52eb4f74076d and adopted measured actual 0.86h, excluding 21.99 recorded queue minutes across the two passing suites. Proceeding directly to normal landing. Afterwards reopen/release the underlying discovery cause as directed; next CI failure, read the retained facts. No causal fix is claimed by this SHIP.

### 2026-10-18 — reopened for parked causal investigation

Reopened per hub directive 75b0dbdf-a6a2-4d97-b7d4-013501c3c4fc: only the diagnostic stage landed as PR429, merge b386574636297f0529f0e4c0d7cbc29b1ff58e4d (GitHub and Git agree). Authoritative integrated-head suite on 5762aacb23916e687332bafc92fbd0078b40c732 passed exit 0 in 11m9s, log SHA256 prefix ddc803f0a234, with all five CI checks green. Integration suite queued 913s; saved log is in the common Git directory under `.git/sdlc/local-checks/5762aacb23916e687332bafc92fbd0078b40c732/suite.log`. No suite failed or was retried blindly.

The discovery flake's cause is still unresolved. Park and release ownership now: next CI failure, read the retained detach/client/actor/socket/server facts before choosing a fix. The original server was alive in the historical failure; the known upstream client-ID reuse family remains a hypothesis for this incident. Do not build an upstream controlled reproduction or a production accommodation until a failure points at it. The local native lane was released after the single granted attempt; no live resources or running checks remain owned by this issue.

## Revisions

### 2026-10-18 — BR-1 consumer sweep

Retain the same diagnostic-only scope and unresolved-cause handoff. Add the complete five-stream failure-collector regression and replace every event-history prefix consumer, including detach error reporting, with bounded latest evidence. Original frame-comparison prefix diagnostics remain intentional. This is the review-required completion of the approved observations, not a production fix or native-reproduction expansion.

### 2026-10-18 — approved diagnostic landing, underlying cause parked

Hub 75b0dbdf-a6a2-4d97-b7d4-013501c3c4fc narrowed this landing to observations and early dead-client reporting, with the bug parked for the next CI failure. Spec/Done-when/Plan now describe that deliverable. The original causal criteria (identify why discovery fails, reproduce its ordering before a behavioral fix, and retain native same-process assertions) remain unresolved investigation obligations, not claimed achievements. No production accommodation, extra native run, timeout bump or retry is authorized by the diagnostic landing.

### 2026-10-18 — investigate failed discovery before assuming session death

The original Problem/Spec/Done-when wording calls the session gone. Complete retained server logs show it stayed active until fixture cleanup. Preserve those original acceptance obligations, interpreting "why gone" as why attach's discovery returned absence; confirm actual socket/server/agent state before changing behavior. Await hub approval of this diagnostic-first plan before code.
