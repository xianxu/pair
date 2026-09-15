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

---

## Re-review — 2026-09-14T22:53:48-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..1f2f487ebc7305457e43dfcbe19280ac3aa31c88 |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-14T22:53:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Four prior findings are addressed. BR-4 and BR-6 remain open, and interrupted journal publication leaves an additional uncollected artifact. Existing tests passed across 11 affected packages; four scratch-copy regressions reproduced the remaining gaps. The reviewed worktree is unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays covers capture cleanup followed by metadata expiry. Restoring empty-session rejection makes its metadata-retirement assertion fail.
  - id: BR-2
    disposition: addressed
    note: |
      Central staging and coordinated cleanup handle unpublished JSON. TestInterruptedMetadataPublisherProcess exercises killed publishers with complete and partial writes; restoring destination-local staging makes the regression fail.
  - id: BR-3
    disposition: addressed
    note: |
      Recovery and admission reclaim confirmed-dead pre-spawn reservations while preserving live, unknown and spawned evidence. Disabling recovery makes TestRecoverDeadUnspawnedStartsPreservesUnknownAndSpawned fail.
  - id: BR-4
    disposition: not-addressed
    note: |
      storagegc/inventory.go:175 only merges known owners into physically discovered namespaces. A legacy Couch archive without its Pair repos/<scope> directory is omitted from Apply, so collector.go:475 never onboards it. TestReviewLegacyArchiveWithoutPairScope reproduces this on the pinned head.
  - id: BR-5
    disposition: addressed
    note: |
      Production phase advancement calls ReduceTransaction through the persistence adapter. The phase/event matrix, sequence tests and bypass guard pass; a regressive retirement transition makes the matrix fail. The revised plan names the implemented phases and symbols.
  - id: BR-6
    disposition: not-addressed
    note: |
      gcruntime/schedule.go:64 still invokes a full owner preview for every diagnostic page: a limit-2 regression probes all 8 owners. Recovery also reaches blocking Couch flock through retention.go:392, ignoring an expired maintenance context while holding the root lock. Both scratch regressions fail.
findings:
  - id: new
    severity: Important
    family: interrupted-publication-recovery
    title: |
      Unpublished quarantine directories have no recovery path
    detail: |
      storagegc/transaction.go:208 creates the unique quarantine directory before publishing its journal at line 222. Publication failure or cancellation leaves it unreachable by journal-only recovery at line 586; three injected publication failures leave three directories. This is the 2nd finding in family interrupted-publication-recovery. Define and enforce recovery for every artifact created before authoritative publication, rather than fixing this instance alone.
