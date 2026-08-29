# Boundary Review — pair#156 (whole-issue close)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..a13897ec17b07d52043f2b03181bd0ea9a948c5e |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T13:33:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation has strong low-level primitives and all executed suites pass, but several central Spec requirements are unreachable or test-only. Most importantly, proofless migration is never invoked, launch-boundary offsets are recorded but ignored, stale catalog writers can overwrite newer state, and the promised catalog-backed production façade does not exist. These are blocking ARCH-PURPOSE correctness gaps.

### 1. Strengths

- `CatalogStore` has a clean locked/atomic publication boundary and tests write, sync, rename, directory-sync, and unlock outcomes ([catalog_store.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/catalog_store.go:61)).
- Platform metadata uses filesystem APIs rather than a per-file `stat` subprocess.
- Scanner validation consumes records through a stable observed EOF and rejects replacement/truncation ([incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:295)).
- Alt+X confirmation is isolated into a small local-only module and tested before subprocess execution ([confirm_quit.lua](/Users/xianxu/workspace/pair/nvim/confirm_quit.lua:18)).
- Authorization-proof parsing is strict, rejects duplicate artifacts, and validates complete offsets ([record.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:400)).

### 2. Critical findings

- [proof_migration.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/proof_migration.go:39): `ProofMigrator.Request` has no production caller. [query.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query.go:96) only reports that migration is pending, permanently leaving legacy bindings unavailable. Wire targeted validation and proof publication into the owner lookup lifecycle, with an integration test that fails if no durable proof appears.

- [target.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/target.go:15): `TargetArtifactBoundary` discards stable ID, generation, mutation, and `RawSize`; [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:204) copies only path fields. No production code uses `RawSize` or reads `B-1`, so the specified newline/incomplete-record launch-boundary framing is absent. Add the boundary algorithm and positive newline/incomplete/truncation/replacement watcher tests.

- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:181): catalog merging unconditionally overwrites an existing same-key entry with the caller’s possibly stale state. Two serialized writers can therefore publish a lower cursor or erase a newer disputed transition. Recompute/merge monotonically against the locked current entry and test same-key writers completing newest-first and oldest-last.

- [incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:29): the plan’s `IncrementalInventory` façade does not exist, and `ReconcileCatalog` has no production caller. Launch, watcher, and query each implement parallel metadata/proof logic; the performance test invokes reconciliation manually. Route production consumers through the catalog/reconciliation seam, eliminating the duplicated continuity policy. This violates ARCH-DRY and ARCH-PURPOSE and contradicts the Core concepts table.

- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:101): an upgraded watcher handling a v1 launch still performs whole-corpus inventory and event scans every poll. [shadow_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/shadow_test.go:22) explicitly allowlists these interactive calls despite the plan limiting exceptions to diagnostic/conformance entry points. Replace this compatibility path with bounded migration or fail it closed without scanning.

### 3. Important findings

- [provider_live_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/provider_live_test.go:36): the live “conformance” check accepts one recognizable installed sample; it neither compares normalized behavior with `FakeRuntime` nor observes append behavior. Connect live observations to the stateful fake’s modeled transitions and fail on behavioral divergence. ARCH-MOCK remains incomplete.

- [000156 plan](/Users/xianxu/workspace/pair/workshop/plans/000156-incremental-session-inventory-plan.md:169): all eleven Task 2 steps remain unchecked, although the issue claims full completion. Resolve the checklist honestly after the implementation gaps are fixed.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed at pinned HEAD:

- `go test -p 20 ./... -count=1`
- Focused `go test -race -p 20 ...`
- `go vet -p 20 ./...`
- `make test-lua`
- Relevant terminal and watcher shell suites
- `make test-session-inventory-conformance`
- `zellij --config-dir zellij setup --check`
- `git diff --check`

The missing regressions correspond directly to the critical findings: reachable proof migration, positive launch-boundary framing, stale same-key catalog writers, production warm-catalog reuse, and removal of the v1 watcher scan allowlist.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** continuity and selection policy is repeated instead of flowing through `ReconcileCatalog`.
- **ARCH-PURE — pass:** reconciliation, selection, scanner transitions, and metadata merge remain pure; IO is behind injected runtimes.
- **ARCH-PURPOSE — flag:** the persistent catalog, proofless migration, and launch-boundary contracts are not fully delivered.
- **ARCH-MOCK — flag:** the stateful fake is useful, but the promised live behavioral comparison is missing.

