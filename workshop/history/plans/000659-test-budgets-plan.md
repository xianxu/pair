---
collaboration-mode: discussion
---

# Load-independent test budgets Implementation Plan

> **For agentic workers:** Follow AGENTS.md Section 3. Delegate bounded exploration; keep each fixture correction with its failing control. One atomic close, no milestone tags.

**Goal:** The seven groups in #659 assert behavior rather than tool speed, report host context, and keep a loud required native-conformance path.

**Architecture:** Keep product deadlines unchanged. Use existing stateful fakes for semantics, testwait for patience, and optional/required verdicts at native conformance boundaries. No retries or increased constants.

**Tech Stack:** Go tests/context, SlotCatalogFake/FakeRunner, private subprocess fixtures, channel handshakes, testwait.

## Evidence and priority

Read the issue and all named tests; two independent read-only subagents traced their budget owners. Measured host load during planning: `{ 60.65 65.52 86.42 }`. That is context, not a reproduction count. Preserve the issue's historical failed log before retention rolls it out.

The seven groups contain nine top-level functions. Park growth is in couchcore, not couchcmd, and already uses fake zellij; native skips are wrong there. Continuation's missing origin is explicitly nonfatal production behavior, not proven causality.

#576 has priority: on the hub's #651-landed announcement, finish the atomic edit, commit #659's state and return to #576 immediately. Do not wait for this suite/landing. Cancel owned recorded checks with `sdlc local-check cancel` if necessary. No implementation before hub plan approval.

## Core concepts

**Pure entities:** No pure production entity is added or changed. The existing
`RateLimit.Allow` decision in `cmd/internal/broadcast/point.go` remains unchanged;
the HTTP test now supplies its logical time through the integration boundary.

| Integration | Files | Seam |
|---|---|---|
| Optional/required probe verdict | `cmd/internal/diagnosticlog/proof_test.go` | Test integration helper reads environment policy, reports through testing.T and samples host diagnostics. Its policy checks launch subprocesses; it is not PURE. |
| Failure-only host diagnostics | `cmd/internal/testwait/host.go`, platform helpers/tests | Best-effort bounded load/CPU/runtime metadata; explicit unavailable reason. No authority. |
| HTTP rate clock | `broadcast/server.go`, `session.go`, `pointer_test.go` | Inject logical limiter time, reusing Config.Now; default time.Now. Real socket read deadlines stay real. |
| Recovery fixture | `couchcore/slotrecovery_test.go`, `recovery_fresh_missing_test.go` | Existing SlotCatalogFake before initial Discover; separate slow-tool refusal control. |
| Park test query | `couchcore/park_lifecycle_live_test.go`, `park_client_readiness_test.go` | Query parameter in test-only readiness helper; native default wrapper unchanged. |
| Continuation child | `couchcmd/continuation_acceptance_test.go` | Private real writer; testwait-derived context. |
| Clipboard child/log | `couchcmd/supervision_copy_test.go` | Private stalled fake tool; typed product timeout/fallback evidence. |
| Activity invariant | `couchtty/workbench_activity_invariant_test.go` | Existing probeTimeout seam, same counted real runtime. |
| Native lsof | `diagnosticlog/proof_test.go` | Optional/required classification, unchanged product budget. |

## Classifications and corrections

