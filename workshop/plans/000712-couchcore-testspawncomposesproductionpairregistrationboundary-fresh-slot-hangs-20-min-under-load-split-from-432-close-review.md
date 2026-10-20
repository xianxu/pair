# Boundary Review — pair#712 (whole-issue close)

| field | value |
|-------|-------|
| issue | 712 — couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432) |
| repo | pair |
| issue file | workshop/issues/000712-couchcore-testspawncomposesproductionpairregistrationboundary-fresh-slot-hangs-20-min-under-load-split-from-432.md |
| boundary | whole-issue close |
| milestone | — |
| window | a040b13bab7015c399ff6c9a81c1b4c02d7d7b24..4f846ccf2248cd84b6dca6324925c8af56b4e526 |
| command | sdlc close --issue 712 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T14:39:22-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The change correctly distinguishes unconfirmed cleanup from proved absence and preserves existing diagnostic formats. Focused tests pass, including race checks. However, blocking thread-store reads bypass the new deadline, so the promised bounded return is not yet enforced. Composed verification also failed before reaching cleanup in this review environment.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: cleanup-budget-covers-all-io
    title: |
      Thread-store reads can block beyond the session cleanup deadline
    detail: |
      ARCH-CONSTRAINTS / ARCH-PURPOSE: cmd/internal/couchcore/couch.go:1004 calls quiesceThreadSession, whose recovery_execute.go:86 GetThread uses context.Background and blocking store locking (threadstore.go:299,182; storelock_unix.go:24-28). The new timeout branch at couch.go:1001 performs another blocking GetThread after expiration. A concurrent store-lock holder can therefore prevent this session phase from returning indefinitely. Carry the shared deadline through all session-phase metadata reads, retain the exact session identity without rereading after expiration, and add contention regressions proving return before the held lock is released.
  - id: new
    severity: Important
    kind: unverifiable
    family: boundary-verification-reaches-target
    title: |
      Composed cleanup and loaded completion could not be verified
    detail: |
      ARCH-MOCK / ARCH-CONSTRAINTS: Running go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(UnknownCleanup|SessionCleanup|PersistentSessionCleanup|PostAckSessionCleanup|DecideStartCleanup|SpawnCompos|RegistrationFixture|RegistrationTerminalFixture|LifecycleFailureReceipt)' -count=1 -timeout=180s failed both ordinary/fresh-slot composed families before readiness with “couch session repository scope changed.” Review TMPDIR is inside the checkout, while testpair.Environment clears GIT_CEILING_DIRECTORIES, allowing discovery of the outer repository. Provide pinned-head host evidence that these cases reach cleanup and pass 20/20 under the documented owned load, or make the fixture work within review confinement and rerun.
```

1. **Strengths**

   - `CleanupKnowledge` defaults to unconfirmed and the pure decider retains ownership across all tested shapes and phases (`startcleanup.go:127`).
   - Timeout handling bypasses destructive reconciliation and preserves the exact pending binding through the existing `StartRecoveredUnknown` transition (`couch.go:753`).
   - The public persistent-refusal regression verifies durable history readback, diagnostic identity and actual show dispatch (`session_cleanup_test.go:52`).
   - Fixture listings now derive identity from persisted server state, process generation and a reachable endpoint, with stale-state regressions.

2. **Critical findings**

   The session deadline does not cover blocking thread-store reads. Fix the complete metadata-read path, including timeout handling, and test contention using the existing `holdStoreLock` helper.

3. **Important findings**

   Composed and loaded completion remain unverified here. The observed failure precedes the behavior under review; it is not evidence that cleanup itself failed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused cleanup, decision, fixture and receipt tests passed normally and with `-race`.
   - Existing post-acknowledgement, helper-reaping and ownership regressions passed.
   - `git diff --check` passed.
   - Current deadline tests exercise repeated refusals and contextual deletion, but leave the thread store uncontended.
   - No prior boundary findings required disposition. No source files were modified.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared cleanup shell and existing recorder reused. |
   | ARCH-PURE | Pass: decision logic remains pure; IO stays in the shell. |
   | ARCH-PURPOSE | Flag: the promised bounded session phase still contains unbounded reads. |
   | ARCH-MOCK | Flag: stateful fixture design is sound, but composed execution could not be verified here. |
   | ARCH-CONSTRAINTS | Flag: the five-second envelope excludes blocking metadata reads. |
   | ARCH-SECURE | Pass: uncertain cleanup retains ownership; fixture state errors refuse visibly. |
   | ARCH-ORDER | Pass: explicit cleanup knowledge feeds existing durable transitions. |
   | ARCH-FUNERAL | Pass: fixture server state is removed and owned accept loops are joined; no retry worker is introduced. |
   | ARCH-EVOLVE | Pass: accepted `launch`/`unproven` values and existing formats are reused. |

   All listed core entities exist at their planned paths. The pure entities require no IO. Atlas documents the new behavior; existing README diagnostic and recovery commands remain applicable, with no new command or configuration surface introduced.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring every session-phase metadata read to respect the shared deadline, eliminating post-expiration rereads and adding held-lock regressions. Record pinned-head composed/load evidence separately from the review-environment failure before marking verification complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"494fdc62-28cc-4f1f-b196-f39c16281080","repository":"github.com/xianxu/pair","issue_id":"000712","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9f2475c8d40546c8","reviewed_head":"4f846ccf2248cd84b6dca6324925c8af56b4e526","slot":"pair:3","requested_at":"2026-10-19T21:37:08.460559Z"},"phase":"finished","pid":76143,"started_at":"2026-10-19T21:37:08.778602Z","finished_at":"2026-10-19T21:39:22.024624Z","session_id":"01a15618-d100-79e0-ab04-fc1d3144413c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.150723666s
head: "4f846ccf2248cd84b6dca6324925c8af56b4e526"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T15:16:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 712 — couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432) |
| repo | pair |
| issue file | workshop/issues/000712-couchcore-testspawncomposesproductionpairregistrationboundary-fresh-slot-hangs-20-min-under-load-split-from-432.md |
| boundary | whole-issue close |
| milestone | — |
| window | a040b13bab7015c399ff6c9a81c1b4c02d7d7b24..328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd |
| command | sdlc close --issue 712 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T15:16:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are addressed. The cleanup deadline now reaches metadata reads and native deletion, expiry avoids rereading storage, and bounded disposition failure preserves the occupied claim. Composed tests reach their target under review confinement and pass 20/20 under owned load. No blocking findings remain; one minor consolidation opportunity remains.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Contextual metadata reads, cached expiry identity, legacy-index propagation, and bounded disposition persistence are covered by contention regressions. Removing PreviewThreadContext through a temporary Go overlay makes TestSessionCleanupMetadataReadHonorsContextWhileStoreHeld fail after waiting for lock release; the pinned implementation passes.
  - id: BR-2
    disposition: addressed
    note: |
      The original focused command passes within checkout-nested TMPDIR. The pinned-head composed family passes 20/20 with 20 owned CPU workers in 106.456 seconds; all workers were killed and reaped. Removing the explicit Git ceiling makes the nested-checkout regression fail with repository-scope changes.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: single-incarnation-transition-authority
    title: |
      Exact live-to-unknown transition logic is duplicated
    detail: |
      ARCH-DRY: cmd/internal/couchcore/couch.go:960-995 repeats the identity matching, live-to-unknown mutation and revision-conflict loop in threadstore.go:1034. Extract a shared transition helper or contextual store operation while preserving the callers’ different already-unknown behavior.
```

