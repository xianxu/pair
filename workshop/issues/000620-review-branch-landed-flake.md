---
id: 000620
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '0111cd3e751fc816dab414ac7b617a97a2b46079' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:31:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ddfa5bf7", done: "e0800183"}
actual_hours: 0.41
---

# Review branch restore intermittently misses landed acknowledgment

## Problem

The recorded full suite for #604 failed in tests/review-branch-restore-test.sh: its review-pane handoff did not publish data/landed.json within the fixture's five-second wait, then read_text raised FileNotFoundError. The fixture printed an empty draft Neovim log; it did not retain the review pane's state at that boundary.

Failed head: fe4eb6d6e83bd53b9d42fcf27811a383c348b222. First observed here: 2026-10-16, during the #604 recorded suite. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log. The local-check record reports exit 2 in 5m38s and SHA256 prefix eee92d770af5. The failure was before go test ./..., so #604's test executables and their private clipboard environment had not run.

One unchanged isolated repeat passed: bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh; log /tmp/pair604-review-branch-restore-repeat.log. The current head differed only by #604's atlas/issue prose; fixture and executable source were unchanged. This is one failure and one passing repeat, not a calibrated rate.

## Spec

Capture the review-pane handoff/landing state at the missed publication boundary and determine why acknowledgment is missing. Distinguish scheduling delay, lost watcher event, rejected context and editor failure. The existing five-second wait followed by an unconditional file read is where the failure is observed, not a proven cause. Do not merely increase that timeout.

The #604 production diff only adds the explicit Couch switcher clipboard action and its clipboard-runtime seam. This fixture runs review readiness/editor handoff operations, without entering that action. Keep diagnosis/fix independent of #604.

## Done when

- A reproducer and bounded diagnostics identify the missed handoff acknowledgment.
- A regression exercises the identified ordering through the real fixture seam.
- Repeated checks report before/after failure counts and conditions without weakening the assertion or hiding failed attempts.

## Plan

- [x] Capture bounded review-pane admission/apply/publication evidence through the real fixture; preserve the initial five-second assertion during diagnosis.
- [x] Reproduce the demonstrated cause deterministically and apply the approved protocol-derived completion bound without weakening branch/context/CRLF/BOM/recovery checks.
- [x] Record before/after conditions and counts, targeted controls and the required full suite.

## Log



- 2026-10-17: unclaimed: Parked at the operator’s priority change to pair#648 (delivery 5f52a2ca-4b9c-4bd0-ae51-923cf3c9316c). PR407 remains open as draft: https://github.com/xianxu/pair/pull/407. Implementation and first-attempt branch suite passed13m11s (e6441fcd9bf5), close review SHIP with no findings; integrated CI including Couch/Zellij conformance passed on e7aa34cad97e7f2abc0dfa2c4535a3169375bcad (Git verified). The authoritative integration suite dd5c6cb38b31d5f9a96e7a10d5218c2a never started: queued behind two machine-suite holders, then my merge PID33746 was interrupted intentionally with SIGINT for the priority change. It exited with suite lock interrupted: context canceled and released its landing turn; this is not a failed test or a flake retry. Resume by claiming this pushed handoff branch, ready PR407 after checking its state, then sdlc merge --yes; let the merge gate integrate current main and run the still-required integration suite. Do not claim this item landed.
- 2026-10-17: closed — Demonstrated fixture aggregate deadline mismatch: five real serial resolver authorizations each allow 2500ms; delayed real successful calls reproducibly exceed the old 5s assertion. TL approved deriving actual acknowledgment completion from six possible resolver phases plus 1s polling/publication/observation, currently16s, using the real trace deadline. Delayed failures before repair2/2; delayed successes after repair3/3, including6.120s in full suite. Publication-refusal negative controls still fail with bounded pane evidence; context, CRLF/BOM bytes, branch restoration and recovery preserved. Fixture guards passed. Recorded full suite first attempt on892af00200451630bdbbf638e3df087d75f0dcc8 passed13m11s, log e6441fcd9bf5; survived Couch restart, no rerun. Atlas/lesson updated; no product or operator-environment change. Historical discarded pane state remains unprovable. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-16: Filed unassigned after one recorded failure and a passing unchanged targeted repeat. #604 retained its failed suite and started make -k to complete all remaining checks; no fixture or timeout change is included there.