```

## 1. Strengths

- Exact ownership, identity checks and quarantine replay protect replacement files and newer owner incarnations.
- Metadata-only retirement now uses the transaction lifecycle, with protection and interruption tests.
- Startup recovery distinguishes confirmed pre-spawn death from uncertain child effects.
- README and atlas updates cover retention buckets, migration, managed access and recovery. The plan explicitly acknowledges the combined M1/M2 implementation boundary.

## 2. Critical findings

**BR-4 — Archive onboarding remains unreachable for absent Pair namespaces.**
At `cmd/internal/storagegc/inventory.go:175–195`, known owners only enter namespaces discovered on disk. Registered Couch archives can exist without corresponding Pair payload directories; they disappear from reports and never receive grace.

**Fix:** Build the namespace inventory from both physical entries and validated known owners. Cover legacy archives, tracked archives and metadata-only scoped owners with absent payload directories. Assert onboarding, full grace and eventual collection through `Collector.Apply`. **ARCH-PURPOSE, ARCH-FUNERAL.**

## 3. Important findings

**BR-6 — Maintenance bounds are incomplete.**

- `cmd/internal/gcruntime/schedule.go:64`: diagnostic pages perform full owner evaluation. Reproduction: **8 owner probes with limit 2**.
- `cmd/internal/couchcore/retention.go:392`: recovery reaches blocking `flock` through `withRetentionWrite` and `withStoreLock`. Reproduction: a **20 ms deadline remained blocked after 150 ms**, until the test released the Couch lock.

**Fix:** Apply one maintenance-budget rule across session discovery, diagnostic discovery and reference recovery. Avoid full owner evaluation per diagnostic page; make nested maintenance lock acquisition nonblocking or cancellation-aware. Add production-path tests for both cases. **ARCH-CONSTRAINTS, ARCH-ORDER.**

**New — Quarantine creation precedes its recovery authority.**
`cmd/internal/storagegc/transaction.go:208–223` leaves unique empty directories after failed publication. Recovery only enumerates transaction journals.

**Fix:** Make pre-publication artifacts recoverable across errors, cancellation and process death. Journal-first creation with recovery support, or coordinated orphan reconciliation, must cover the whole publication lifecycle. **ARCH-FUNERAL, ARCH-ORDER.**

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed: storagegc, gcruntime, gccmd, artifactpath, couchcore, diagnosticlog, retentioncmd, launcher, opener, scrollbackcmd and pairlog.
- Mutation checks made the BR-1, BR-2, BR-3 and BR-5 regressions fail.
- Four additional regressions failed against restored, unmodified production sources in a scratch copy: missing-scope archive, diagnostic owner budget, blocked Couch lock and orphan quarantine.
- Lua/shell suites were not rerun in this review.

Reproduction files remain in [/tmp/pair239-review.YxlFkq](/tmp/pair239-review.YxlFkq).

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared ownership, policy and coordination mechanisms. |
| ARCH-PURE | Pass — deterministic policy/reducer tests; filesystem effects remain in adapters. |
| ARCH-PURPOSE | Flag — BR-4 omits valid archive owners. |
| ARCH-MOCK | Pass — portable stores, stateful process doubles and subprocess conformance tests. |
| ARCH-CONSTRAINTS | Flag — BR-6 leaves full diagnostic scans and blocking nested acquisition. |
| ARCH-SECURE | Pass — inspected paths reject malformed evidence and unsafe identities conservatively. |
| ARCH-ORDER | Flag — maintenance cancellation and pre-publication lifecycle remain incomplete. |
| ARCH-FUNERAL | Flag — omitted archives and orphan quarantine directories lack effective collection. |

## 7. Plan revision recommendations

Add `## Revisions` entries specifying:

- **Owner discovery:** registered references and retention metadata establish owners even without payload directories.
- **Maintenance envelope:** budgets and cancellation apply across every phase and nested lock.
- **Publication recovery:** enumerate artifacts created before journal publication and define their recovery/removal paths.

---

## Re-review — 2026-09-14T23:22:52-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..4bff5da772f6bd9efe1c3c9f7f72c3d043ddc37c |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-14T23:22:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Archive onboarding and journal-before-quarantine publication now have substantive regression coverage. However, BR-6 and BR-7 remain incomplete at their broader boundaries, and diagnostic deletion has a newly reproduced recovery failure. The six reviewed package suites pass; three additional scratch-copy regressions expose these gaps. The repository remains unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Eligible metadata-only retirement is implemented and covered by protection, empty-admission, and interrupted-retirement tests in storagegc/transaction_test.go; the package suite passes.
  - id: BR-2
    disposition: addressed
    note: |
      Coordinated retention JSON stages in .retention/pending; stateio_test.go covers killed publishers, bounded cleanup, and legacy pending files. These tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      start_test.go covers confirmed-dead pre-spawn reclamation, admission-cap recovery, actual parent death, and preservation of spawned or unknown evidence; the package suite passes.
  - id: BR-4
    disposition: addressed
    note: |
      OnboardArchiveGrace journals missing clocks against exact archive bytes. TestApplyRetainsAndCollectsOwnersWithoutPairNamespace exercises legacy archives without Pair directories, read-only preview, full grace, and eventual collection; malformed-clock and replay-identity tests also pass.
  - id: BR-5
    disposition: addressed
    note: |
      The revised Core concepts table names the actual ReduceTransaction implementation; production phase advancement uses it, with phase/event, sequence, and AST bypass tests passing.
  - id: BR-6
    disposition: not-addressed
    note: |
      Owner paging and nonblocking locks are covered, but diagnosticlog/proof.go:25-27 provides no maintenance context to inspections, and line 77 creates an independent background deadline. Cancellation during OpenFiles still starts subsequent Runtimes work under the shared lock. A scratch regression observed two runtime inspections after cancellation. ARCH-CONSTRAINTS: complete the bounded-maintenance-work rule across nested inspections and traversal helpers.
  - id: BR-7
    disposition: not-addressed
    note: |
      Journal-before-quarantine ordering is fixed, but diagnostic registry publication still uses unrecoverable .pending-* files via diagnosticlog/registry.go:38 and writer.go:470. Enumeration filters these entries while gcruntime/runtime.go:112 treats the filtered count as completion. A scratch fixture discovered only 53 of 101 registered paths. ARCH-PURPOSE/ARCH-FUNERAL: the interrupted-publication-recovery family remains incomplete.