| Group | Actual budget owner | Change / retained assertion |
|---|---|---|
| Pointer rate limit | Logical refill; test assumes HTTP posts fit a real window | Freeze clock, assert exact accepted/limited count, advance time and assert refill through HTTP. Keep rate constants. |
| Recovery Fresh ordinary/foreign-owner | Product5s Git probe leaks into semantic fixture via OSSlotCatalog | Build relevant fixture with SlotCatalogFake **before initial Discover**, not after bounded real discovery. Preserve filesystem, ownership, no-signals/no-launch assertions. Separate slow/unanswered catalog test checks refusal and unchanged state; real ProvisionIO slow-tool contract remains. |
| Park helper growth | Test1s plus nested product5s fake-CLI query | Test-only injected query gives gated baseline→unchanged→growth. Use testwait patience. Existing fake CLI tests retain parser/argv/no-reply coverage. |
| Continuation exact checkpoint | Test-only20s subprocess context | Use testwait.Budget; retain publication/digest/ownership/content assertions. Don't invent origin to hide expected warning. |
| Clipboard key→facts | Product1s may expire before fake exit7; test incorrectly requires exit7 | Deliberately stall fake pbcopy and assert typed timeout, executable/tool identity, unconfirmed OSC fallback and content privacy. Exact exit7/stderr remains independently tested in clipcmd; pure projection test remains. Test polling/cleanup use testwait. No new clipboard production seam. |
| Idle UI tick counts | Product2s per-row plus test30s pass wait | Configure existing probeTimeout with testwait.Budget here; use testwait.After. Preserve zero-byte/cold/mutated controls and dedicated default/deadline tests. |
| Native lsof open-file tests | Product2s inspection plus child5s readiness | Keep cap; readiness uses testwait. Optional unavailable/incomplete skips with named reason. `PAIR_REQUIRE_LSOF_CONFORMANCE=1` fails instead. Complete wrong holders/identities never skip. Fake proof tests always run. |

## Prove each failure before changing it

Use temporary private controls, retain output and exact head, remove temporary edits. No production sleeps/hooks. Record entry/result/release instead of guessing interleavings.

| Group | Old failing control | Corrected check |
|---|---|---|
| Pointer | Inter-post delay permits real refill, defeating rapid-post count | Same delayed transport with frozen logical time still exact; permanent normal test has no sleep. |
| Recovery | ProvisionIO wrapper delays exact worktree-list until request budget expires | Typed timeout/refusal, no effects; unrelated success fixture no longer depends on bounded real query. |
| Park | Hold reply beyond old caller1s | Gated in-process replies wait until release; unchanged/unknown never proves attachment. |
| Continuation | Owned invocation shim holds beyond old20s | Old killed; testwait parent waits then validates exact publication. Cleanup reaps child. |
| Clipboard | Owned fake tool stalls beyond product1s | Old exit7 assertion fails; new test expects timeout/fallback, not longer deadline. |
| Activity | Context-aware runtime delay exceeds old2s | Old loses row; new counted test waits for release; separate product deadline tests still reject overdue probes. |
| Lsof | Private delayed wrapper exceeds product budget | Optional named skip; required hard failure; wrong-answer controls hard-fail in either mode. |

Host diagnostics sample once on affected failure/skip, using existing platform load primitive if available, otherwise a bounded best-effort command/read. Print unavailable reason on failure. No per-row process scan; omit process count unless already cheap. Test formatting and unavailable/canceled sampling with fakes. No clipboard, notice contents or credentials in diagnostics.

## State projection

Actions: investigate/rerun exact test; run required conformance; fix asserted behavior. No product-visible state changes.

| Observation | Visible verdict | One next action | Possible loss |
|---|---|---|---|
| Expected event/decision | PASS | Continue | None |
| Event absent at test deadline | FAIL with awaited event + host state | Investigate/rerun exact test | Partial temporary output retained in log |
| Optional tool unavailable/incomplete | SKIP with tool/reason + host state | Run required conformance | No native claim for this run |
| Required tool unavailable/incomplete | FAIL with reason + host state | Restore availability and rerun | No passing native evidence |
| Complete wrong result/effects | FAIL with expected/actual + host state | Fix asserted behavior | External unknown effects stay unknown |

Cancellation never becomes success; unknown evidence never authorizes mutation/deletion. All fake children/workers are joined before fixture cleanup.

## Architecture and envelope

