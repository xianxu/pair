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

---

## Re-review — 2026-10-17T13:05:40-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..daac3160b931f3344bae2adc31afd21c8ec07153 |
| command | sdlc milestone-close --issue 647 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T13:05:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1, BR-2 and BR-3 are addressed, with regressions verified to fail when their fixes are removed. The strict contracts, listing reconciliation and cancellation tests provide solid coverage. One blocking error-composition gap remains: both query-runner consumers can discard a process-cleanup failure and return successful admission evidence.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The retry is removed. TestCouchSessionAttachStopsAtFirstUnknownListing passes at HEAD and fails at all five production/exported attach fences when the retry is restored in a disposable copy.
  - id: BR-2
    disposition: addressed
    note: |
      Native and cached short/full listings now require exact identity-set agreement before projection or client queries. Removing reconciliation makes the contradictory-listing and production RunLaunch regressions fail.
  - id: BR-3
    disposition: addressed
    note: |
      The shared fixture now emits native-valid full listings and is_plugin fields. Listed-detached and composed-launch consumers pass with isolated Git discovery; reverting the fixture makes TestRegistrationTerminalFixtureDistinguishesListingForms fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: unknown-observation-is-terminal
    title: |
      Absence diagnostics erase query-process cleanup failures
    detail: |
      cmd/internal/launcher/zellij.go:227-230 and cmd/internal/launcher/osruntime.go:175-178 use errors.As to recognize an exit-1 diagnostic, then return success even when zellijexec.Run joined that ExitError with a cleanup failure at cmd/internal/zellijexec/run.go:45-53. Injecting EIO at the collector boundary reproduced nil errors from both production consumers. This violates ARCH-ORDER: a failed query boundary becomes successful absence or name-acceptance evidence. This is the 2nd finding in family unknown-observation-is-terminal. Do NOT fix only one instance: state and enforce the rule that diagnostic normalization may consume only the subprocess exit failure, never cancellation or collection failures. The sweep found this pattern in both of the runner's two production consumers. Preserve cleanup status separately and add regressions for both consumers.
```

1. **Strengths**

   - Listing reconciliation preserves exact names, rejects missing/additional identities and accepts reordered compatible rows.
   - Attachment regressions exercise actual strict parsing through production orchestration.
   - Handoff tests control cancellation and child-exit ordering, including joining post-birth queries before lease release.

2. **Critical findings**

   - The finding above blocks M1. Separate subprocess status from collector errors so accepted native diagnostics cannot suppress ownership failures.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Complete launcher, reply, process-owner, fake, pane and procutil package tests passed. Targeted Couch lifecycle tests passed, as did launcher admission/handoff race tests and reply/fake race tests.

   The composed-launch tests initially failed because temporary repositories discovered the enclosing review checkout. They passed with a disposable Git-discovery barrier. All mutation controls ran in disposable copies that were removed; the pinned checkout remains unchanged. Native qualification and the whole-issue suite were not run.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared validators and process-group observation avoid parallel implementations.
   - **ARCH-PURE — pass:** parsing/projection logic is separate from process execution.
   - **ARCH-PURPOSE — flag:** terminal-unknown handling still misses joined collector failures.
   - **ARCH-MOCK — pass for M1:** stateful CLI tests use the production subprocess boundary; native qualification remains M3 work.
   - **ARCH-CONSTRAINTS — pass:** shared query budgets and independent client lifetimes have deterministic coverage.
   - **ARCH-SECURE — pass:** strict parsing preserves missing facts and rejects contradictory identities.
   - **ARCH-ORDER — flag:** diagnostic normalization collapses failed collection into successful evidence.
   - **ARCH-FUNERAL — pass structurally:** explicit process cleanup and temporary-root ownership exist; cleanup failures must remain visible.

   Atlas updates cover the new internal surfaces. This range introduces no user-facing command, flag or configuration requiring a README update.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that subprocess diagnostic normalization cannot suppress cancellation or collection failures, enumerating both `zellijexec.Run` consumers and requiring regression coverage at each.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ff41bbe4-d232-453e-ace3-310896fa8f77","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9c75f4d5e27995bc","reviewed_head":"daac3160b931f3344bae2adc31afd21c8ec07153","slot":"pair:1","requested_at":"2026-10-17T20:01:49.243621Z"},"phase":"finished","pid":90232,"started_at":"2026-10-17T20:01:49.461273Z","finished_at":"2026-10-17T20:05:39.167824Z","session_id":"01a14b74-d389-7ed3-b767-c5c1be0263a0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.738749542s
head: "daac3160b931f3344bae2adc31afd21c8ec07153"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T13:19:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair647-query-fix |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..dad70e903c4abaa882f4d38b3fa84680fa8c7bba |
| command | sdlc milestone-close --issue 647 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T13:19:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 matches the Spec and revised Plan. BR-4 is addressed across both production consumers, with regression evidence that fails when the correction is removed. No new blocking findings emerged. This verdict covers M1; M2/M3 and native refusal qualification remain pending.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Terminal unknown remains final across launcher and Couch attach fences; production-path and controlled helper-exit regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Native and cached short/full listings must agree on exact identities before granting authority; contradictory and reordered controls pass.
  - id: BR-3
    disposition: addressed
    note: |
      The registration fixture distinguishes short/full listing forms and supplies typed pane evidence; selected registration controls pass.
  - id: BR-4
    disposition: addressed
    note: |
      Both consumers normalize only direct ExitError values and preserve collector plus cancellation causes. TestQueryCollectionFailuresCannotBecomeDiagnosticEvidence passes at head; restoring pre-fix handling in a disposable copy produces six failing subtests. The copy was removed.
```

