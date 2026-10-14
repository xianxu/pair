# Boundary Review — pair#530 (whole-issue close)

| field | value |
|-------|-------|
| issue | 530 — couch: score stale session files and stuck slots from activity Couch already sees (baseline: 15 min without a write → ping) |
| repo | pair |
| issue file | workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 386e0b48e13a57bf37f4a0584291015aab630c75..717bf3ca7caa968ebf70bebad83f9bda011ad58f |
| command | sdlc close --issue 530 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T21:26:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation follows the revised rules-only scope and separates deterministic decisions from delivery effects. Three behavioral gaps block shipping: generated wait notices still create file obligations, asynchronous startup-prompt failures disappear silently, and observation-capacity exhaustion disables supervision. Broker tests passed, but missing generated assets prevented running couchcmd tests.

Done-when: stale — the primary section still requires the deferred file-ping baseline, feature logging, and sampling; the revised acceptance criteria appear only under Revisions.

### 1. Strengths

- Pure transitions cover exact thresholds, completion after returning to main, stale report generations, and submission resets.
- Report observations run after caller authentication, including attempts refused because the hub is absent.
- Startup prompts reuse normal broker admission and queues; the coordinator introduces no persistent report replay.
- README, atlas, embedded skill, and artifact inventory describe the new surface and explicitly identify deferred work.

### 2. Critical findings

- **Generated wait notices still create session-file obligations.** `cmd/internal/couchcmd/wait_notice.go:397` sends directly through the broker without registering system provenance. Consequently, `message_service.go:1058` passes these messages to `deliveryRecorder.record`, which appends an owed row. This violates the revised contract excluding prompts **and wait notices** from file obligations. Route all generated notices through shared provenance handling and test both exclusion and ordinary-message inclusion. **ARCH-PURPOSE**

- **Startup delivery failures and uncertainty are silently ignored.** `cmd/internal/couchcmd/supervision.go:168` checks admission errors only. Later endpoint failures return immediately at lines 78–80; queued expiry never reaches that endpoint callback. Since the address is already marked attempted, no fresh prompt or required failure notice follows. Observe terminal broker outcomes and emit one readable notice for failed/unconfirmed prompts without retrying them. **ARCH-ORDER**

- **Observation-table saturation silently disables supervision.** `cmd/internal/couchcmd/supervision.go:34–46` retains up to 256 IDs for two hours, including refused attempts and failed generated sends. At capacity, `sending` returns before recording reports or reply provenance (lines 65–66), while subsequent nudges cannot send. This can happen while the broker still has capacity because refused attempts consume no broker receipt. Separate deduplication from authoritative report/action tracking, define bounded overflow behavior, and test saturation followed by a new report and confirmed reply. **ARCH-CONSTRAINTS**

### 3. Important findings

- **Update the primary acceptance criteria.** `workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md:35–37` still requires work explicitly deferred to parley.nvim#321 and pair#543. Move the revised rules-only acceptance into `## Done when`, retaining the scope-change history under Revisions.

### 4. Minor findings

None.

### 5. Test coverage notes

- Ran the required stat and name-status recipes before targeted patch inspection; HEAD matches the pinned commit.
- Ran `go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'TestSupervision|Test.*Submission' -count=1`.
- **couchmessage passed.** **couchcmd failed during setup:** `cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`.
- Root `AGENTS.md` is absent; inspected `AGENTS.local.md`.
- Existing uncertainty tests assert no duplicate prompt but discard notices, so they cannot catch the missing failure notification.
- Add regressions for wait-notice ledger exclusion, asynchronous prompt cancellation/expiry/uncertainty, and observation-capacity overflow. No tracked files were modified.

### 6. Architectural notes

- **ARCH-DRY — pass:** existing broker admission and watcher lifecycle are reused.
- **ARCH-PURE — pass:** rule transitions contain no IO; the coordinator executes effects.
- **ARCH-PURPOSE — flag:** generated-message exclusion does not cover the full promised class.
- **ARCH-MOCK — pass:** integrations use existing stateful endpoint fixtures and private stores.
- **ARCH-CONSTRAINTS — flag:** capacity is bounded, but overflow silently disables required behavior.
- **ARCH-SECURE — pass:** authenticated report observations and identity-based prompt provenance avoid body-prefix impersonation.
- **ARCH-ORDER — flag:** startup admission is modeled, but subsequent failed/unconfirmed outcomes lack reconciliation.
- **ARCH-FUNERAL — pass:** new state is volatile, report episodes are pruned, and no new durable artifact family is introduced.

The final plan revision accurately names the implemented core entities and supersedes the historical concept tables.

### 7. Plan revision recommendations

Add `## Revisions` entries specifying:

- Shared provenance handling for every supervisor-generated message, including existing wait notices.
- Startup terminal-outcome handling and one-time failure/uncertainty notices.
- Observation-capacity behavior that preserves report/action correctness.
- Synchronization of the issue’s primary Done-when section with approved scope.

