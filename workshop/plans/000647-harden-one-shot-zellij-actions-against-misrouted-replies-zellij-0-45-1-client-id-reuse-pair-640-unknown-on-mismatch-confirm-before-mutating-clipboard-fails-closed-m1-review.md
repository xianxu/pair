# Boundary Review — pair#647 (milestone M1)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..b4236f4ef378d88e5bee66906b828d0f28f298ce |
| command | sdlc milestone-close --issue 647 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T12:23:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 establishes useful strict parsers, bounded confirmation, and query-process ownership. However, lifecycle admission still contains a blind retry and accepts contradictory session listings. An existing Couch integration control also fails deterministically against the stricter contracts. These need correction before closing M1.

```findings
findings:
  - id: new
    severity: Critical
    family: unknown-observation-is-terminal
    title: |
      Launcher Couch attachment still retries unknown observations into authority
    detail: |
      cmd/internal/launcher/couch_session.go:149 retries every SessionLiveness error. A malformed or wrong-kind first reply now rejected by the strict parser can therefore be replaced by a valid second reply, allowing verifyCouchSessionAttach to succeed and attachment to continue. This violates the approved no-retry contract and ARCH-PURPOSE/ARCH-ORDER. Remove this retry and add a production-path regression asserting that unknown first evidence prevents attachment even when the next reply would be valid.
  - id: new
    severity: Critical
    family: contradictory-evidence-is-unknown
    title: |
      Individually valid but contradictory session listings still grant live authority
    detail: |
      cmd/internal/launcher/zellij.go:116-142 validates short and full listings independently without reconciling their session identities. For example, short output containing pair-a and full output containing only pair-b both pass; snapshot then reports pair-a as SessionLive at line 98 because omission from the full listing is treated as absence of an EXITED flag. ListedPairSessionContext consumes this projection as positive attachment evidence. Under ARCH-SECURE/ARCH-ORDER, conflicting observations must remain unknown. Reconcile the decision-relevant listing facts before one-shot admission and cover contradictory listings with zero-effect regressions.
  - id: new
    severity: Important
    family: strict-contract-fixture-conformance
    title: |
      Existing terminal lifecycle fixture fails the new listing contract
    detail: |
      cmd/internal/couchcore/terminal_registration_fixture_test.go:147 emits only a session name for both list-sessions forms. Consequently, go test ./cmd/internal/couchcore -run '^TestListedDetachedTerminalAttachesWithoutNewServer$' -count=1 fails deterministically at terminal_warm_fixture_test.go:171 with “session listing row 1 lacks creation metadata”. Update the shared fixture to emit native-valid full listings and owner-pane fields, then rerun its lifecycle consumers without weakening production validation.
```

1. **Strengths**

   - `zellijreply.Confirm` stops after invalid first evidence, preserves cancellation, and returns the second admitted snapshot.
   - Owner admission brackets both pane observations with exact server-generation checks.
   - Warm-attach tests explicitly exercise simultaneous helper exit and observation refusal, preserving both error causes.
   - The stateful fake records actual targets and effects through the subprocess seam.
   - Atlas documentation clearly distinguishes completed M1 work, pending milestones, and residual correlated-reply risk.

2. **Critical findings**

   - The launcher retry bypasses the newly strict failure policy. Remove it and test the complete attach path.
   - Listing validation checks shape without checking agreement. Reject contradictory decision facts before deriving live authority.

3. **Important findings**

   - Repair the terminal fixture and restore its existing integration control.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed complete packages: `launcher`, `zellijreply`, `zellijtest`, `zellijexec`, and `zellijpane`.
   - Passed race checks: `zellijreply`, `zellijtest`, and `zellijexec`.
   - Targeted Couch checks found the deterministic fixture failure above.
   - A separate public warm-attach control timed out during the combined run but passed when rerun alone; that does not establish an additional defect.
   - No native qualification is claimed here; the plan assigns it to M3. The checkout remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared parsers, confirmation, and process-group inspection consolidate behavior.
   - **ARCH-PURE — pass:** parsing and projections remain separate from subprocess execution.
   - **ARCH-PURPOSE — flag:** the lifecycle retry sweep missed `verifyCouchSessionAttach`.
   - **ARCH-MOCK — pass for M1:** stateful external fake uses the production subprocess boundary; native qualification remains pending.
   - **ARCH-CONSTRAINTS — pass for inspected new mechanisms:** confirmation uses bounded contexts; warm query deadlines preserve helper lifetime.
   - **ARCH-SECURE — flag:** contradictory listings become positive evidence.
   - **ARCH-ORDER — flag:** a failed observation can still be replaced by later success.
   - **ARCH-FUNERAL — pass:** fake roots are test-owned, evidence is call-scoped, and query processes have explicit cleanup.

   Atlas coverage is present. M1 introduces no new command, flag, keybinding, or configuration key requiring a README update.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording:

   - The remaining launcher attach retry and the lifecycle-owner sweep/regression that removes it.
   - The explicit agreement rule for short/full listing evidence.
   - The actual confirmation-helper location: `cmd/internal/zellijreply/reply.go`, replacing the stale `confirm.go` reference.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1812c841-1c00-42c8-9326-8058ce3f0839","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/dc3c9f8c8c58455f","reviewed_head":"b4236f4ef378d88e5bee66906b828d0f28f298ce","slot":"pair:1","requested_at":"2026-10-17T19:20:50.357935Z"},"phase":"finished","pid":42448,"started_at":"2026-10-17T19:20:50.468522Z","finished_at":"2026-10-17T19:23:49.481621Z","session_id":"01a14b4f-4d0d-7c71-bca5-f3b997f089c3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.373117167s
head: "b4236f4ef378d88e5bee66906b828d0f28f298ce"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
