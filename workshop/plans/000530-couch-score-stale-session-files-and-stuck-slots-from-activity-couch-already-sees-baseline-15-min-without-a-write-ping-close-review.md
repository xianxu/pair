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
