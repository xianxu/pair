# Boundary Review — pair#155 (milestone M1)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4c454436038e2ae049690bc343def9f0511fca8c..d9869be4ba1c53ac6f4280487470884dcf6760f0 |
| command | sdlc milestone-close --issue 155 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-28T13:48:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 foundation has a sound pure-core/runtime-seam shape, and focused tests plus live conformance pass. It cannot cross the boundary yet: deterministic repository contract tests fail, several exact Spec contracts are not represented, and scanner behavior violates ordering, diagnostics, filesystem-safety, and token-usage requirements.

## 1. Strengths

- `BuildForest` cleanly separates topology construction from scanner IO and fails closed on conflicting or cyclic parent relationships.
- Agent-specific scanners use bounded, versioned parsing with sanitized fixtures.
- Production and test flows share the `Runtime` boundary; `FakeRuntime` models persistent filesystem, SQLite, process identity, and open-file state.
- Live conformance passed with redacted output for all four agents: Agy 20 nodes, Claude 1,392, Codex 1,192, and Muse 362.
- Atlas coverage appropriately describes the new M1 surface; no public CLI exists yet, so a README change is not required at this boundary.

## 2. Critical findings

1. **Repository contract suites fail at the pinned head.**
   `cmd/internal/artifactpath/coverage_test.go:165-177` reports every new `sessioninventory` production file as absent from the exhaustive inventory, including an unclassified `agent` reference in `scan_claude.go`. `cmd/internal/couchcore/plan_contract_test.go:102-118` also reports the new Go sources as missing from its disposition catalog. Update both governing inventories now and pin the package classifications; deferring artifact classification to Task 10 leaves the repository red throughout M1/M2.

2. **The Core Concepts table contradicts the implementation.**
   The plan promises `NativeRecordFact`, `SessionNode`, `SessionForest`, `ParentEdge`, and `EdgeProvenance`, but `model.go:58-85` defines only `Fact`, `Node`, and `Forest`, with parent data reduced to `ParentID`. No edge provenance exists. Implement the named edge/provenance representation and test that it survives forest projection, or append a plan revision for intentional type renames. Parent-edge provenance itself cannot simply be deferred because both M1 and the final binding contract require it.

3. **The diagnostic contract is not implemented.**
   `model.go:87-117,359-374` assigns every diagnostic `warning`, omits `storage_absent`, hashes `detail` instead of the specified canonical tuple, and lacks the exact registry behavior. `order.go:105-121` orders by code before severity and does not coalesce identical diagnostics. `scan_helpers.go:101-103` silently emits nothing for absent storage. Centralize the exhaustive code→severity registry, canonical ID tuple, severity ordering, coalescing, and absent-storage diagnostic. Add table-driven tests covering every M1 code. This also addresses the duplicated registry logic in `conformance.go:151-158` (`ARCH-DRY`, `ARCH-PURPOSE`).

4. **Node ordering violates the documented total tuple.**
   `model.go:460-472` compares `TimeSource` when timestamps are equal, and `order.go:74-84` applies that before `native_id`. The Spec requires `(time missing, time, native_id, storage_root, relative_path)`; equal instants must therefore fall through directly to native ID. Remove source from chronology comparison used for node ordering and add a regression with equal timestamps from different sources (`ARCH-PURPOSE`).

5. **The filesystem boundary admits blocking non-regular files and loses partial inventories.**
   `runtime_os.go:70-103` rejects symlinks but accepts FIFOs, sockets, and devices as files; a later `ReadAt` can block indefinitely on a FIFO. Separately, `runtime_os.go:109-110` returns already-enumerated files with a walk error, but `scan_helpers.go:112` discards those partial results. Require `Mode().IsRegular()`, reject all non-regular entries, and preserve valid entries alongside structured per-entry diagnostics. Add a FIFO fixture and an injected partial-walk failure test.

6. **Missing token-usage objects are accepted as real zero usage.**
   In `usage.go:27-40`, a Codex `info:{}` record produces a valid zero value; in `usage.go:43-57`, a normal Claude assistant record with absent/null `usage` also overwrites the last valid usage with zero. The Core Concept explicitly excludes null usage. Represent both nested usage objects with pointers/presence checks and add regressions proving absent/null records cannot replace the last accepted usage.

## 3. Important findings

None beyond the blocking findings above.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- `go test ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest ./cmd/internal/procutil -count=1`
- `go vet ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest`
- `git diff --check <base> <head>`
- Opt-in live native-session conformance

Deterministically failed:

- `go test ./cmd/internal/artifactpath -run '^TestProductionArtifactReferencesAreExactlyClassified$' -count=1`
- `go test ./cmd/internal/couchcore -run '^TestIssue149M5DeclarationDispositionSourceSetMatchesMilestoneDiff$' -count=1`