findings:
  - id: new
    severity: Critical
    family: durable-deletion-replay
    title: |
      Diagnostic deletion cannot recover after removing its parent directories
    detail: |
      diagnosticlog/collect.go:199 removes empty segment ancestors before clearing the durable Deleting intent at lines 205-209. Death or cancellation between those effects leaves replay calling syncDir on a missing parent at line 183, permanently failing. A scratch regression reproduces ENOENT. Make replay tolerate already-completed directory cleanup while preserving identity checks, and test interruption after each parent removal and before intent retirement (ARCH-ORDER, ARCH-FUNERAL).
```

## 1. Strengths

- Legacy archive onboarding preserves malformed evidence and grants the full grace period without creating missing Pair payload directories.
- Quarantine publication tests kill actual processes before and after journal publication.
- Pure transaction tests enforce legal transitions and reject production phase assignments outside the reducer.
- README and atlas updates document commands, migration, retention buckets, and managed I/O.

## 2. Critical findings

**Diagnostic deletion replay:** `cmd/internal/diagnosticlog/collect.go:183–209`. A completed directory removal becomes a permanent replay error. Preserve recoverability through intent retirement; add segment-specific failure and cancellation tests.

## 3. Important findings

- **BR-6 — nested maintenance cancellation:** `cmd/internal/diagnosticlog/proof.go:25–77`. Propagate the worker context through `Proof`, `Inspection`, and subprocess execution; check it between inspections and filesystem traversal steps. Existing cancellation tests stop around the proof callback, not inside its production work. This continues the second finding in family `bounded-maintenance-work`; fix the entire nested-effect rule.
- **BR-7 — diagnostic registry staging:** `cmd/internal/diagnosticlog/registry.go:38,75–98` and `cmd/internal/gcruntime/runtime.go:112`. Interrupted registry writes have no cleanup owner and can truncate discovery. This is the third finding in family `interrupted-publication-recovery`. Enumerate every publication destination and its recovery authority, including concurrent diagnostic publishers; do not sweep their active temporary files indiscriminately. Return explicit pagination completion rather than deriving it from filtered results.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- `storagegc`, `artifactpath`, `gcruntime`
- `diagnosticlog`, `gccmd`, `couchcore`

Scratch-copy regressions reproduced:

- `ENOENT` replay after diagnostic parent cleanup.
- Two runtime inspections starting after cancellation.
- Premature registry completion after 53 of 101 paths.

Scratch tests are in [review_recovery_test.go](/tmp/pair239-review-3fzlf19g/cmd/internal/diagnosticlog/review_recovery_test.go). Full-tree, Lua, and hosted conformance checks were not rerun.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared ownership classification and retention APIs. |
| ARCH-PURE | Pass: listed pure entities and transaction decisions remain free of I/O. |
| ARCH-PURPOSE | Flag: publication recovery has not covered every publisher. |
| ARCH-MOCK | Pass: portable stores, process doubles, and real-process conformance tests exist. |
| ARCH-CONSTRAINTS | Flag: nested inspection work escapes maintenance cancellation. |
| ARCH-SECURE | Pass for inspected boundaries: exact identities, malformed-evidence retention, and unsafe-path tests. |
| ARCH-ORDER | Flag: diagnostic cleanup cannot replay every partially completed sequence. |
| ARCH-FUNERAL | Flag: registry staging lacks removal authority; interrupted diagnostic cleanup can remain stuck. |

## 7. Plan revision recommendations

Add `## Revisions` entries that:

- Enumerate publication destinations, synchronization owners, and interrupted-stage recovery.
- Extend the maintenance envelope through nested inspections and traversal helpers.
- Specify diagnostic deletion recovery after payload, metadata, and ancestor removal, including cancellation before intent retirement.

---