- **ARCH-DRY:** reuse fakes, clock, testwait and existing deadline seams; one diagnostic helper.
- **ARCH-PURE:** existing rate decisions remain pure; clock delivery and optional/required verdict reporting belong to IO/test integration, including environment reads, subprocess tests and host diagnostics.
- **ARCH-PURPOSE:** all seven groups/nine functions; no skipping semantic correctness.
- **ARCH-MOCK:** stateful catalog/runner/fake CLI, gated stall evidence; required lsof mode checks native contract.
- **ARCH-CONSTRAINTS:** no speed claims in normal tests. Test patience bounded by go-test deadline. Failure sampling bounded, off success paths. Sequential repetitions, no unbounded CPU burner.
- **ARCH-SECURE:** private roots/scripts only; no live Couch/clipboard/staging/settings. lsof observes test-owned file; facts remain content-free.
- **ARCH-ORDER:** explicit start/result/release; timeout, unknown, late reply and cancellation distinguished. No new production lifecycle states.
- **ARCH-FUNERAL:** cleanup reaps children/joins workers/removes roots. Preserve bounded evidence, not unbounded run data; no residual load process.

No new production hot path: rate checks O(1), unchanged UI tick read/decode0 at20×30MB retained. Host diagnostics once per affected failure, never per byte/row.

## Tasks (one close)

- [x] Preserve historical log in task-owned temp directory; record hash/head/load. Collect old failing controls before their corresponding edits.
- [x] Write red tests for clock/default/verdict/diagnostic seams, implement minimum changes and verify unchanged product budgets. Commit each coherent group with Agent trailers.
- [x] Run `go test ./cmd/internal/testwait ./cmd/internal/broadcast -run 'Test.*(Host|PointerRateLimit|RateLimit)' -count=1`; exact counts and diagnostic controls pass.
- [x] Run `go test ./cmd/internal/couchcore -run 'Test(RecoveryFreshOrdinarySlotKeepsOrdinaryLaunch|RecoveryFreshMissingRecordLeavesForeignOwnersUntouched|ParkHelperAttachmentRequiresClientGrowth|ParkClientCountQueryContract)' -count=1`; all ownership/growth checks pass.
- [x] Run `go test ./cmd/internal/couchcmd ./cmd/internal/clipcmd -run 'Test(ContinuationWriterPublishesExactCheckpointAcrossWorktrees|ClipboardCopyConsoleKeysReachContentFreeFactLog|.*Clipboard.*)' -count=1`; exact checkpoint and typed metadata pass.
- [x] Run `go test ./cmd/internal/couchtty ./cmd/internal/diagnosticlog -run 'Test(WorkbenchPerformanceIdleUITick|ConsoleActivity.*|OpenFileInspectionConformance|UnregisteredChildOpenFileBlocksLegacyExpiry)' -count=1`; count invariants pass, native pass/named optional skip. Run required lsof mode separately and retain verdict.
- [x] Run all nine named functions20 times, subtests included, and record actual concurrent fleet work/load. Per the hub's accepted Revision, injected-delay controls are primary proof and the measured-load cohort supports it; neither load100 nor another campaign is required.
- [x] Push head and pass the recorded suite `sdlc local-check run --name suite`; record its actual load separately from the repetition cohort. Complete all checks, retain failures and follow the flake rule.
- [ ] Update issue/atlas for any new seam, close through mandatory review, land on SHIP with `sdlc merge --yes`, report merge/PR. #576 preemption overrides continuation at any point.

## Revisions

### 2026-10-17 — close review BR-1/BR-2: classification and accepted evidence

The initial table called the optional/required verdict a pure entity. Its implemented helper reads `os.Getenv`, emits testing verdicts and may sample host diagnostics; its policy tests execute subprocesses. Classify that whole helper and its checks as test integration. No pure extraction or production change is needed; the existing pure rate decision is reused unchanged (ARCH-PURE).

The original task rows proposed a deliberately generated load cohort and a suite under the same load. Hub delivery 7b1e37d3-9f5e-4f37-9565-fd234eea0376 accepted the actual observed-load repetitions as support for the primary injected-delay controls and explicitly said not to chase100. Synchronize active task wording with that approved decision; keep repetition and suite loads separately recorded. The issue Revisions preserves the original acceptance wording.