The purported shuffle in `forest_projection_test.go:27-33` does not actually reorder anything. Expand it to multiple forests, siblings, artifacts, and diagnostics while adding the ordering regressions above.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** diagnostic severity, identity, ordering, and conformance policy need one registry.
- **ARCH-PURE — pass:** forest construction, ordering, IDs, and rendering are pure; IO remains behind the injected runtime.
- **ARCH-PURPOSE — flag:** missing edge provenance and exact-contract drift under-deliver the deterministic-forest boundary.
- **ARCH-MOCK — pass:** production and fake implementations share the same runtime seam, and the live conformance probe exercises installed dependencies.

## 7. Plan revision recommendations

Append a `## Revisions` entry recording:

- The final M1 exported type naming and explicit `ParentEdge`/`EdgeProvenance` representation.
- Movement of artifact-source classification and historical source-set maintenance from Task 10 into M1, because their existing repository contracts fail as soon as the package is introduced.
- The expanded M1 verification commands for diagnostic registry, equal-time ordering, non-regular/partial traversal, and absent token-usage cases.

```findings
findings:
  - id: new
    severity: Critical
    family: repository-contracts-stay-green
    title: |
      New inventory sources leave deterministic repository contract tests failing
    detail: |
      artifactpath coverage and the issue-149 source-set contract both fail at the pinned head; classify the new sources and update the governed catalog before closing M1.
  - id: new
    severity: Critical
    family: core-concepts-match-code
    title: |
      M1 omits the promised ParentEdge and EdgeProvenance entities
    detail: |
      model.go exposes only ParentID on Node, while the Core Concepts table requires explicit edge and provenance entities; implement and projection-test them, and reconcile the renamed Fact/Node/Forest types through a plan revision.
  - id: new
    severity: Critical
    family: diagnostic-registry-single-source
    title: |
      Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
    detail: |
      Every diagnostic is currently warning, storage_absent is omitted, IDs include free-form detail, sorting ignores the required severity order, and duplicates are retained; centralize and exhaustively test the documented registry.
  - id: new
    severity: Critical
    family: documented-total-order
    title: |
      Equal timestamps are ordered by time source instead of native ID
    detail: |
      compareNativeTime inserts TimeSource into the node comparator even though the specified tuple falls through from time directly to native_id; add an equal-time mixed-source regression and implement the exact tuple.
  - id: new
    severity: Critical
    family: storage-boundary-regular-partial
    title: |
      Filesystem enumeration admits blocking special files and discards valid partial results
    detail: |
      ListFiles rejects symlinks but accepts FIFOs, sockets, and devices, while scannerFiles drops valid entries returned with non-listing walk errors; reject every non-regular candidate and retain partial facts with structured diagnostics.
  - id: new
    severity: Critical
    family: optional-metrics-require-presence
    title: |
      Missing Claude and Codex usage objects overwrite valid token usage with zero
    detail: |
      Value-typed nested usage structs cannot distinguish absent or null usage from a real zero; require explicit presence before accepting a record and test that missing records do not replace the last valid sample.
```

---

## Re-review — 2026-08-28T14:05:07-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 27d1aa922b8e455cb405c14dceb0846be55e6252..27d1aa922b8e455cb405c14dceb0846be55e6252 |
| command | sdlc milestone-close --issue 155 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-28T14:05:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 implementation and all six prior fixes are substantively covered and pass their focused tests. However, the Core Concepts table still contradicts the code: `NativeRecordFact` is declared in `model.go`, not its documented `scan.go` location. Because this repeats an existing finding family, the plan-to-code inventory needs class-level enforcement before the boundary can close.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Both governed repository contract tests pass and directly detect removal of the new inventory sources from their catalogs.
  - id: BR-2
    disposition: addressed
    note: |
      ParentEdge and EdgeProvenance exist, are populated by BuildForest, and are asserted through model and canonical projection tests.
  - id: BR-3
    disposition: addressed
    note: |
      The exhaustive registry test pins every code and severity, detail-independent IDs, severity ordering, coalescing, and storage_absent behavior.
  - id: BR-4
    disposition: addressed
    note: |
      The mixed-source equal-time regression requires native-ID ordering and compareNativeTime now ignores TimeSource after equal timestamps.
  - id: BR-5
    disposition: addressed
    note: |
      Tests exercise a real FIFO, rejected symlink, retained regular entries, and valid facts returned alongside a generic partial-listing error.
  - id: BR-6
    disposition: addressed
    note: |
      Claude and Codex regressions place absent usage after valid usage; pointer-backed presence checks prevent the prior zero overwrite.
