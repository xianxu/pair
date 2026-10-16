# Boundary Review — pair#569 (whole-issue close)

| field | value |
|-------|-------|
| issue | 569 — Couch → Qoder delivery: text typed into Qoder's input but not submitted (pending * prompt); confirm submission |
| repo | pair |
| issue file | workshop/issues/000569-couch-qoder-delivery-text-typed-into-qoder-s-input-but-not-submitted-pending-prompt-confirm-submission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99e11ecb8c72b7e5325a58692c1efd679e50710c..35c420d4db5f19ea1f573400ebc2614949e9602d |
| command | sdlc close --issue 569 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:22:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The retry, confirmation, and quarantine changes are well structured, and focused tests pass normally and under the race detector. Two inexpensive corrections remain: recovery instructions name the wrong submission key, and README guidance still describes the previous behavior. Native Qoder acceptance remains pending under the plan’s explicit TL-directed hold; this review does not establish it.

```findings
findings:
  - id: new
    severity: Important
    family: recovery-instructions-match-input-routing
    title: |
      Recovery notices instruct Enter, which inserts a newline in Qoder
    detail: |
      cmd/internal/couchcmd/automatic_input_stall.go:63 and cmd/internal/couchmessage/delivery.go:89–91 tell the operator to press Enter. The Qoder profile in cmd/internal/wrapcmd/harness_tty.go:90–98 maps plain Enter to backslash-CR for a newline in a recognized composer; Alt+Enter emits the submission CR. Following the notice therefore leaves the automatic message pending. Correct all new recovery instructions and add a regression that connects the prescribed recovery action to production input routing rather than only asserting the notice string. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: user-facing-contract-documentation
    title: |
      README update is missing for changed confirmation and operator recovery
    detail: |
      README.md:724–736 still says a newly opened turn can confirm submission, says the wrapper does not reconstruct the envelope, and gives no quarantine recovery guidance. This range changes all three for Qoder and introduces an operator action, but does not update README.md. Document the Qoder exception, bounded key-only retries, prompt refusal during quarantine, and the correct manual recovery action. atlas/couch.md was updated appropriately.
```

1. **Strengths**

   - Retry effects pass through the existing reducer and writer seam, with a three-attempt limit and no body repaste.
   - Qoder confirmation requires fresh rendered empty-composer evidence; unrelated turn changes cannot establish success.
   - Stateful receiver tests exercise dropped submits, changed text, interruptions, terminal refusal, and next-delivery recovery.
   - Quarantine reuses authenticated reporting and normal render wakes without adding idle polling.

2. **Critical findings**

   None.

3. **Important findings**

   The two findings above require recovery-key and README corrections.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused `Qoder|AutomaticInput|PeerDelivery` tests across `couchmessage`, `wrapcmd`, and `couchcmd`, including race runs. Return-routing tests also passed. `git diff --check` passed.

   The new notice tests currently pin the incorrect wording without validating the prescribed action. Native idle, mid-turn, and subsequent-delivery acceptance remains explicitly unchecked; retain the landing hold.

6. **Architectural notes**

   - **ARCH-DRY — PASS:** existing reducers, parser, writer, and reporter are reused.
   - **ARCH-PURE — PASS:** deterministic evidence and transition logic remain separate from effects.
   - **ARCH-PURPOSE — FLAG:** the recovery instruction does not perform the intended submission.
   - **ARCH-MOCK — PASS:** stateful receiver tests share the production writer boundary; native qualification remains pending.
   - **ARCH-CONSTRAINTS — PASS:** bounded attempts, spacing, and existing delivery deadline are enforced.
   - **ARCH-SECURE — PASS:** incomplete or unsupported composer evidence declines ownership; reporting retains authentication.
   - **ARCH-ORDER — PASS:** retry and quarantine transitions are explicit; terminal events suppress further effects.
   - **ARCH-FUNERAL — PASS:** no new runtime artifact family or background worker; existing retention and queue bounds are reused.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting the recovery action to Alt+Enter under normal Pair remapping and requiring a routing-backed recovery test. Preserve the unchecked native acceptance and landing tasks.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8061042d-0637-41f6-9d13-b203d22b9d0a","repository":"github.com/xianxu/pair","issue_id":"000569","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/13e8512e56e7eb33","reviewed_head":"35c420d4db5f19ea1f573400ebc2614949e9602d","slot":"pair:5","requested_at":"2026-10-16T00:20:25.462939Z"},"phase":"finished","pid":3041,"started_at":"2026-10-16T00:20:25.559614Z","finished_at":"2026-10-16T00:22:14.348432Z","session_id":"01a14214-dc3a-7782-875a-7a5d464f5627","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.384790792s