1. **Strengths**

   - Strict response contracts remain separate from tolerant advisory parsing.
   - Both diagnostic boundaries preserve independent cleanup failures: `cmd/internal/launcher/zellij.go:226` and `cmd/internal/launcher/osruntime.go:170`.
   - Shared admission deadlines preserve interactive-client lifetime, with controlled cancellation and child-exit tests.
   - Atlas documentation accurately distinguishes delivered M1 behavior from pending work and states the residual stale-reply limitation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the complete launcher package; complete `zellijreply`, `zellijexec`, and `zellijtest` packages; selected Couch warm/detached/reply/registration tests; and targeted launcher/Couch race checks. BR-4’s mutation check failed as expected in all six fault cases. The full Couch package and live native qualification were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared contracts, process ownership, and exact-group inspection avoid duplicated boundary logic.
   - **ARCH-PURE — pass:** Parsing and projections remain separate from injected process/runtime seams.
   - **ARCH-PURPOSE — pass:** M1 covers lifecycle admission; BR-4 sweeps both production consumers.
   - **ARCH-MOCK — pass:** Stateful fake uses the production subprocess seam; native qualification remains assigned to M3.
   - **ARCH-CONSTRAINTS — pass:** Query budgets are shared and caller-capped, independently of client lifetime.
   - **ARCH-SECURE — pass:** External replies require explicit typed evidence before authority.
   - **ARCH-ORDER — pass:** Unknown observations terminate admission; tests control cancellation and helper-exit ordering.
   - **ARCH-FUNERAL — pass:** Query work is collected; fixture artifacts have test-owned cleanup.

7. **Plan revision recommendations:** None. Existing revisions explain the implementation locations, query phases, and diagnostic-normalization rule. Atlas updates cover the new architectural surface; M1 adds no user-facing syntax requiring a README update.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"25b927cd-c808-4b0c-8daa-b7932a475749","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ebe4614c77573ff9","reviewed_head":"dad70e903c4abaa882f4d38b3fa84680fa8c7bba","slot":"pair:1","requested_at":"2026-10-17T20:17:31.970583Z"},"phase":"finished","pid":16919,"started_at":"2026-10-17T20:17:32.108502Z","finished_at":"2026-10-17T20:19:46.599697Z","session_id":"01a14b83-3509-7ce1-b876-4e83c0953d92","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.808236791s
head: "dad70e903c4abaa882f4d38b3fa84680fa8c7bba"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