## Re-review — 2026-09-14T23:50:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..44eb1f10e342bff16c8a9f24b67e55fb9f64eb07 |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-14T23:50:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The specific BR-6/BR-7/BR-8 corrections pass regression tests, and scratch mutations confirm those tests detect their removal. One related correctness gap blocks shipping: interrupted diagnostic appends leave valid payloads permanently incompatible with their metadata, disabling subsequent logging and collection. The repository remained unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Metadata-only admission, protection and interrupted-retirement tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Central pending-metadata recovery and killed-publisher tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      Dead pre-spawn reservation recovery and uncertain-start protection tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      Legacy archive onboarding and namespace discovery tests pass, including missing payload directories.
  - id: BR-5
    disposition: addressed
    note: |
      The plan names the production reducer; phase/event, sequence and production bypass-guard tests pass.
  - id: BR-6
    disposition: addressed
    note: |
      Production owner-budget, 100,000-filename, contention and cancellation tests pass. Removing subprocess deadline propagation makes both lsof and ps deadline regressions fail.
  - id: BR-7
    disposition: addressed
    note: |
      Journals now precede unique quarantines. Restoring pre-publication directory creation makes all three publication-failure regressions fail. A separate remaining publication-class instance is reported below.
  - id: BR-8
    disposition: addressed
    note: |
      Parent-removal and intent-retirement replay tests pass. Replacing surviving-ancestor synchronization with direct parent synchronization reproduces ENOENT.
findings:
  - id: new
    severity: Critical
    family: interrupted-publication-recovery
    title: |
      Interrupted diagnostic appends permanently block logging and collection
    detail: |
      cmd/internal/diagnosticlog/writer.go:281 changes the payload before publishing Size/ModTime at line 294. Cancellation, publication failure or process death between those effects leaves metadata stale; Open, Write, Maintain and Collect subsequently reject the generation. A scratch regression confirms persisted bytes followed by failures from both reopen and expired collection. This is the 3rd finding in family interrupted-publication-recovery. State and enforce the class-wide rule: every payload effect requiring matching metadata must have recoverable authority before mutation. Sweep initial/current-file creation, append, rotation, deletion and retirement; add interruption tests without weakening replacement-file checks (ARCH-ORDER, ARCH-FUNERAL, ARCH-PURPOSE).
```

### 1. Strengths

- Exact ownership and retention policies have exhaustive family checks and conservative protection precedence.
- Collection phases use a production reducer with sequence tests and a bypass guard.
- Recovery tests exercise actual killed publishers, cancellation and unsafe replacements.
- README and atlas cover commands, migration, retention clocks and the cooperative scheduling budget.

### 2. Critical findings

**Diagnostic append recovery:** [writer.go:281](cmd/internal/diagnosticlog/writer.go#L281) writes bytes before publishing matching metadata at line 294.

The scratch test interrupts publication after a successful payload write. Both subsequent `Open` and collection after eight days return:

> diagnostic generation changed outside protocol

Add a replayable append protocol, preserving strict rejection of unrelated modifications. The reproducer is available through [this scratch overlay](/tmp/pair239-review-b7ujg5tu/append.json):

```sh
go test -overlay /tmp/pair239-review-b7ujg5tu/append.json \
  ./cmd/internal/diagnosticlog \
  -run '^TestReviewAppendPublicationInterruption$' -count=1 -v
```

### 3. Important findings

None additional.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed full suites for storagegc, diagnosticlog, gcruntime, gccmd, artifactpath, opener and retentioncmd; targeted retention/publication suites for Couch, launcher, pairlog and scrollback also passed.

Existing publication tests verify staging cleanup but miss the ordinary append’s payload-to-metadata failure window. This review did not rerun the full repository, race or Lua suites.

### 6. Architectural notes

| Marker | Result |
|---|---|
| ARCH-DRY | Pass — shared ownership, coordination and diagnostic implementations. |
| ARCH-PURE | Pass — core policy/reducer tests exercise deterministic logic directly. |
| ARCH-PURPOSE | Flag — interrupted appends strand diagnostic data outside successful retention. |
| ARCH-MOCK | Pass — portable stores, injected process evidence and live conformance tests. |
| ARCH-CONSTRAINTS | Pass — tested owner budgets, continuation, contention and cooperative cancellation. |
| ARCH-SECURE | Pass — inspected paths reject unsafe identities and uncertain evidence. |
| ARCH-ORDER | Flag — append effects lack recoverable intermediate authority. |
| ARCH-FUNERAL | Flag — the resulting generation has no successful automatic retirement path. |

### 7. Plan revision recommendations

Add a `## Revisions` entry defining recovery across **every diagnostic payload/metadata boundary**, including ordinary append. Update the checked “crashes at each durable boundary” claim at plan line 243 after the missing regressions pass.

---