head: "35c420d4db5f19ea1f573400ebc2614949e9602d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T18:15:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 569 — Couch → Qoder delivery: text typed into Qoder's input but not submitted (pending * prompt); confirm submission |
| repo | pair |
| issue file | workshop/issues/000569-couch-qoder-delivery-text-typed-into-qoder-s-input-but-not-submitted-pending-prompt-confirm-submission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99e11ecb8c72b7e5325a58692c1efd679e50710c..aa4c2240dec53d735a567e90105b45b60c621433 |
| command | sdlc close --issue 569 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T18:15:20-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation addresses both prior findings, and no new defects were found. Retries remain bounded and key-only; Qoder confirmation requires fresh empty-composer evidence; quarantine preserves uncertainty and prevents further automatic input. This approves the prepared implementation review. The plan’s TL-directed hold on native acceptance and landing remains in force.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      AutomaticInputRecoveryHint supplies Alt+Enter advice to both receipts and supervisor notices. TestPeerQoderRecoveryAdviceRoutesToNativeSubmit follows that advice through production translateChunk and verifies consumption and quarantine release. In a disposable pinned-head copy, restoring Enter makes the test fail because routing emits backslash-CR.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:724–738 now documents fresh-empty Qoder confirmation, bounded exact-envelope key retries, prompt quarantine refusal and Alt+Enter recovery. README.md:747 distinguishes uncertain body replay. These passages agree with peer_delivery.go, peer_qoder_submission.go and the Qoder keymap in harness_tty.go:90–98.
```

1. **Strengths**

   - Recovery instructions share one constant, with a regression testing the prescribed action through production routing.
   - Stateful receiver tests independently count pasted bodies, submit attempts and accepted envelopes.
   - Quarantine reconciliation uses existing render wakes, including after terminal receipt publication.
   - README, atlas and artifact inventory cover the changed surface.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed focused delivery, Qoder and automatic-input tests across wrapcmd, couchmessage and couchcmd, including race runs. Full couchmessage and artifactpath suites passed. The recovery regression passed unchanged and failed with the prior instruction restored; the disposable copy was removed. The reviewed checkout remains unchanged.

   Native idle, mid-turn and subsequent-delivery acceptance was not performed; it remains explicitly unchecked under the plan revision.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing reducers, writer and reporter reused; recovery advice centralized.
   - **ARCH-PURE — pass:** deterministic evidence and transitions remain separate from terminal and reporting effects.
   - **ARCH-PURPOSE — pass:** automated coverage includes retries, confirmation, residue refusal and subsequent recovery; native acceptance remains the declared held task.
   - **ARCH-MOCK — pass:** stateful receiver shares the production writer boundary; native qualification remains pending.
   - **ARCH-CONSTRAINTS — pass:** three attempts, minimum spacing and original deadline remain bounded.
   - **ARCH-SECURE — pass:** incomplete, altered and unsupported evidence declines retry; reporting retains authentication.
   - **ARCH-ORDER — pass:** reducer transitions govern retries and quarantine; tests exercise interruption, stale evidence and terminal no-ops.
   - **ARCH-FUNERAL — pass:** no new durable runtime artifact family or worker; existing retention and queue bounds are reused.

   Core-concept entities exist at the documented paths and match their declared roles.

7. **Plan revision recommendations:** None; existing revisions accurately preserve the outstanding native and landing obligations.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"87c96b05-0f37-4ab5-9e75-1f51fb8aa40e","repository":"github.com/xianxu/pair","issue_id":"000569","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6ab8c5ffa150519f","reviewed_head":"aa4c2240dec53d735a567e90105b45b60c621433","slot":"pair:5","requested_at":"2026-10-16T01:13:08.03883Z"},"phase":"finished","pid":17889,"started_at":"2026-10-16T01:13:08.236875Z","finished_at":"2026-10-16T01:15:19.710686Z","session_id":"01a14245-1f60-7d11-89c4-af4e610d3d9d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.062156583s
head: "aa4c2240dec53d735a567e90105b45b60c621433"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