- 2026-10-17: claimed after landing #598. Fresh branch prepared from main; first start-plan refreshed Ariadne and refused the older executing binary, then the second start-plan succeeded. One unchanged baseline via bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh passed (/tmp/pair620-baseline.log). Historical failed log confirms only an empty draft log and missing landed.json; it contains no review-pane state. Cause remains unknown.
- 2026-10-17 investigation plan posted to hub as proceeding unless objected: capture actual admission/authorization/apply/publication results and pending editor state in bounded diagnostics, then use controlled orderings to distinguish slow scheduling/IO from rejected context or failed editor effects. The watcher polls every 100 ms; there is no filesystem event to lose on this path. Preserve the five-second publication assertion. ARCH-PURPOSE: diagnose before changing budgets. ARCH-DRY: reuse existing real fixture and review observation seams. No operator environment action or production fix selected yet.
- 2026-10-17 measurements: instrumented real fixture passes at 0.506 s and 0.638 s acknowledgment latency. Reused tests/lib/review_observation_trace.lua, capturing actual resolver completions without adding observer subprocesses. Between publication and acknowledgment the baseline performs five serial synchronous resolver calls (~103–111 ms each), each with an existing 2500 ms wait. Retained /tmp/pair620-instrumented.log and /tmp/pair620-resolver-baseline.log. Three normal baseline/instrumented checks passed so far; no failures hidden.
- 2026-10-17 controlled failure: private /tmp/pair620-slow-probe.sh delegates to the real binary after 1.05 s delay only while the handoff phase is active. The unchanged five-second publication assertion fails (/tmp/pair620-slow-probe.log). Completed calls resolve the same review/a identity in ~1178–1191 ms (within the existing 2500 ms per-call bound); the pane remains normal/focused with no deferred round. landed.json appears during the bounded failure observation, with the handoff consumed. This demonstrates an aggregate fixture budget mismatch. It does not prove the discarded historical state was identical. Investigating an extra in-flight old-context resolver: RPC can reenter vim.system:wait, so the earlier 0.2 s sleep and subsequent snapshot do not establish completion of rejection. No new timeout or product change applied. Asked hub whether a measured protocol-derived fixture completion bound satisfies the Spec's prohibition on merely increasing the timeout; dependent fix awaits that decision.
- 2026-10-17 regression RED: new review-branch-restore-diagnostics-test.py drives the complete fixture through the existing explicit PAIR_TEST_REVIEW_INIT seam. Its private wrapper delays only readiness calls while a handoff exists, then execs the real Pair. Against the five-second bound, the delayed-success test fails while the controlled publication-refusal negative control passes (two methods, 24.822 s; /tmp/pair620-regressions-red.log). Together with the earlier controlled probe this is 2/2 delayed failures before repair; the three undelayed baseline checks were 3/3 passing. No failed attempt was discarded or retried as passing evidence.
- 2026-10-17 regression GREEN: the fixture now obtains the real per-resolver budget from its existing bounded observation trace and derives six allowed calls plus one second (currently 16 s), waiting on the actual landed artifact. Both new controls pass in 37.280 s (/tmp/pair620-regressions-green.log). The publication-refusal case still fails the real fixture and preserves uncertain outcome, pending handoff, missing landed artifact, resolved identity evidence and the controlled editor error. A second delayed-success run passes in 18.469 s overall; acknowledgment at 6.026 s, five resolver calls, real per-call budget 2500 ms and aggregate budget 16.0 s (/tmp/pair620-second-slow-green.log). Thus delayed success is 2/2 passing after repair, with full branch/CRLF/BOM/recovery checks retained; both observed refusal controls continue to reject missing publication. No product resolver/handler/state changes. The repeated ordering is fixture timing; no separate product reentrancy defect has been demonstrated.

- 2026-10-17 verification: targeted TestPairSubprocessesUseIsolatedEnvironment and TestNoPerTestBuildsOfFullPair pass in 2.001 s (/tmp/pair620-fixture-guards.log); git diff --check passes. Registered both real-fixture controls in make test-review; atlas/review-workbench.md maps the derived completion contract and lesson records the fixture-budget class.
- 2026-10-17 recorded full suite: sdlc local-check run --name suite on pushed 892af00200451630bdbbf638e3df087d75f0dcc8 passed first attempt, exit 0 in 13m11s, log SHA256 prefix e6441fcd9bf5. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/892af00200451630bdbbf638e3df087d75f0dcc8/suite.log. The suite survived the operator's 10:48 Couch restart; no interruption or rerun. Normal branch fixture acknowledgment 0.541 s; both diagnostic controls passed in 37.412 s, delayed success acknowledgment 6.120 s with five resolver calls and the 16.0 s derived bound. Remaining shell/editor checks and full go test ./... passed. Delayed success now 3/3 passing after repair; all three controlled publication-refusal checks rejected missing acknowledgment. No production or operator environment change in this patch; close review and landing follow as SDLC gates.

## Revisions

2026-10-17: TL approved waiting for actual handoff completion with a protocol-derived bound (delivery afeeff17-d436-405f-9110-16053b203595; canonical receipt submitted). Derivation: five serial resolver phases precede acknowledgment: controller admission, on_agent_round authorization, apply_round authorization, BufWritePre authorization and BufWritePost recovery authorization. One previous rejected-payload admission may already be in flight (the watcher running guard admits at most one); account for that call without changing product behavior. Obtain the per-resolver wait budget from the real observation trace (currently 2500 ms), multiply by six, then allow one second for timer polling, atomic artifact publication and the external 25 ms observation loop: currently 16 s total. The success condition remains publication of the actual landed artifact with the current context and exact CRLF/BOM bytes. Add a real-fixture delayed-resolver regression exceeding the old five seconds, and a controlled publication-failure negative control proving missing acknowledgment still fails with evidence. Preserve historical failure limits: their discarded pane state cannot prove the same cause. No separate product reentrancy defect has been demonstrated; any such finding will be filed separately per TL direction.
