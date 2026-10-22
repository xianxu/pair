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