### 7. Plan revision recommendations

Append a dated `## Revisions` entry recording:

- the missing production `IncrementalInventory` façade and corrective consumer routing;
- proofless migration wiring and durable publication;
- implementation of raw launch-boundary framing;
- monotonic same-key catalog publication;
- removal or explicit redesign of the v1 whole-scan adapter;
- corrected file lists, including `Makefile.local`, and completion state for Task 2.

```findings
findings:
  - id: new
    severity: Critical
    family: proofless-binding-migration
    title: |
      Proofless binding migration is unreachable in production
    detail: |
      ProofMigrator.Request has only test callers, while QuerySession merely reports migration pending and returns provisional forever. Wire one-root validation and durable proof publication into the production owner lifecycle, with an integration regression that fails without the wiring.
  - id: new
    severity: Critical
    family: launch-boundary-framing
    title: |
      Persisted raw launch boundaries are ignored by watcher selection
    detail: |
      TargetArtifactBoundary retains only path fields, and no production code consumes RawSize or reads the B-1 boundary byte. The newline/incomplete-record contract and its positive watcher regressions are therefore absent.
  - id: new
    severity: Critical
    family: catalog-monotonic-publication
    title: |
      A stale same-key catalog writer can overwrite newer cursor or disputed state
    detail: |
      persistTrackedCatalog unconditionally assigns each incoming entry over the locked current entry. Serialization prevents byte corruption but does not prevent an older validator from publishing after and replacing a newer cursor, scanner state, or disputed transition.
  - id: new
    severity: Critical
    family: catalog-authority-single-source
    title: |
      The planned catalog-backed IncrementalInventory production seam does not exist
    detail: |
      ReconcileCatalog is called only by tests, there is no IncrementalInventory entity, and launch, watcher, and query use parallel policies. This contradicts the Core concepts table and violates ARCH-DRY and ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: latency-path-whole-scan
    title: |
      The v1 watcher compatibility path still scans the complete corpus
    detail: |
      sessionwatch.Run invokes InventoryWithRuntime and NativeEventsWithRuntime on every v1 poll, and the shadow test explicitly allowlists both calls. An in-place upgrade therefore retains the latency behavior the Spec prohibits on interactive paths.
  - id: new
    severity: Important
    family: external-conformance-cadence
    title: |
      Live provider conformance does not compare installed behavior with the stateful fake
    detail: |
      The live test succeeds after finding one whole-file sample accepted by the validator. It does not compare normalized transitions with FakeRuntime or exercise the append-only behavior Pair relies on, leaving ARCH-MOCK drift undetected.
  - id: new
    severity: Important
    family: plan-checklist-integrity
    title: |
      Task 2 remains wholly unchecked at the claimed whole-issue boundary
    detail: |
      Eleven ProviderContract, reconciliation, concept-contract, verification, and commit steps remain unchecked in the durable plan. Reconcile the checklist after the blocking implementation gaps are addressed.
```

---

## Re-review — 2026-08-29T13:57:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..a75ca16de75b53668f652b3faa99d733d9a15b9f |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T13:57:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The corrective round adds useful primitives and passes focused, integration, and live-conformance checks, but three prior Critical findings remain incomplete: proof migration lacks a reachable upgrade lifecycle, B−1 framing remains test-only, and catalog publication can still regress the parser cursor. Two additional production paths can publish proofless authority or suppress a legitimate post-launch artifact. These block the boundary.

### 1. Strengths

- The v1 unbound watcher now fails closed without corpus enumeration, with a production-boundary regression at [run_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run_test.go:121).
- `IncrementalInventory` now gives launch, query, and watcher code a shared metadata/reconciliation entry point at [incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:30).
- The focused race suite, Lua suite, watcher/terminal shell tests, vet, Zellij configuration check, and diff check passed.
- Live verification passed for the 1,350-file metadata inventory in 44.6 ms and for the installed Claude, Codex, Muse, and Agy samples.
- Task 2 is now reconciled, and the plan contains a dated revision entry.

