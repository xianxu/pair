# Boundary Review — pair#562 (whole-issue close)

| field | value |
|-------|-------|
| issue | 562 — Zellij CLI queries can finish before their reply |
| repo | pair |
| issue file | workshop/issues/000562-zellij-empty-reply-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 965fcb8147cbb4f63a43d867a571cff7afa140e0..77145de9091256e8390296f2fe10cf127bc2679c |
| command | sdlc close --issue 562 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T01:52:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The empty-reply boundary and atomic fixture publication are well covered, including counterfactual failures. Shipping is blocked by a remaining name-probe path that converts failed observations into acceptance, plus acceptance criteria that still promise a protocol regression beyond the delivered tests.

Done-when: stale — the first criterion requires deterministic query and ConnStatus protocol regressions; the revision delivers a Pair boundary fake and proposed upstream scenarios.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: observation-errors-must-preserve-uncertainty
    title: |
      Failed name probes with diagnostics still teach positive acceptance
    detail: |
      cmd/internal/launcher/osruntime.go:145-151 returns nil when cmd.Run fails with nonempty output other than the recognized length refusal. A temporary regression using stderr "failed to connect to server" and exit 2 fails because ProbeSessionName accepts the name; sessionNameAcceptor can then generalize that false evidence to shorter names. Propagate unrecognized failures, including cancellation with partial output, and accept only established validator evidence. Add subprocess-level regressions covering these outcomes and their monotone-cache consumers. This leaves the revised uncertainty sweep incomplete (ARCH-PURPOSE, ARCH-ORDER, ARCH-SECURE).
  - id: new
    severity: Important
    kind: defect
    family: acceptance-criteria-must-match-delivery
    title: |
      Done-when promises protocol regressions absent from the delivered scope
    detail: |
      workshop/issues/000562-zellij-empty-reply-race.md:52 requires deterministic protocol evidence for both early query completion and ConnStatus exclusion, while line 59 marks reproduction complete. The committed helper at cmd/internal/launcher/zellij_empty_reply_test.go:116-124 implements its own Log/UnblockInput behavior and contains no ConnStatus/Connected model; the upstream draft lists receiver tests as requested scenarios. Reconcile Done-when and the completed Plan checkbox with the revised boundary-testing scope and explicit upstream dependency, or deliver the promised protocol regressions (ARCH-PURPOSE).
```

1. **Strengths**

   - The shared `ErrZellijNoReply` guard preserves the explicit observed-empty inventory exception.
   - Sequence tests verify no automatic retry and recovery through a separately requested observation.
   - The live PID/pane fixture uses the same publisher exercised by the deterministic FIFO-barrier test.
   - The atlas documents uncertainty and the limitation of nonempty incomplete inventories.

2. **Critical findings**

   - [osruntime.go:145](/Users/xianxu/Library/Caches/sdlc/review-checkouts/8503ec25161e4365/cmd/internal/launcher/osruntime.go:145): preserve failed probe outcomes instead of returning acceptance. Retain any necessary recognized no-session validator response explicitly.

3. **Important findings**

   - [Issue:52](/Users/xianxu/Library/Caches/sdlc/review-checkouts/8503ec25161e4365/workshop/issues/000562-zellij-empty-reply-race.md:52): align acceptance criteria and completed checkboxes with the actual evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused tests passed across launcher, couchcore, and pairlifecycletest. The broader selected tests passed twice under the race detector.

   Temporary overlays confirmed that removing the central guard breaks snapshot regressions and restoring direct publication breaks both fixture-value cases. The added failed-name-probe regression fails on pinned HEAD. Temporary files were removed; repository files remain unchanged. Full-suite and live CI results were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared query boundary and client parser.
   - **ARCH-PURE: pass** — small IO guards; ownership IO remains injected.
   - **ARCH-PURPOSE: flag** — incomplete uncertainty sweep and acceptance mismatch.
   - **ARCH-MOCK: pass** — stateful executable fake uses the production boundary; live conformance lane exists.
   - **ARCH-CONSTRAINTS: pass** — bounded queries; no added retries or fan-out.
   - **ARCH-SECURE: flag** — failed external evidence still becomes name acceptance.
   - **ARCH-ORDER: flag** — that false acceptance contaminates subsequent monotone decisions.
   - **ARCH-FUNERAL: pass** — no new production durable artifacts; fixtures use temporary-directory cleanup.

   Atlas coverage is present. No new command, flag, or configuration surface requires a README addition. No Core concepts table is present.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing delivered Pair boundary regressions from outstanding upstream receiver regressions, and update Done-when accordingly. Extend the probe outcome enumeration to cover nonzero diagnostic failures and cancellation with partial output.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.190809833s
head: "77145de9091256e8390296f2fe10cf127bc2679c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T02:02:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 562 — Zellij CLI queries can finish before their reply |
| repo | pair |
| issue file | workshop/issues/000562-zellij-empty-reply-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 965fcb8147cbb4f63a43d867a571cff7afa140e0..54d5b2defa4d2c44f827a17cd9100c89f3069afd |
| command | sdlc close --issue 562 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T02:02:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are addressed. The implementation preserves uncertainty for unanswered reads and failed name probes, retains established empty-inventory evidence, and adds no retries. The revised acceptance criteria accurately distinguish delivered Pair regressions from outstanding upstream protocol work. No new blocking findings found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      osruntime.go:128–144 checks cancellation before validator evidence and propagates unrecognized subprocess failures. TestFailedZellijNameProbeCannotTeachMonotoneBounds covers failed diagnostics and partial stdout/stderr through the production monotone acceptor. It passes on HEAD; restoring the previous probe implementation via a temporary overlay makes all five cases fail.
  - id: BR-2
    disposition: addressed
    note: |
      The pinned before/after diff revises issue lines 52 and 59 to claim Pair boundary regressions, explicitly leaving upstream receiver tests and captured IPC ordering outstanding. This matches zellij_empty_reply_test.go:116–124, which models CLI completion rather than ConnStatus, and the internal draft’s requested upstream scenarios.
```