1. **Strengths**

   - Cleanup knowledge defaults to unconfirmed; the pure decision table retains ownership across shapes and phases.
   - Disposition persistence uses one bounded context and preserves the original claim when storage refuses (`couch.go:948`).
   - Public regression coverage checks exact ownership, persisted failure history, diagnostic identity and actual show dispatch.
   - Fixture listing derives from persisted server identity, process generation and endpoint lifetime.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   Consolidate the duplicated incarnation transition described above.

5. **Test coverage notes**

   - Original focused cleanup/composition/receipt command passed.
   - Focused race tests passed.
   - Composed ordinary/fresh-slot family passed 20/20 under 20-worker CPU load.
   - Metadata-read and Git-isolation mutation checks both failed as expected.
   - Existing helper ownership, cleanup and switch-agent regressions passed.
   - Artifact-path, terminal-command and repository lint checks passed; `git diff --check` passed.
   - Temporary overlays were removed; tracked files remain unchanged. The full recorded suite remains a separate landing requirement.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Flag: minor duplicated incarnation transition. |
   | ARCH-PURE | Pass: cleanup decisions remain pure; storage/native effects stay in integration code. |
   | ARCH-PURPOSE | Pass: persistent refusal returns boundedly while retaining ownership. |
   | ARCH-MOCK | Pass: stateful fixture exercises production composition; existing native conformance workflow remains available. |
   | ARCH-CONSTRAINTS | Pass: shared session deadline, separate persistence deadline and diagnostic cap have contention coverage. |
   | ARCH-SECURE | Pass: exact identities guard mutation; uncertainty grants no destructive authority. |
   | ARCH-ORDER | Pass: explicit cleanup knowledge controls disposition; tests cover live, unknown and replaced identities. |
   | ARCH-FUNERAL | Pass: fixture state is removed and owned processes/accept loops are joined. |
   | ARCH-EVOLVE | Pass: existing persisted fields and accepted `launch`/`unproven` values are reused. |

   Planned core entities exist at their stated paths; pure tests require no IO. Atlas documents the new envelope. No new command, flag or configuration surface requires a README addition.

7. **Plan revision recommendations**

   None required. Existing R4 entries accurately describe the implemented deadline and persistence behavior.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a3e2ab98-5f8f-4cce-a744-351173dd134d","repository":"github.com/xianxu/pair","issue_id":"000712","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/2965270817b1d2d8","reviewed_head":"328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd","slot":"pair:3","requested_at":"2026-10-19T22:13:06.041356Z"},"phase":"finished","pid":50639,"started_at":"2026-10-19T22:13:06.238772Z","finished_at":"2026-10-19T22:16:21.529288Z","session_id":"01a15639-bc18-7de2-987d-d0fa17a9335b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.756941s
head: "328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
