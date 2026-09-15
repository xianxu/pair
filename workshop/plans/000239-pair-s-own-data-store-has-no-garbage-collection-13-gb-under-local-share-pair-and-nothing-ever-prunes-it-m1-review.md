# Boundary Review — 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it#239 (milestone M1)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..8f2401f4822a6262653120a546695cdf74902586 |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-14T22:22:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range includes both milestones, as the plan’s revision acknowledges. Exact ownership, conservative liveness checks, and quarantine recovery have substantial implementation and test coverage. However, normal lifecycle and crash scenarios can permanently stop collection, legacy archives never enter their promised grace period, and the implementation lacks the promised pure transaction model.

## 1. Strengths

- Exact artifact matching rejects ambiguous owners and uses canonical constructors; manifest coverage tests pass.
- Process protection distinguishes alive, dead, and unknown identities, with stateful fakes and subprocess conformance tests.
- Quarantine recovery checks file identities and protects replacement source incarnations.
- README and atlas updates cover the new commands, retention buckets, migration, and managed I/O boundaries.

## 2. Critical findings

1. **Expired metadata-only owners stop collection.**  
   `cmd/internal/storagegc/collector.go:228` emits a session item whenever activity exists, even without session members or archives. Once eligible, `transaction.go:193` rejects it as an “empty collection,” aborting apply and the scheduled worker. This occurs naturally after independent capture cleanup; `TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays` stops before the resulting activity record expires.  
   **Fix:** support coordinated retirement of metadata-only owners without losing outstanding capture/process protection. Test subsequent sweeps beyond sixty days. **ARCH-FUNERAL, ARCH-PURPOSE.**

2. **Interrupted metadata publication can permanently block the root.**  
   `cmd/internal/storagegc/stateio.go:41` creates `.pending-*` files and relies on deferred removal. Process death before rename leaves them behind. `collector.go:103` treats every owner-directory entry as authoritative metadata: partial temporary files fail decoding; complete ones fail filename validation. Transaction-directory leftovers similarly obstruct recovery and managed access.  
   **Fix:** separate unpublished temporary files from authoritative records and recover their residue under coordination. Test actual interruption before publication across the metadata directories. **ARCH-ORDER, ARCH-FUNERAL.**

3. **Abandoned startup reservations have no recovery path.**  
   `cmd/internal/storagegc/use.go:60` recovers intents and process registrations but never `Starts`. A launcher dying after `ReserveStart` and before `MarkStartSpawned` leaves a reservation permanently blocking collection, despite provable absence of child effects. Repetition eventually reaches the 32-reservation launch failure at `start.go:50`. Spawned reservations lacking an acknowledgment also have no reconciliation mechanism.  
   **Fix:** reconcile dead pre-spawn reservations automatically; implement evidence-based reconciliation or explicit resolution for uncertain spawned reservations. Preserve live/unknown children. **ARCH-ORDER, ARCH-FUNERAL.**

4. **Pre-upgrade Couch archives never receive onboarding grace.**  
   The base archive writer created no grace sidecar. At head, `cmd/internal/couchcore/retention.go:78` treats its absence as an error, which becomes permanent `ClockError` evidence at line 176. Apply initializes Pair activity only; no path initializes these archive clocks. Consequently, legacy archives and associated session data remain blocked indefinitely after migration.  
   **Fix:** initialize missing legacy archive grace through Couch’s coordinated journal, preserving exact archive identity and granting the full sixty days. Keep malformed existing clocks blocked. **ARCH-PURPOSE, ARCH-FUNERAL.**

5. **The promised pure transaction model is absent.**  
   `workshop/plans/000239-storage-gc-plan.md:39` promises pure transition tests; line 133 marks `ReduceTransaction` tests complete. No `ReduceTransaction` exists. Production directly mutates exported string state inside filesystem code (`cmd/internal/storagegc/transaction.go:403`), and the transaction tests require filesystem I/O. Actual phases also differ from the documented enumeration.  
   **Fix:** introduce and enforce the pure state/event transition function, with independently stated invariants and sequence tests. Reconcile the Core concepts tables, including explicit kinds and the nonexistent `ManagedUse`/`GCCLI` entity names. **ARCH-PURE, ARCH-ORDER.**

## 3. Important findings