## Re-review — 2026-09-15T00:13:33-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..84e00c86a34069d53715510b92dc2c88cb7529ea |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-15T00:13:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

BR-9 is addressed: interrupted diagnostic appends and creation now have recoverable authority, with meaningful regression coverage. Focused tests and diagnostic race tests passed. One residual BR-5 issue blocks this gate under the supplied Core concepts rule: `StoreRegistry` remains classified PURE despite filesystem-dependent validation. No new runtime correctness defect was demonstrated.

## 1. Strengths

- Append recovery preserves partial-write counts and reconciles only bytes actually written; it rejects replacement inodes, truncation and foreign tails (`diagnosticlog/append.go:89`).
- Creation publishes an identified staged inode without replacing an existing file (`diagnosticlog/creation.go:41`).
- Production collection advances through `ReduceTransaction`, backed by phase/event, sequence and bypass tests (`storagegc/transaction.go:709`).
- README and atlas document migration, retention clocks, recovery and the cooperative maintenance budget.

## 2. Critical findings

**BR-5 — not-addressed: remaining PURE classification contradiction.**

`workshop/plans/000239-storage-gc-plan.md:33` classifies `StoreRegistry` as PURE. Its method `StoreRegistry.validate` calls `canonicalStore` (`cmd/internal/storagegc/stores.go:54`), which resolves symlinks, opens directories and reads entries (`stores.go:30`, `stores.go:39`). The same registry value can therefore validate differently as external filesystem state changes. Registry tests appropriately use temporary directories and actual filesystem changes.

**Fix:** classify this entity and its validation as INTEGRATION. Alternatively, separate pure schema validation from filesystem availability checks. Record the correction under `## Revisions`.

This reopens the classification portion of BR-5; its reducer correction remains validated. **ARCH-PURE.** Critical severity follows the explicit Core concepts gate, rather than a demonstrated data-loss defect.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- Full package tests: `diagnosticlog`, `storagegc`, `artifactpath`, `gccmd`, `retentioncmd`.
- Diagnostic race suite.
- Targeted Couch/runtime retention, archive, publication, registry and maintenance tests, including the 100,000-filename fixture.
- Pinned-range `git diff --check`.

A temporary Go overlay removed append-intent publication. `TestAppendCanceledAfterPayloadRetainsRecoverableIntent` then failed on reopen with `diagnostic generation changed outside protocol`; the unmodified test passed. This independently establishes BR-9 regression sensitivity.

Repository files remained unchanged. Full-tree, Lua and hosted conformance suites were not rerun in this review.

## 6. Architectural notes

| Marker | Result |
|---|---|
| ARCH-DRY | Pass — shared ownership, coordination and publication helpers. |
| ARCH-PURE | **Flag — BR-5 classification mismatch above.** Other listed pure entities and reducer logic match their classifications. |
| ARCH-PURPOSE | Pass — combined M1/M2 scope is explicitly reconciled; BR-9 covers creation and append alongside existing rotation/deletion recovery. |
| ARCH-MOCK | Pass — portable stores, injected process evidence, subprocess tests and scheduled Zellij conformance. |
| ARCH-CONSTRAINTS | Pass — bounded owner effects, cancellation and contention tests; cooperative limits are documented. |
| ARCH-SECURE | Pass — malformed metadata and replacement evidence fail closed in inspected paths. |
| ARCH-ORDER | Pass — production reducer use and interruption/replay tests establish ordering. |
| ARCH-FUNERAL | Pass — inspected intents, stages and quarantines have recovery and retirement paths. |

## 7. Plan revision recommendations

Add a `## Revisions` entry correcting `StoreRegistry` to INTEGRATION because validation reads external filesystem state. Apply the classification rule to entity methods as well as fields. A documentation-only correction needs source inspection, not a wording-presence test.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Metadata-only admission, protection and interrupted-retirement tests pass in storagegc.
  - id: BR-2
    disposition: addressed
    note: |
      Coordinated pending publication recovery and killed-metadata-publisher tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      Dead pre-spawn reservation recovery, admission cleanup and uncertain-start protection tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      Durable references establish missing payload namespaces; legacy archive grace and identity-checked onboarding tests pass.
  - id: BR-5
    disposition: not-addressed
    note: |
      The reducer and its production enforcement are corrected, but workshop/plans/000239-storage-gc-plan.md:33 still labels StoreRegistry PURE. Its validate method at cmd/internal/storagegc/stores.go:54 calls filesystem-dependent canonicalStore, including EvalSymlinks and directory reads. Reclassify it as INTEGRATION and record the correction under Revisions (ARCH-PURE); no wording-presence test is required.
  - id: BR-6
    disposition: addressed
    note: |
      Owner-budget, 100,000-filename, diagnostic-page isolation, contention and cancellation tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Prepared journal authority precedes quarantine creation; publication failure, killed-process and unsafe-quarantine recovery tests pass.
  - id: BR-8
    disposition: addressed
    note: |
      Diagnostic deletion replay tests pass across payload, metadata and ancestor removal, including replacement refusal.
  - id: BR-9
    disposition: addressed
    note: |
      Bounded append and exact-inode creation intents precede payload effects and recover through production entrypoints. Partial-write, cancellation, killed-process and replacement tests pass; removing append-intent publication in a scratch overlay makes the cancellation/reopen regression fail.
