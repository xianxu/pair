# Boundary Review — pair#791 (whole-issue close)

| field | value |
|-------|-------|
| issue | 791 — Couch delivery: capture delayed refusal evidence |
| repo | pair |
| issue file | workshop/issues/000791-couch-admit-idle-claude-slot-refused-as-busy-auto-mode-footer-update-banner-name-which-busy-condition-fired.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c695793916f7baa49f2666835b42e92ac03a647..e131701684bcc59bdb2920aa319c6809b341d7fc |
| command | sdlc close --issue 791 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T12:54:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The diagnostic-only scope is respected, admission policy remains unchanged, and the exercised tests pass. Shipping is blocked by an encoder defect that can omit a screen which fits within the record limit, plus unavailable tracker evidence for the required recognition follow-up.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: preserve-evidence-within-budget
    title: |
      The encoder can discard the entire screen even when it fits within 64 KiB
    detail: |
      cmd/internal/wrapcmd/peer_wait_capture.go:105 limits raw ANSI rows to left/6-64 before measuring their encoded size. A supported 2000×1 screen alternating default and bold cells produces roughly 17 KiB of ANSI and 37 KiB of JSON, but the raw-row allowance cannot exceed 10,858 bytes, so the encoder emits no rows. This violates the plan's full-screen-when-it-fits contract (ARCH-PURPOSE). Budget against actual JSON encoding while retaining bounded construction, and add a regression asserting this fitting screen survives intact.
  - id: new
    severity: Important
    kind: unverifiable
    family: verify-required-tracker-deliverables
    title: |
      The required unclaimed recognition follow-up cannot be verified
    detail: |
      workshop/plans/000791-couch-admission-plan.md:171 claims the recognition follow-up is filed, and line 180 identifies pair#810. The pinned checkout contains neither its issue document nor tracker cards; git rev-parse --verify refs/remotes/origin/issue-tracker fails with “fatal: Needed a single revision”. No fetch was attempted. Provide a pinned tracker snapshot or measured host evidence showing #810 exists, remains unclaimed, references #791, and waits for the first failing capture.
```

1. **Strengths**

   - Capture selection uses the existing dispatch owner; encoding and writing occur after both admission and mailbox locks are released.
   - `SnapshotGeneration` pairs copied cells with their generation under one terminal lock.
   - Tests exercise threshold timing, repeated polls, recovery, writer failure, supported personalities, and Claude batch preparation.
   - Atlas updates explain privacy, retention, truncation, and unavailable diagnostics.

2. **Critical findings**

   - `peer_wait_capture.go:105`: replace the conservative row rejection with actual encoded-byte budgeting. Preserve complete fitting screens and test that invariant.

3. **Important findings**

   - `000791-couch-admission-plan.md:171`: supply authoritative tracker evidence for #810. Its existence and state currently rest on implementation prose.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Full `wrapcmd` and `diagnosticlog` packages.
   - Targeted capture/polling/buffered-writer tests under the race detector.
   - `artifactpath` tests and `make test-affected-inventory`.
   - Encoder fuzzing: 238 executions.

   The encoder tests check size limits and explicit truncation, but do not require preservation whenever the complete screen fits. The counterexample above comes from byte accounting against the implementation; no files were edited.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses refusal details, terminal snapshots, polling, and diagnostic storage.
   - **ARCH-PURE — pass:** timing transition and encoder are directly testable without external IO.
   - **ARCH-PURPOSE — flag:** avoidable screen omission weakens the central evidence-capture deliverable; follow-up status remains unverified.
   - **ARCH-MOCK — pass:** production dispatch is exercised through existing stateful terminal/peer fixtures.
   - **ARCH-CONSTRAINTS — pass:** once-per-head capture, existing polling, and complete-record cap bound work and output.
   - **ARCH-SECURE — pass:** screen evidence remains local; no new transport or credential handling.
   - **ARCH-ORDER — pass:** diagnostic eligibility has a dedicated transition; capture failure does not authorize input.
   - **ARCH-FUNERAL — pass:** existing diagnostic rotation, retention, and close ownership apply.
   - **ARCH-EVOLVE — pass:** additive diagnostic records introduce no read-modify-write loss.

   Current concept-table entities exist at their stated paths. Atlas coverage is present; no new command, flag, keybinding, or configuration key requires a README update.

7. **Plan revision recommendations**

   No scope change is needed. Retain the full-screen-when-it-fits requirement, record the encoder correction and regression evidence, and attach authoritative follow-up evidence before closing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b17bf4ff-2db2-4325-88cf-5db1e1092799","repository":"github.com/xianxu/pair","issue_id":"000791","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/398f4dddcaa4a9d3","reviewed_head":"e131701684bcc59bdb2920aa319c6809b341d7fc","slot":"pair:1","requested_at":"2026-10-22T19:49:26.750948Z"},"phase":"finished","pid":43940,"started_at":"2026-10-22T19:49:27.013267Z","finished_at":"2026-10-22T19:54:21.74831Z","session_id":"01a16529-4b61-7303-98b5-0ff5f7bd5897","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.752023125s
head: "e131701684bcc59bdb2920aa319c6809b341d7fc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T13:06:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 791 — Couch delivery: capture delayed refusal evidence |
| repo | pair |
| issue file | workshop/issues/000791-couch-admit-idle-claude-slot-refused-as-busy-auto-mode-footer-update-banner-name-which-busy-condition-fired.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c695793916f7baa49f2666835b42e92ac03a647..ecc76e29ec0cc2d0ca5252453ebe19883bdeca86 |
| command | sdlc close --issue 791 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T13:06:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the approved diagnostic-only scope. Capture preserves fitting screens, names the refusal, respects the 15-second threshold, and leaves delivery safety unchanged. Both prior findings are resolved; no new findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The encoder now budgets actual escaped JSON bytes, including envelopes and newline. TestPeerWaitCaptureFittingStyledRow passes at HEAD and fails with the pre-fix encoder through a temporary overlay: “fitting 37009-byte JSON row was discarded: rows=0 truncated=true”. Temporary files were removed.
  - id: BR-2
    disposition: withdrawn
    note: |
      The supplied host evidence is corroborated locally: tracker commit 7ba6cc0eb5f81693864fc39260d0866794d87cab contains card d7f96db618cdea2cd2770c66f859d20e43262bb4 for open, ownerless #810. Its origin/main detail references #791 and explicitly requires waiting for the first real failed-admission capture before implementation.
```