### 2. Critical findings

- **BR-1 — not addressed:** [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:166) migrates a proofless binding only when `session-watch` runs for that already-bound launch. Production launches spawn the watcher when creating a launch, but the original watcher has already exited after writing a v1 binding; no lifecycle restarts it merely because Pair was upgraded. [run_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run_test.go:91) calls `Run` directly and therefore proves the worker logic, not production reachability. Add an upgrade integration path that begins with a persisted legacy owner, enters a lifecycle that actually runs after upgrade, and observes durable proof publication.

- **BR-2 — not addressed:** `ObserveLaunchBoundarySuffix` exists at [incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:80), but the only callers are its unit tests. [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:247) copies the boundary tuple but uses it only for baseline exclusion; no production path reads B−1. Wire the helper into the authorized established/explicit-target flow and add watcher-level newline, incomplete-record, truncation, and replacement regressions. This remains an `ARCH-PURPOSE` failure.

- **BR-3 — not addressed:** [catalog.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/catalog.go:139) rejects a lower raw offset but accepts an incoming entry with a higher raw offset and lower parser-complete offset. A stale validator at parser cursor 10 that observes an incomplete tail through raw offset 30 can therefore overwrite current `{raw:20, parser:20}` with `{raw:30, parser:10}`. The regression only tests both offsets decreasing together. Define the monotonic partial order over every cursor/state axis and test crossed cursor values. This is the second finding in family `catalog-monotonic-publication`; fix the rule, not only this tuple.

- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:114) clears `proofs` when catalog publication fails, but [lifecycle.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/lifecycle.go:198) then falls back to `AppendBindingIfCurrent` when that proof is missing. Thus the path whose diagnostic says “binding authority withheld” can publish a proofless binding and terminate the watcher. Remove proofless publication from the v2 watcher and test catalog failure through `Run`, asserting no binding is written. This is the second finding in family `proofless-binding-migration`; the governing rule should be that v2 never publishes binding authority without its proof.

- [incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:60) limits a new launch to entries classified `CatalogWorkNew` before applying the launch baseline. If another watcher catalogs a genuinely post-launch artifact first, this watcher sees it as reused/append rather than `new` and never validates it, even though it was absent from this launch’s durable baseline. Launch eligibility must derive from the launch boundary; catalog state may determine how authorized work advances but must not erase per-launch newness. This is the second finding in family `catalog-authority-single-source`; enumerate concurrent catalog/launch orderings and enforce one rule across them (`ARCH-DRY`, `ARCH-PURPOSE`).

### 3. Important findings

- **BR-6 — not addressed:** [provider_live_fake_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/provider_live_fake_test.go:22) compares incremental/full replay for Claude, Codex, and Muse only. The Agy live test still accepts one copied whole-file sample without comparing joined-transcript append transitions against the stateful fake. Extend the same normalized transition comparison to Agy’s database-plus-transcript seam (`ARCH-MOCK`).

- `make test-session-inventory-conformance` is a new operator-facing command documented in the atlas, but `README.md` is unchanged in the review range. Add the runnable verification surface and its purpose to the README.

### 4. Minor findings

None.

### 5. Test coverage notes

Verified passing:

- Focused race suite for session inventory, fake runtime, ledger, watcher, and launcher.
- `go vet -p 20 ./...`
- `make test-lua`
- Terminal shortcut and watcher shell suites.
- Zellij configuration and `git diff --check`.
- Both installed-data conformance commands.

`go test -p 20 ./... -count=1` could not complete cleanly because the sandbox denied `/bin/ps`; all reported failures were in `TestPublicPairCommandFamiliesIgnoreCouchStore` with `operation not permitted`.