```findings
findings:
  - id: new
    severity: Critical
    family: generated-message-accounting
    title: |
      Generated wait notices still create session-file obligations
    detail: |
      cmd/internal/couchcmd/wait_notice.go:397 bypasses system-provenance registration, so message_service.go:1058 records delivered wait notices as owed rows. ARCH-PURPOSE: apply the revised exclusion contract to all generated notices and add ledger regression coverage.
  - id: new
    severity: Critical
    family: asynchronous-outcome-reconciliation
    title: |
      Startup prompt failures after admission produce no required notice
    detail: |
      cmd/internal/couchcmd/supervision.go:168 checks only admission errors, while lines 78–80 discard failed or unconfirmed delivery outcomes and queued expiry has no coordinator observer. ARCH-ORDER: reconcile terminal broker outcomes into one readable failure notice without regenerating the prompt; test cancellation, expiry, and uncertainty.
  - id: new
    severity: Critical
    family: bounded-observation-overflow
    title: |
      Observation capacity exhaustion silently disables report and reply tracking
    detail: |
      cmd/internal/couchcmd/supervision.go:34–46 retains failed attempts alongside successful sends for two hours; once 256 entries accumulate, lines 65–66 discard new reports and reply provenance, and generated nudges also fail admission to this table. ARCH-CONSTRAINTS: preserve authoritative report/action observations under bounded overflow and test saturation with subsequent successful traffic.
  - id: new
    severity: Important
    family: acceptance-scope-consistency
    title: |
      Primary Done-when criteria still require explicitly deferred work
    detail: |
      workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md:35–37 requires the file-ping baseline, feature vectors, and sampling despite the approved rules-only revision. Update the primary section to the revised acceptance criteria and retain links to the companion issues.
```

---

## Re-review — 2026-10-13T22:00:52-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 530 — couch: score stale session files and stuck slots from activity Couch already sees (baseline: 15 min without a write → ping) |
| repo | pair |
| issue file | workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 386e0b48e13a57bf37f4a0584291015aab630c75..03ef39d94b27982d1c793574d1612005c1d3d0c1 |
| command | sdlc close --issue 530 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T22:00:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-2, BR-3, and BR-4 are addressed. BR-1 remains blocking: receipt reconciliation can discard generated-message provenance while a delivery callback is still running, allowing automatic notices to create session-file obligations. A deterministic test reproduced this for both startup and worker-to-hub generated sends in all three repetitions.

1. **Strengths**

   - Pure transition tests cover thresholds, suppression, coalescing, replacements, and stale report generations.
   - Generated sends reuse normal broker admission and queue behavior.
   - BR-2 and BR-3 regression tests pass at the pinned head and fail with the pre-fix implementation restored.
   - README and atlas document the new behavior; the issue’s primary acceptance criteria correctly defer file pings and logging.

2. **Critical findings**

   **BR-1 — not-addressed; `generated-message-accounting` (ARCH-ORDER, ARCH-PURPOSE).**
   At [supervision.go:128](/Users/xianxu/Library/Caches/sdlc/review-checkouts/7245ed5d2f9f6b1f/cmd/internal/couchcmd/supervision.go:128), reconciliation deletes provenance after observing a terminal receipt. However, sender disconnection can make that receipt terminal before the endpoint returns. [message_service.go:1060](/Users/xianxu/Library/Caches/sdlc/review-checkouts/7245ed5d2f9f6b1f/cmd/internal/couchcmd/message_service.go:1060) subsequently looks up the deleted provenance and records an uncertain owed row.

   Reproduction: block endpoint delivery → disconnect sender → reconcile receipts → release endpoint with cancellation error → observe an owed row for the generated message.

   **This is the 2nd finding in family `generated-message-accounting`.** Fix the rule across startup prompts, rules nudges, and ordinary wait notices: generated provenance must remain available through completion of delivery accounting, independently of receipt terminality. Capture immutable provenance before entering endpoint IO, or otherwise retain it until the callback finishes. Add controlled late-completion regressions across these generated-message paths.