1. **Strengths**

   - `zellij.go:202` centralizes unanswered-read rejection while preserving the explicit empty-inventory diagnostic.
   - `createflow.go:1387` retains assignment-local uncertainty without learning positive or negative length bounds.
   - Stateful tests verify no automatic retry, recovery through fresh observations, and unchanged ownership-generation fences.
   - The live PID/pane fixture uses the same atomic publisher exercised by the deterministic barrier test.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Full launcher tests passed. Focused launcher, couchcore, and shared-fake tests passed normally and under the race detector. Public-command and production-registration composition tests passed.

   Counterfactual overlays independently made regressions fail when restoring the previous name probe, removing the shared no-reply guard, or restoring direct fixture publication. Temporary overlays were removed; the checkout remains unchanged.

   The repository-wide suite and live conformance were not independently rerun. Pinned upstream [query receiver](https://github.com/zellij-org/zellij/blob/efd8fd5a89a20c07a111d248ad7fce53848d2c18/zellij-client/src/cli_client.rs#L212-L258) and [discovery probe](https://github.com/zellij-org/zellij/blob/efd8fd5a89a20c07a111d248ad7fce53848d2c18/zellij-utils/src/sessions.rs#L148-L167) support the documented possible interleavings.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared read boundary and client-count parser.
   - **ARCH-PURE: pass** — small IO guards; existing injected ownership seam retained.
   - **ARCH-PURPOSE: pass** — enumerated authority consumers preserve uncertainty; acceptance matches delivery.
   - **ARCH-MOCK: pass** — stateful executable fake exercises production boundaries; live conformance lane exists.
   - **ARCH-CONSTRAINTS: pass** — bounded queries, no added retries or fan-out.
   - **ARCH-SECURE: pass** — failed external evidence cannot grant name or ownership authority.
   - **ARCH-ORDER: pass** — unknown observations stop assignment; fresh observations and generation replacement are covered.
   - **ARCH-FUNERAL: pass** — no new production durable artifacts; fixtures use temporary-directory cleanup.

   Atlas coverage is present. No new command, flag, keybinding, or configuration surface requires a README update. No Core concepts table is present.

7. **Plan revision recommendations:** None.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.824387583s
head: "54d5b2defa4d2c44f827a17cd9100c89f3069afd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T04:08:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 562 — Zellij CLI queries can finish before their reply |
| repo | pair |
| issue file | workshop/issues/000562-zellij-empty-reply-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 81ca972350c61881306f0393a666f03c5efa2809..98f4336f2382c360894192080638cb678789d9cc |
| command | sdlc close --issue 562 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T04:08:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides’ intent in all three conflict resolutions. Prior fixes remain intact, the native fixture correction has a demonstrated failing counterfactual, and no new defects or scope expansion were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      osruntime.go:144 checks cancellation before interpreting validator evidence and rejects unrecognized subprocess failures. The five failed-probe regressions and eleven validator controls pass on the pinned head; this interdiff preserves that implementation.
  - id: BR-2
    disposition: addressed
    note: |
      Inspected the historical before/after correction: Done-when and Plan now claim Pair boundary regressions and explicitly retain upstream receiver tests as outstanding. This matches zellij_empty_reply_test.go:116–124. The current window preserves those criteria.
```

1. **Strengths**

   - `osruntime.go:118` retains main’s bounded query and saved-layout preservation.
   - `terminal_registration_fixture_test.go:112` distinguishes observed empty results from unanswered queries.
   - `selected_root_test.go:84` combines explicit failure injection, client headers, and positive pane evidence while preserving attachment assertions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused native-registration and composition tests passed normally and under `-race`. Selected-root, failed-name-probe, validator-control, and empty-observation tests passed.

   Restoring the previous native fixture helper through a temporary overlay makes the new regression fail for both listing modes and clients. Temporary files were removed; the checkout remains unchanged. The full suite and live conformance were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY: pass** — updates the existing shared fixture rather than adding parallel behavior.
   - **ARCH-PURE: pass** — changes remain in the subprocess boundary and integration fixtures.
   - **ARCH-PURPOSE: pass** — conflict resolutions preserve uncertainty handling and main’s admission behavior.

   No new architectural or user-facing surface requires atlas or README changes. No Core concepts table applies.

7. **Plan revision recommendations:** None. The source-identity correction and integration notes record corrections without expanding scope.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.388849875s
head: "98f4336f2382c360894192080638cb678789d9cc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