The claimed regressions do not currently turn red for production B−1 reachability, real legacy-upgrade migration, crossed raw/parser cursors, catalog-publication failure, or concurrent catalog-before-watcher ordering.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** catalog-delta newness and launch-baseline newness form competing authorities.
- **ARCH-PURE — pass:** reconciliation, selection, framing, and merge policy remain isolated pure logic; IO stays behind runtimes.
- **ARCH-PURPOSE — flag:** key launch-boundary and durable-proof contracts remain unreachable or bypassable.
- **ARCH-MOCK — flag:** the stateful fake is shared by production-boundary tests, but Agy’s live append behavior is not compared against it.

### 7. Plan revision recommendations

Append a dated revision recording:

- the actual post-upgrade lifecycle that triggers legacy proof migration;
- production wiring for B−1 framing;
- the full monotonic catalog publication rule across raw offset, parser offset, dispute, generation, and scanner state;
- removal of v2 proofless binding publication;
- independence of launch-baseline eligibility from concurrent catalog classification;
- Agy live fake comparison;
- correction of the Core concepts claims that `ProofMigrator` is injected into ordinary lookups and that `IncrementalInventory` itself performs targeted reads/publication.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The worker is called by session-watch, but no production lifecycle restarts that watcher for an already-bound v1 owner after upgrade; the regression invokes Run directly.
  - id: BR-2
    disposition: not-addressed
    note: |
      ObserveLaunchBoundarySuffix has only test callers, so production still never reads B-1.
  - id: BR-3
    disposition: not-addressed
    note: |
      A higher raw offset with a lower parser-complete offset is accepted and can regress the parser cursor.
  - id: BR-4
    disposition: addressed
    note: |
      IncrementalInventory now exists and production launch, watcher, query, and launcher paths call the shared reconciliation/selection seam.
  - id: BR-5
    disposition: addressed
    note: |
      The v1 unbound watcher fails closed without inventory or event corpus scans, with a production-boundary regression.
  - id: BR-6
    disposition: not-addressed
    note: |
      Stateful append comparison covers Claude, Codex, and Muse, but Agy still validates one copied whole-file sample without a fake transition comparison.
  - id: BR-7
    disposition: addressed
    note: |
      Task 2 is fully checked and the corrective work is recorded in a dated Revisions entry.
findings:
  - id: new
    severity: Critical
    family: proofless-binding-migration
    title: |
      Catalog publication failure still writes a proofless v2 binding
    detail: |
      Run clears the proof map after catalog failure, but ObserveAndPersist falls back to AppendBindingIfCurrent and terminates with proofless authority. This is the 2nd finding in family proofless-binding-migration; enforce the class rule that v2 never publishes a binding without its proof.
  - id: new
    severity: Critical
    family: catalog-authority-single-source
    title: |
      Concurrent catalog publication can hide a genuinely post-launch artifact
    detail: |
      TargetNewLaunch first retains only CatalogWorkNew entries, so an artifact absent from this launch baseline becomes ineligible if another writer catalogs it before the watcher observes it. This is the 2nd finding in family catalog-authority-single-source; define launch newness solely from the durable launch boundary.
  - id: new
    severity: Important
    family: user-facing-surface-documentation
    title: |
      README documentation is missing for the conformance target
    detail: |
      The range adds the operator-runnable make test-session-inventory-conformance surface, but README.md is unchanged.
