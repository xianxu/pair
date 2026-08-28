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
