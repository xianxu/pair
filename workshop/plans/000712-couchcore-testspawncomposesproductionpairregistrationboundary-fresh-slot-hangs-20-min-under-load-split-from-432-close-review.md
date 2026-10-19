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