```

---

## Re-review — 2026-08-29T14:24:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..37d8142cde3793217c3e888d9fd5da5d88789a63 |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T14:24:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The corrective round resolves the prior authority-lifecycle, monotonic-publication, concurrent-launch, conformance, and documentation findings. However, the issue’s central single-source guarantee remains unenforced: interactive queries instantiate an empty catalog and never persist advanced proof/catalog state, while the shadow sweep no longer detects independent native scanners or path reconstruction. This is the third recurrence of `catalog-authority-single-source` and blocks closure.

### 1. Strengths

- `cmd/internal/sessionwatch/run.go:134` requires proof-bearing watcher publication; the catalog-failure integration regression reaches this production path.
- `cmd/internal/sessioninventory/catalog.go:122` enforces independent monotonicity for raw and parser-complete offsets and preserves terminal disputes.
- `cmd/internal/sessioninventory/target.go:125` defines launch newness solely from the durable baseline, independent of concurrent catalog publication.
- `cmd/internal/launcher/createflow.go:29` makes proof migration production-reachable; launcher and persisted-filesystem tests separately pin invocation and durable publication.
- All four providers now participate in the stateful-fake conformance flow, and `README.md:393` documents the runnable target.

### 2. Critical findings

- `cmd/internal/sessioninventory/query.go:119` and `cmd/internal/sessioninventory/shadow_test.go:16` — **ARCH-DRY, ARCH-PURPOSE:** every `QuerySession` constructs `IncrementalInventory` with an empty catalog and never publishes an advanced proof or catalog entry. After a proof-bearing artifact appends, repeated context/title/activity/opener/review queries therefore revalidate the same suffix from the stale ledger proof instead of deriving from one durable catalog. Simultaneously, the revised shadow test only recognizes calls named `InventoryWithRuntime`, `NativeEventsWithRuntime`, or the activity subprocess; it removed the former guards against direct native paths, parser duplication, `lsof`, and config-based authority.

  **This is the 3rd finding in family `catalog-authority-single-source`.** Do not patch only `QuerySession`. State and enforce the class rule: enumerate every latency-sensitive consumer, route each through one persistent catalog/proof advancement owner, and make the shadow sweep reject both known whole-inventory calls and independent reconstruction of native authority. Add a repeated-query regression where an append is consumed once and the next unchanged query performs zero scanner body reads.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

- Passed: focused `sessioninventory`, `sessionwatch`, and `launcher` packages.
- Passed: `go vet -p 20 ./...`, `make test-lua`, watcher and terminal shell suites, Zellij configuration validation, and `git diff --check`.
- The full `go test -p 20 ./... -count=1` reached `cmd/pair-go` but its subprocess cases could not execute `/bin/ps` in the review sandbox (`operation not permitted`). No changed-logic assertion failed.
- Missing regression: append once, query twice, and prove the second query performs no repeated suffix validation.
- Missing enforcement regression: a synthetic interactive consumer that reconstructs a native path or scanner must make the shadow sweep fail.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** query advancement and durable catalog ownership remain parallel authorities.
- **ARCH-PURE — pass:** reconciliation and target selection are cleanly separated from injected IO.
- **ARCH-PURPOSE — flag:** the shadow-sweep and repeated-query contracts do not enforce the issue’s central single-source purpose.
- **ARCH-MOCK — pass:** production and test flows share the runtime boundary, including installed-provider comparison for Claude, Codex, Muse, and Agy.

### 7. Plan revision recommendations

Append a dated `## Revisions` entry that:

- Defines the persistent advancement owner used by every interactive query.
- Enumerates launch, existence, owner, activity, recovery, context, review, slug, opener, title-poller, and confirmation consumers.
- Restores shadow enforcement for direct native paths, independent parsers, external discovery commands, and compatibility-config authority—not only calls with two known function names.
- Adds the repeated-append/query zero-reread regression.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Launcher startup invokes bounded migration, and persisted-ledger integration coverage proves durable proof publication.
  - id: BR-2
    disposition: withdrawn
    note: |
      The appended authority-first revision supersedes B-1 framing: preexisting unbound artifacts remain unread, while authorized targets advance from proof parser offsets.
  - id: BR-3
    disposition: addressed
    note: |
      Catalog merge now rejects regression on either cursor independently, with crossed-cursor coverage.
  - id: BR-4
    disposition: addressed
    note: |
      IncrementalInventory is production-reachable from watcher and targeted query flows.
  - id: BR-5
    disposition: addressed
    note: |
      The v1 unbound watcher fails closed without listing the native corpus.
  - id: BR-6
    disposition: addressed
    note: |
      Installed Agy database-plus-transcript behavior now joins the stateful-fake prefix-to-append comparison and durable target.
  - id: BR-7
    disposition: addressed
    note: |
      The claimed plan checklist is checked and the corrective revisions record the changed design.
  - id: BR-8
    disposition: addressed
    note: |
      V2 watcher publication requires a proof, and catalog failure is covered through Run with no binding written.
  - id: BR-9
    disposition: addressed
    note: |
      New-launch selection now derives from the durable launch baseline rather than catalog work classification.
  - id: BR-10
    disposition: addressed
    note: |
      README documents the conformance target and its purpose.