6. **Automatic work budgets do not bound actual owner processing.**  
   `cmd/internal/storagegc/collector.go:342` holds the shared coordinator across two complete snapshots and initialization/recovery of every owner. Each recovery unconditionally persists state. The limit only counts deletions; `gcruntime/schedule.go:58` advances a round counter rather than an owner cursor. No two-second scheduling budget is applied, and transaction effects do not check cancellation between steps. This can block managed writes and Couch operations behind optional maintenance.  
   **Fix:** budget discovery, recovery, and initialization as well as deletion; persist actual continuation progress, use nonblocking automatic lock acquisition, and check cancellation between effects. Test the production batch with oversized inventories and counted operations. **ARCH-CONSTRAINTS.**

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed thirteen affected package suites, including storagegc, artifactpath, diagnosticlog, gcruntime, gccmd, retentioncmd, launcher, couchcore, opener, scrollbackcmd, pairlog, wrapcmd, and couchcmd.
- Pinned-range `git diff --check` passed.
- Existing fault tests simulate errors after selected durable boundaries; they do not cover process death during temporary-file publication.
- The 100,000-name test exercises the matcher, not production scheduler work bounds.
- Full-tree, race, and Lua suites were not rerun during this review. No files were edited.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | **Pass:** shared constructors and diagnostic writer reuse. |
| ARCH-PURE | **Flag:** transaction decisions remain mixed with I/O. |
| ARCH-PURPOSE | **Flag:** legacy archives and metadata-only owners cannot complete retention. |
| ARCH-MOCK | **Pass:** portable stores, injectable process evidence, and local conformance tests. |
| ARCH-CONSTRAINTS | **Flag:** production batches exceed their declared work envelope. |
| ARCH-SECURE | **Pass:** inspected decoders and deletion paths conservatively reject uncertain authority. |
| ARCH-ORDER | **Flag:** missing transition enforcement and incomplete crash/start recovery. |
| ARCH-FUNERAL | **Flag:** temporary files, abandoned starts, and orphan activity lack effective endings. |

## 7. Plan revision recommendations

Add `## Revisions` entries that:

- Reconcile actual entity names, PURE/INTEGRATION kinds, transaction phases, and transition enforcement.
- Specify metadata-only retirement and temporary-publication recovery.
- Enumerate startup reconciliation outcomes, including confirmed pre-spawn death.
- Define legacy archive onboarding and its regression evidence.
- Replace the claimed scheduler completion with enforceable processing budgets and production-path acceptance tests.

```findings
findings:
  - id: new
    severity: Critical
    family: metadata-only-retirement
    title: |
      Expired metadata-only owners abort collection
    detail: |
      collector.go:228 emits eligible empty session groups, while transaction.go:193 rejects them. Add coordinated metadata-only retirement and a regression covering capture cleanup followed by sixty-day expiry (ARCH-FUNERAL, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: interrupted-publication-recovery
    title: |
      Crashed metadata writes leave temporary files that block recovery
    detail: |
      storagegc/stateio.go:41 relies on deferred temporary-file removal, but collector.go:103 and transaction recovery interpret leftovers as authoritative records. Recover unpublished residue under coordination and test process death before rename (ARCH-ORDER, ARCH-FUNERAL).
  - id: new
    severity: Critical
    family: startup-reservation-reconciliation
    title: |
      Abandoned startup reservations never retire
    detail: |
      storagegc/use.go:60 never reconciles Starts; even confirmed pre-spawn parent death permanently blocks collection and eventually exhausts start.go:50's reservation cap. Add evidence-based recovery with live and unknown child protection (ARCH-ORDER, ARCH-FUNERAL).
  - id: new
    severity: Critical
    family: legacy-retention-onboarding
    title: |
      Pre-upgrade Couch archives cannot enter retention grace
    detail: |
      couchcore/retention.go:78 turns absent legacy grace into permanent ClockError evidence, and apply never initializes archive clocks. Journal a full onboarding grace for missing legacy clocks while retaining malformed evidence (ARCH-PURPOSE, ARCH-FUNERAL).
  - id: new
    severity: Critical
    family: enforced-pure-transitions
    title: |
      The completed plan claims a transaction reducer that does not exist
    detail: |
      The plan at lines 39 and 133 promises pure transition coverage, but transaction.go:403 directly mutates phases inside I/O. Implement the enforced pure state/event model and reconcile the Core concepts table and phase enumeration (ARCH-PURE, ARCH-ORDER).
  - id: new
    severity: Important
    family: bounded-maintenance-work
    title: |
      Scheduled collection limits deletions but processes every owner under the shared lock
    detail: |
      collector.go:342 performs full snapshots and owner rewrites regardless of batch limit; gcruntime/schedule.go:58 has no owner continuation cursor. Enforce the declared scheduling budget, nonblocking acquisition, and cancellation between effects with production-batch tests (ARCH-CONSTRAINTS).
```
