---
id: 000878
status: done
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'b9f7237ed3b5211a1304470b63401ff0eb1906cf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T12:54:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c4d7f89a", done: "cb65462b"}
actual_hours: N/A
---

# Agent fixture quota exhaustion misses completion and cleanup join

## Problem

The full default local Go set on clean main e2f58424b124c38b4340acee459f21fa009394f2 failed TestRunStopAfterOrdinaryRequestQuotaExhausted in cmd/internal/testcouch/agentfixture. It reported run_test.go:324 fixture observation deadline, then run_test.go:311 fixture failed to join during test cleanup; shortroot.go:28 could not remove owned actor/results because the directory was not empty. Test duration13.45s, package117.146s. This was a once-only main baseline comparison for #866, not a candidate-only finding.

Exact command from the detached scratch Pair root: bash tests/lib/with-pair-test-binary.sh go test ./... -count=1 -timeout20m. GOFLAGS=-p=3 -trimpath; TMPDIR and GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot5. Main source was clean before/after and its exact Git head unchanged.

Complete log /Volumes/TestScratch/pair866-main-go-buzlps33/go.log SHA2569cfce92fe2f6bc9ca95f7e03b478728ddbdff02d0027734b7e08e349f6ec59b9; metadata.json/result.json beside it record exact head/command/exit1. Scratch source checkout was removed after completion; captures remain. The other main failure TestCLILiveLeaderCancellationJoinsBeforeCompletion is already #869 and outside this issue.

## Spec

Investigate ordinary-request quota exhaustion in the retained agent fixture, including its stop transition, final observation, owned goroutine/process joins and filesystem cleanup. Preserve the existing request quota and semantic completion assertions. Do not hide a live fixture behind cleanup suppression or an arbitrary timeout increase.

The actor owns its mailbox drain, stdin reader and CLI workers until Run returns. Cancellation must be observed between ordinary requests, so a canceled drain does not continue repainting or publishing the remaining snapshot. An already executing synchronous write may finish; no promise cancels a blocked arbitrary io.Writer. The existing deferred input close/reader join/worker join remains the sole cleanup owner.

The quota scenario retains all128 completed ordinary requests, rejects overflow and admits the separate generation-fenced Stop request. Observe each semantic request completion with the existing bound rather than placing one four-second deadline over128 filesystem transactions. Gate a final observation in a separate controlled test, prove no premature completion, then release it and verify Stop and joined cleanup. Cover Claude and Codex with the same fixture owner.

## Done when

- Exhausting the ordinary request quota reaches the intended stop/completion state and joins all owned work before cleanup returns.
- The existing test passes with deterministic coverage of delayed final observation and cleanup ordering.
- Failure diagnostics identify any unjoined actor/results writer, and the owned short root can be removed without a concurrent writer.

## Plan

- [x] Add a channel-controlled stdout writer in run_test.go and a full-mailbox cancellation regression for both agents: hold an ordinary repaint, publish the remaining requests, advance into the next snapshot, cancel while its repaint is held, release it, and assert Run joins without servicing another ordinary request. Release gates and join the actor on every failure path. Run the new case against unchanged run.go to obtain deterministic RED; if it does not reproduce the cancellation defect, stop and revise before implementation.
- [x] In run.go, check the existing context between ordinary mailbox entries before beginning another request; preserve the current cancellation result, reserved Stop admission, quota, result identity and deferred joins. Do not add a timer, worker, queue or retry. Verify the controlled regression GREEN and assert the owned actor root can be removed after completion with no writer left behind.
- [x] In TestRunStopAfterOrdinaryRequestQuotaExhausted, retain each result and await its exact correlated completion before publishing the next ordinary request, using the existing observation bound per transition. Preserve128 retained results, overflow refusal, successful reserved shutdown and final result-count checks. Add controlled delayed-final-observation coverage without sleeps or a larger timeout, plus useful pending/result identity diagnostics on failure.
- [x] Run targeted quota, cancellation, stale shutdown, retired-result and request-lifetime tests, then complete agentfixture and agentfixture/cmd packages with race detection. Run repolint and test-affected-inventory; audit and individually update affected readers only if the inventory requires it. Record RED/GREEN and clean-root evidence before the recorded pre-close check and the one binary-owned close review.

## Log

TL233cf35e directs an unclaimed issue, after searching active main issue details for the exact test/contour. No production/test change or flake registration was made under #866 for this independent main failure.