findings:
  - id: new
    severity: Critical
    family: catalog-authority-single-source
    title: |
      Interactive query advancement still bypasses durable catalog authority and its enforcement sweep
    detail: |
      This is the 3rd finding in family catalog-authority-single-source. QuerySession creates an empty catalog on every call and publishes neither advanced proof nor catalog state, so repeated queries can reread the same appended suffix. The replacement shadow sweep also dropped direct native-path, parser, lsof, and config-authority guards; state the rule for every latency-sensitive consumer and enforce the complete enumeration rather than patching this instance.
```

---

## Re-review — 2026-08-29T14:47:30-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..da2643d03da68d5818d36ea045334e70d6b9fd6e |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T14:47:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The persistent query advancement itself is correctly implemented and mutation-pinned, and the full verification suite ultimately passes. The boundary remains blocked because BR-11’s “exhaustive” authority sweep excludes the entire inventory package, allowing new interactive whole-corpus consumers there without detection. The Core concepts table also names the wrong location for `IncrementalResult`.

## Strengths

- `QuerySession` loads and publishes durable catalog advancement through the shared boundary ([query.go:110](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query.go:110)).
- Removing publication in a scratch copy made `TestQuerySessionPersistsAppendAdvancementAcrossQueries` fail with repeated body reads, proving the regression test is effective.
- `PublishTargetValidations` gives watcher and query paths one monotonic publication helper ([catalog_publication.go:8](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/catalog_publication.go:8)).
- Atlas and README updates cover the catalog, bounded owner projection, conformance command, and maintenance cadence.
- Live conformance passed for all four providers; the observed 1,350-file metadata pass completed in 45.6 ms.

## Critical findings

1. **BR-11 remains not addressed — ARCH-DRY, ARCH-PURPOSE.**
   [shadow_test.go:76](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/shadow_test.go:76) skips all of `cmd/internal/sessioninventory`, rather than allowing only named authority/diagnostic seams. A scratch `badInteractiveShadow` calling `InventoryWithRuntime` inside that directory passed `TestNoNativeAuthorityShadowInInteractiveConsumers`.

   **This is the 4th finding in family `catalog-authority-single-source`.** Do not add another instance-specific regex. Replace the directory exemption with a syntax-aware, fixed allowlist covering definitions and explicitly named diagnostic/conformance entry points. Add an in-package synthetic violating consumer that must make the sweep fail.

2. **Core concepts table points to a nonexistent location.**
   [000156-incremental-session-inventory-plan.md:34](/Users/xianxu/workspace/pair/workshop/plans/000156-incremental-session-inventory-plan.md:34) states that both `ScannerState` and `IncrementalResult` live in `scanner_state.go`, but `IncrementalResult` is declared at [incremental_inventory.go:13](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:13).

   Fix by appending a plan revision that splits or corrects the row and records the implemented location.

## Important findings

None.

## Minor findings

None.

## Test coverage notes

- Focused BR-11 tests passed at the pinned head.
- Publication-removal mutation correctly failed the repeat-query regression.
- In-package shadow mutation incorrectly remained green, demonstrating BR-11’s enforcement gap.
- `go test -p 20 ./... -count=1` passed on rerun. The first run had one transient launcher timeout; that test then passed five consecutive focused runs.
- `go vet -p 20 ./...`, `make test-lua`, both relevant shell suites, Zellij config validation, live conformance, and `git diff --check` passed.

## Architectural notes

- **ARCH-DRY: flag.** Query and watcher share publication logic, but the guard does not enforce that authority inside the owning package.
- **ARCH-PURE: pass.** Pure reconciliation/selection remain separated from runtime and catalog-store IO.
- **ARCH-PURPOSE: flag.** The complete consumer enumeration promised by the issue is not mechanically enforced.
- **ARCH-MOCK: pass.** Stateful runtime behavior and installed-provider conformance exercise the production boundary.

## Plan revision recommendations

Append a `## Revisions` entry that:

- Corrects `IncrementalResult` to `cmd/internal/sessioninventory/incremental_inventory.go`.
- Supersedes the claim that the authority sweep is exhaustive until it uses a closed, named allowlist including inventory-package consumers.