1. **Strengths**

   - [Capture dispatch](/Volumes/TestScratch/review-checkouts/1997963d945d653a/cmd/internal/wrapcmd/peer_delivery.go:445) releases both admission and mailbox locks before encoding and writer invocation.
   - [Exact byte budgeting](/Volumes/TestScratch/review-checkouts/1997963d945d653a/cmd/internal/wrapcmd/peer_wait_capture.go:95) preserves fitting styled screens while bounding the entire JSONL record.
   - Tests exercise threshold timing, duplicate suppression, recovery, failed/disabled diagnostics, Claude/Codex/Qoder, and Claude batch preparation.
   - Atlas additions explain privacy, truncation, storage ownership, retention, and the deferred recognition work.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed during this review:

   - Complete `wrapcmd` package: 134.270s.
   - Complete `diagnosticlog` and `artifactpath` packages.
   - Targeted capture tests under the race detector.
   - Capture and existing wait/preflight tests.
   - Encoder fuzzing: 342 executions.
   - `make test-affected-inventory` and range whitespace checks.

   The BR-1 regression demonstrably fails without the correction. Full-package race testing was not repeated; race coverage here was targeted.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** reuses existing refusal details, snapshots, polling, and diagnostic storage.
   - **ARCH-PURE — pass:** timing and encoding remain directly testable pure functions.
   - **ARCH-PURPOSE — pass:** delivers approved capture scope; recognition work is independently tracked.
   - **ARCH-MOCK — pass:** production dispatch tests use existing stateful terminal/peer fixtures.
   - **ARCH-CONSTRAINTS — pass:** once-per-head capture and whole-record limits bound additional work.
   - **ARCH-SECURE — pass:** bounded evidence remains in local managed diagnostics.
   - **ARCH-ORDER — pass:** capture state changes use the transition function; diagnostic failure never authorizes input.
   - **ARCH-FUNERAL — pass:** existing writer close, rotation, and retention ownership apply.
   - **ARCH-EVOLVE — pass:** additive diagnostic records introduce no lossy read-modify-write path.

   Active concept-table entities exist at their documented paths. Atlas coverage is present. No new command, flag, keybinding, or configuration key requires a README change.

7. **Plan revision recommendations**

   None. The existing diagnostic-only and BR-1 revisions match the implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"27efc22f-a58f-41da-ab9e-6e62bdcb40e4","repository":"github.com/xianxu/pair","issue_id":"000791","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1997963d945d653a","reviewed_head":"ecc76e29ec0cc2d0ca5252453ebe19883bdeca86","slot":"pair:1","requested_at":"2026-10-22T20:02:39.185319Z"},"phase":"finished","pid":90171,"started_at":"2026-10-22T20:02:39.443167Z","finished_at":"2026-10-22T20:06:39.419331Z","session_id":"01a16535-633f-7e52-9718-37e1e964def6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.082185625s
head: "ecc76e29ec0cc2d0ca5252453ebe19883bdeca86"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