### 2026-10-24 — dispatched investigation and proposed quick-flow plan
- 2026-10-24: closed — Window503ba31b9fadffa4e965105f805391eaccdc0af8..2d96b3b792fc4b7fdd9ee6e6767cfc7f6f4f1d9d. TL86fe165c authorizes measurement exclusion under ariadne#593: exact refusal measurement branch proof: measurement snapshot does not serialize: json: error calling MarshalJSON for type *gatestate.MeasurementInputs: measurement exceeds 65536 bytes. No typed hours or evidence reconstruction. Controlled full-mailbox cancellation regression RED on unchanged Run for Claude and Codex: completed128/writes129 after cancellation, expected2/3; GREEN with only between-request ctx checkpoint. Quota controls observe each exact completion within the original bound for both agents, retain128 results, reject overflow, accept generation-fenced reserved Stop, and hold final repaint to reject premature completion. Both canceled and successful actors join before owned actor/results roots are removed. Combined targeted PASS5.835s. Complete fixture and fixture/cmd race packages PASS20.316s and3.787s; tracked Go catalog PASS7.882s; fresh affected-reader inventory and whitespace PASS. The changed reader was individually audited as private fixtures. PR595 source2d96b3b792fc4b7fdd9ee6e6767cfc7f6f4f1d9d pushed. Recorded pre-close lease-1348768564 attempt e91bcb864ea13447d74f8f9eff102ed2 remains retained in queue; no PASS claim and landing requires actual result. No new architecture, user surface, format, timeout, quota or process owner; atlas waiver is for this fixture bugfix only.; review verdict: SHIP

Authenticated TL001af097 assigns878 from main and requires a plan before code. Claimed and started on branch000878-agent-fixture-quota-cleanup at main503ba31b9fadffa4e965105f805391eaccdc0af8;866 remains pushed/codecomplete for the separate combined operator stage and722 stays parked. start-plan refreshed the clean resting checkout and its private substrate through the binary-owned catch-up. No protected configuration or source edit was made.

Unchanged targeted command go test ./cmd/internal/testcouch/agentfixture -run '^TestRunStopAfterOrdinaryRequestQuotaExhausted$' -count=1 -v -timeout=45s passed test0.20s/package0.369s. This passing sample does not explain or erase the retained main failure. Source inspection finds Run checks ctx only in the outer select; its ordinary snapshot loop can perform128 synchronous Paint/state/result transactions without another cancellation checkpoint. The original bulk result observation applies a fixed four-second deadline to that aggregate; its cleanup cancels then waits four seconds while the same drain may continue writing. A controlled writer test will distinguish the cancellation defect from an unsupported timing hypothesis before the minimal fix.

Core concepts: no new pure entity or persisted format. Existing AgentState/Command/Result and MaxRequests128 remain unchanged. Integration owner Run in run.go gains a cancellation checkpoint; test-only writer gating in run_test.go controls the existing stdout seam and joins every owned goroutine. ARCH-DRY reuses Run's existing context and deferred join; ARCH-ORDER checks cancellation at request admission; ARCH-CONSTRAINTS retains per-transition bounds and cardinality rather than extending timeouts. No durable artifact family is added; test channels/goroutines end at joined test cleanup. Work stays inside the quick-flow design/code limits; one close boundary, no Mx tags.

Alternatives: raising observation/cleanup timeouts would leave cancellation-insensitive writes and only move the load threshold; reject. Dispatching every state write to another worker adds ordering and join machinery; reject. Recommend one context checkpoint plus deterministic controlled-write coverage and per-request semantic observations. Await TL approval before change-code or any source/test edits.

### 2026-10-24 — approved controlled RED/GREEN and qualification

Authenticated TL24af0d46 approves the exact plan. change-code admitted the quick flow before test/source edits. The unchanged quota target passed as noted above; the new TestRunCancellationStopsMailboxSnapshot failed deterministically against unchanged Run on both agents: completed128/writes129 versus expected2/3 (Claude0.40s, Codex0.34s). The writer held the second ordinary repaint in a snapshot containing the remaining126 requests; cancellation then release proved later requests were admitted despite the canceled context. All test-owned gates were released and actors joined even in RED, with no intentionally leaked writer.

The sole production change is the context check at ordinary-request admission in the existing snapshot loop (five lines including comment). Existing cancellation return value, stdin/worker joins, quota, reserved Stop and serialized state/result publication are unchanged. Regression GREEN for both agents, test0.28s/package0.457s. No policy timer, new worker, retry or persisted format was added.

Quota coverage now runs Claude and Codex, retains every exact result and waits per request within the original bound. A gated final repaint proves only127 completions exist until release, then the128th completes; overflow still refuses and reserved shutdown succeeds. Both the quota and canceled-snapshot tests remove the owned actor/results tree only after Run returns. Combined targeted pass5.835s; complete fixture race packages PASS20.316s and cmd PASS3.787s; tracked Go catalog PASS7.882s. Source whitespace passes. Audited every run_test.go read: private fixtureConfig/ShortRoot actor, transcript and marker paths or t.TempDir; no source/config discovery added. Updated only that reader's local_or_literal_fixture record; fresh make test-affected-inventory PASS. The semantic cancellation boundary is defended by the stateful channel gate; a generic lint ban on loops would not describe this contract.

Implementation/control deliverables above are complete. Next workflow postconditions: push candidate, recorded pre-close on that pushed head, one binary-owned close review, then normal landing. No operator stage is needed for this test-fixture-only behavior.866/722 stay separately pending and unchanged.