findings:
  - id: new
    severity: Critical
    family: core-concepts-match-code
    title: |
      The M1 Core Concepts inventory still names the wrong source for NativeRecordFact
    detail: |
      This is the 2nd finding in family core-concepts-match-code. A sweep of all eight M1 Core Concepts rows found one contradiction: workshop/plans/000155-agent-session-tree-inventory-plan.md:21 locates NativeRecordFact in scan.go, while its declaration is in model.go:82. Do not fix only this row; state and enforce the rule that every concept name, kind, status, and path is mechanically checked against the tree, then append a plan revision recording the effective correction (ARCH-PURPOSE).
```

1. Strengths

- Explicit parent edges and provenance are modeled and projected at [model.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/model.go:56).
- The centralized diagnostic registry and adversarial registry/coalescing tests cover BR-3 at [diagnostic.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/diagnostic.go:5) and [diagnostic_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/diagnostic_test.go:8).
- The runtime rejects non-regular entries while retaining valid partial results at [runtime_os.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/runtime_os.go:63).
- Atlas documentation describes the new M1 boundary and explicitly records which consumers remain unmigrated.

2. Critical findings

- Core Concepts path mismatch described in the machine-readable finding above. Add an exhaustive `#155` concept-contract test analogous to the existing `#149` enforcement, then append the correction under `## Revisions`.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Focused session-inventory, fake-runtime, artifactpath, and historical source-set tests all passed. The broader `go test ./... -count=1` run failed reproducibly in unrelated `cmd/internal/couchcore.TestSpawnComposesProductionPairRegistrationBoundary` with a 20-second `zellij-ready` timeout; it is outside the M1 change surface and was not raised as an M1 finding, but it is not clean full-suite evidence.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass—shared forest, diagnostic, scanner-helper, and runtime authorities are used.
- `ARCH-PURE`: Pass—forest construction and ordering remain pure; effects cross the injected runtime.
- `ARCH-PURPOSE`: Flag—the M1 behavior is delivered, but the repeated concept-inventory contradiction shows its architectural record is not yet enforced.
- `ARCH-MOCK`: Pass—the stateful fake implements the production runtime seam, with portable fixtures and live conformance coverage.

No README update is required at M1 because the public `session-inventory` command remains scheduled for M2; the internal conformance target is documented in the atlas.

7. Plan revision recommendation

Append:

> `### 2026-08-28 — M1 Core Concepts inventory enforcement`
>
> **Reason:** A second `core-concepts-match-code` review found that manually maintained entity paths can remain inconsistent after implementation aliases are introduced.
>
> **Delta:** Record `NativeRecordFact` as declared in `cmd/internal/sessioninventory/model.go` and add an exhaustive contract that validates every Core Concepts row’s name, kind, status, and path against the repository.

---

## Re-review — 2026-08-28T14:10:12-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 155 — deterministic agent session-tree inventory |
| repo | pair |
| issue file | workshop/issues/000155-agent-session-tree-inventory.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 84a7131d1a1d0b34018e0a059bf41103749e36bf..84a7131d1a1d0b34018e0a059bf41103749e36bf |
| command | sdlc milestone-close --issue 155 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-28T14:10:12-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-7 is addressed. Although the pinned Base→Head range is empty, direct inspection of HEAD, the fixing commit, the executable contract, and a counterfactual regression confirms the corrected path and class-wide enforcement. No new findings block M1.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      The plan now locates NativeRecordFact in model.go, a bidirectional declaration contract checks every M1 concept field, and restoring the stale scan.go path makes that test fail.
```

1. Strengths

- The plan correctly locates `NativeRecordFact` at [000155-agent-session-tree-inventory-plan.md:21](/Users/xianxu/workspace/pair/workshop/plans/000155-agent-session-tree-inventory-plan.md:21), matching its declaration at [model.go:84](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/model.go:84).
- [concept_contract_test.go:26](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/concept_contract_test.go:26) checks all M1 concept names, paths, kinds, statuses, and milestones bidirectionally.
- The required revision records both the correction and the general enforcement rule at [000155-agent-session-tree-inventory-plan.md:685](/Users/xianxu/workspace/pair/workshop/plans/000155-agent-session-tree-inventory-plan.md:685).
- Counterfactual verification changing only the plan path back to `scan.go` failed with the expected path mismatch.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

The focused concept contract, session-inventory packages, stateful fake, `procutil`, artifact classification, Couch declaration contracts, and `git diff --check` all passed. The stale-path counterfactual failed as required.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass—the duplicated plan/declaration facts are exhaustively synchronized.
- `ARCH-PURE`: Pass—production forest concepts remain pure; repository inspection stays in the contract test.
- `ARCH-PURPOSE`: Pass—the fix covers the entire enumerable M1 Core Concepts class, not only `NativeRecordFact`.
- `ARCH-MOCK`: Pass—no new external dependency was introduced; M1 retains the shared runtime seam and stateful fake.

7. Plan revision recommendations

None; the required `## Revisions` entry is present and matches the implementation.