```

---

## Re-review — 2026-09-15T00:20:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..d7180a3822262ce5c588afce1b061b9406744b6c |
| command | sdlc milestone-close --issue 239 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-15T00:20:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range satisfies the reviewed retention contracts. BR-5 is addressed: production transitions use the pure reducer, regression tests reject illegal transitions, and the plan correctly classifies filesystem-dependent registry validation as INTEGRATION. No new blocking findings emerged. Required range inspections succeeded; the working tree is clean.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Eligible metadata-only retirement uses the collection journal; protection and interrupted-retirement tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Reserved metadata publication stages have coordinated recovery; subprocess interruption and bounded cleanup tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      Verified dead pre-spawn reservations retire; uncertain spawned reservations retain protection. Recovery and resolution tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      OnboardArchiveGrace grants missing legacy clocks fresh grace while preserving malformed evidence; focused Couch tests pass.
  - id: BR-5
    disposition: addressed
    note: |
      transaction.go:450,494 route transitions through advanceTransaction and ReduceTransaction. Matrix, sequence and bypass-guard tests pass; a scratch regression permitting finalized-to-detached fails both behavioral tests. Plan line 33 now classifies StoreRegistry as INTEGRATION, matching stores.go:27-66 filesystem validation.
  - id: BR-6
    disposition: addressed
    note: |
      Scheduled pages bound visited-owner work; the 100,000-filename, contention, cancellation and diagnostic-isolation tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Journal publication precedes unique quarantine creation; publication-failure and killed-publisher recovery tests pass.
  - id: BR-8
    disposition: addressed
    note: |
      Diagnostic deletion replay handles removed ancestor directories while rejecting replacement identities; replay tests pass.
  - id: BR-9
    disposition: addressed
    note: |
      Bounded append intents reconcile the observed authorized prefix; partial-write, killed-publisher and substitution tests pass.
```

### 1. Strengths

- Transaction authority remains fixed across phase changes; recovery tests protect replacement source files and newer incarnations.
- Preview and apply have separate mutation boundaries, verified by portable-store and real CLI tests.
- README and atlas document migration, retention clocks, recovery, and the absence of a global disk ceiling.
- Managed-use documentation links entrypoints to actual protection mechanisms and behavioral tests.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Full `storagegc`, `artifactpath`, `diagnosticlog`, `gcruntime`, and `gccmd` suites.
- Focused Couch retention, legacy-archive, and archive-detachment tests.
- `tests/retention-test.sh`.
- Pinned-range `git diff --check`.

Reducer mutation testing produced the expected failures. Full-tree, race, and hosted Zellij checks were not rerun in this review.

### 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — artifact constructors and shared coordination APIs supply common authority. |
| ARCH-PURE | Pass — policy and reducer tests execute without IO; registry validation is correctly classified. |
| ARCH-PURPOSE | Pass — session, capture, diagnostic, and Couch retention paths are represented. |
| ARCH-MOCK | Pass — portable stores, stateful process fakes, and native conformance tests exercise the boundaries. |
| ARCH-CONSTRAINTS | Pass — owner limits, cancellation, contention, and large inventories have direct tests. |
| ARCH-SECURE | Pass — inspected recovery paths validate identities and retain uncertain evidence. |
| ARCH-ORDER | Pass — production phase changes use the reducer; illegal and regressive transitions are tested. |
| ARCH-FUNERAL | Pass — inspected metadata, quarantine, receipt, and diagnostic families have cleanup paths or explicit bounds. |

### 7. Plan revision recommendations

None required.