```findings
dispose:
  - id: BR-11
    disposition: not-addressed
    note: |
      Query advancement is mutation-pinned, but the shadow sweep skips the entire sessioninventory package; a synthetic in-package whole-inventory consumer passes undetected.
findings:
  - id: new
    severity: Critical
    family: core-concept-table-integrity
    title: |
      Core concepts table gives IncrementalResult the wrong file location
    detail: |
      The plan places IncrementalResult in scanner_state.go, while the symbol is declared in incremental_inventory.go; append a revision correcting or splitting the row.
```

---

## Re-review — 2026-08-29T14:58:45-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..9b125f1d8d12138117cf58957ef2a0501098efa8 |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T14:58:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The production defect behind BR-11 is fixed and protected by a mutation-sensitive regression, while BR-12 is corrected through the required appended revision. Focused race tests and all independent integration checks passed. One Important enforcement gap remains: the claimed closed authority sweep can be bypassed through function aliases, so it does not yet fully enforce the single-source rule.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      QuerySession now loads and publishes durable catalog advancement; disabling publication makes TestQuerySessionPersistsAppendAdvancementAcrossQueries fail because the second query rereads the suffix.
  - id: BR-12
    disposition: addressed
    note: |
      The fourth plan revision explicitly places ScannerState in scanner_state.go and IncrementalResult in incremental_inventory.go.
findings:
  - id: new
    severity: Important
    family: catalog-authority-single-source
    title: |
      Closed authority sweep can be bypassed through function aliases
    detail: |
      This is the 4th finding in family catalog-authority-single-source. wholeInventoryCalls checks only CallExpr callee names, so scan := InventoryWithRuntime; scan(runtime) is invisible; the synthetic regression covers only a direct call. Do not patch only that spelling: enforce the rule over every reference or resolved call edge to InventoryWithRuntime and NativeEventsWithRuntime, allowing only the two named production sites, and add alias/selector mutation cases.
```

### 1. Strengths

- [query.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query.go:110) now loads catalog state, advances from it, and publishes accepted validation through the persistent runtime seam.
- [query_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query_test.go:66) proves a second unchanged query performs no additional body reads. Disabling publication in a scratch copy made this test fail as intended.
- [catalog_publication.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/catalog_publication.go:8) provides one shared publication boundary for watcher and interactive-query advancement.
- README and atlas changes document the new catalog, targeted query flow, immediate Alt+X ordering, and live conformance target.

### 2. Critical findings

None.

### 3. Important findings

- [shadow_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/shadow_test.go:142): `wholeInventoryCalls` recognizes only direct calls whose callee is named `InventoryWithRuntime` or `NativeEventsWithRuntime`. A local alias evades the sweep. Use reference/type resolution or explicitly reject all references outside the two allowed call sites; add mutation fixtures for local and selector aliases. `ARCH-DRY`, `ARCH-PURPOSE`.

### 4. Minor findings

None.

### 5. Test coverage notes

- Passed focused race suites for session inventory, fake runtime, ledger, watcher, and launcher.
- Passed `go vet ./...`, Lua tests, terminal shortcut tests, session-watch shell tests, Zellij configuration validation, and `git diff --check`.
- Full `go test ./...` completed successfully except `cmd/pair-go`, whose fixtures were uniformly blocked by the review sandbox denying `/bin/ps`. No changed-behavior assertion failed.
- The BR-11 regression was independently mutation-checked and went red without publication.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Flagged only for the alias loophole in the enforcement sweep; production consumers otherwise converge on `QuerySession`.
- `ARCH-PURE`: Pass. Reconciliation, selection, and catalog merging remain pure; filesystem and persistence operations sit behind injected seams.
- `ARCH-PURPOSE`: Flagged because the “closed allowlist” claim is not fully enforced against indirect references.
- `ARCH-MOCK`: Pass. Production and tests share the runtime boundary, and the stateful fake models catalog persistence and multi-query advancement.

### 7. Plan revision recommendations

Append a fifth `## Revisions` entry stating that the authority sweep governs resolved references/call edges—not only direct call syntax—and documenting alias/selector mutation coverage.