3. **Important findings**

   None beyond the unresolved critical finding.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Pinned-head supervision/wait tests passed in both affected packages.
   - Focused regression and broker tests passed under `-race`.
   - Restoring pre-fix production files made all BR-1/2/3 review regressions fail, confirming meaningful coverage of their original cases.
   - The added late-completion test failed consistently; existing tests omit this ordering.
   - Generated missing runtime assets in a disposable copy. Removed that copy afterward; the review checkout remains unchanged. The full suite was not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared generated-send adapter and existing broker policy.
   - **ARCH-PURE — pass:** deterministic report transitions separated from IO.
   - **ARCH-PURPOSE — flag:** BR-1 leaves the generated-message exclusion contract incomplete.
   - **ARCH-MOCK — pass:** stateful endpoint fixtures and private socket/storage seams.
   - **ARCH-CONSTRAINTS — pass:** separate observation-cap failure removed; saturation regression verified.
   - **ARCH-SECURE — pass:** authenticated report hooks and structural provenance rather than text-prefix exemptions.
   - **ARCH-ORDER — flag:** terminal receipt does not imply delivery callback completion.
   - **ARCH-FUNERAL — flag:** provenance cleanup occurs before its last consumer finishes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing receipt completion from callback completion. State the provenance lifetime invariant, enumerate all generated-message producers, and require controlled disconnect/late-completion tests. Amend the boundary-correction passage that currently treats terminal receipts as sufficient grounds for deleting all metadata.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Registration is fixed, but supervision.go:128 removes provenance before an in-flight endpoint callback necessarily finishes. message_service.go:1060 then records generated traffic as owed. Controlled disconnect/reconcile/late-return tests fail for both generated-send directions in three repetitions. Preserve provenance through delivery accounting across the entire generated-message-accounting family.
  - id: BR-2
    disposition: addressed
    note: |
      Terminal prompt outcomes and queued cancellation now produce one actionable notice without resending. Relevant regressions pass at pinned head and fail with pre-fix production files restored; broker tests also verify queue expiry.
  - id: BR-3
    disposition: addressed
    note: |
      Failed attempts no longer consume a separate capped observation history. The regression exceeding MaxReceipts and then sending a successful report/reply passes at head and fails against the pre-fix implementation.
  - id: BR-4
    disposition: addressed
    note: |
      The pinned Done when diff now requires rules, startup prompts, exclusions, and outcome coverage, explicitly assigning file pings to parley.nvim#321 and logging to pair#543. This matches the approved rules-only revision and implemented scope.
```

---

## Re-review — 2026-10-13T22:22:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 530 — couch: score stale session files and stuck slots from activity Couch already sees (baseline: 15 min without a write → ping) |
| repo | pair |
| issue file | workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 386e0b48e13a57bf37f4a0584291015aab630c75..2b0b19c37c73169826ee96eaefa64ace3f681c30 |
| command | sdlc close --issue 530 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T22:22:20-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the revised rules-only scope. BR-1 is addressed with verified red/green regression evidence across all three generated-message paths. No new blocking findings emerged. The reviewed repository remains unchanged, and the disposable test copy was removed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      supervision.go:124 retains provenance while delivery accounting remains active. TestSupervisionReviewLateDeliveryKeepsGeneratedProvenance passes at head; removing that guard fails startup, rules-nudge, and ordinary-wait cases with unwanted ledger obligations. The genuine-message control remains recorded.
  - id: BR-2
    disposition: addressed
    note: |
      Prior disposition retained. Terminal-prompt and queued-cancellation regressions pass; reconciliation produces one actionable notice without regenerating the prompt.
  - id: BR-3
    disposition: addressed
    note: |
      Prior disposition retained. Failed attempts no longer occupy a capped history; the regression exceeding MaxReceipts still tracks a subsequent report and confirmed reply.
  - id: BR-4
    disposition: addressed
    note: |
      Prior disposition retained. The issue's Done when explicitly assigns file pings to parley.nvim#321 and logging to pair#543, consistent with the revised plan and implemented rules-only scope.
```

1. **Strengths**
   - Generated-message provenance survives terminal receipts until endpoint accounting completes (`supervision.go:121`; `broker.go:395`).
   - Controlled late-return tests exercise cancellation ordering and verify both exclusion and eventual metadata cleanup (`supervision_review_test.go:175`).
   - Pure rules cover thresholds, completion after returning to main, stale outcomes, suppression, and once-per-address startup attempts.
   - README, atlas, embedded skill, and artifact inventory document the new surface.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed full `couchcmd`, `couchmessage`, and `sessionsync` package tests.
   - Passed focused supervision tests with `-race`.
   - Verified BR-1 fails without the lifetime guard.
   - Initial couchcmd compilation required the repository’s generated runtime assets; generated these only in the disposable copy.
   - Full repository `make -k test` and live smoke were not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses normal broker admission and the existing watcher.
   - **ARCH-PURE — pass:** Rule transitions remain free of IO; the coordinator performs effects.
   - **ARCH-PURPOSE — pass:** Revised acceptance is implemented; all three generated-message paths use provenance accounting.
   - **ARCH-MOCK — pass:** Integration tests use stateful endpoints and private sockets through production seams.
   - **ARCH-CONSTRAINTS — pass:** Reuses broker capacity and cached observations without another worker or retained report history.
   - **ARCH-SECURE — pass:** Authenticated observations and structural provenance prevent matching prose from acquiring an exemption.
   - **ARCH-ORDER — pass:** Controlled tests reproduce terminal-receipt/late-callback ordering; production preserves uncertainty until accounting completes.
   - **ARCH-FUNERAL — pass:** Completed metadata is reconciled away; report and attempt state has an explicit supervisor lifetime.

7. **Plan revision recommendations:** None. Existing revisions reconcile the historical concept tables with the final implementation.
