---
gate: boundary-review
issue: 155
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T13:48:39-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: New inventory sources leave deterministic repository contract tests failing
          detail: artifactpath coverage and the issue-149 source-set contract both fail at the pinned head; classify the new sources and update the governed catalog before closing M1.
          family: repository-contracts-stay-green
          round: 1
        - id: BR-2
          severity: Critical
          title: M1 omits the promised ParentEdge and EdgeProvenance entities
          detail: model.go exposes only ParentID on Node, while the Core Concepts table requires explicit edge and provenance entities; implement and projection-test them, and reconcile the renamed Fact/Node/Forest types through a plan revision.
          family: core-concepts-match-code
          round: 1
        - id: BR-3
          severity: Critical
          title: Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
          detail: Every diagnostic is currently warning, storage_absent is omitted, IDs include free-form detail, sorting ignores the required severity order, and duplicates are retained; centralize and exhaustively test the documented registry.
          family: diagnostic-registry-single-source
          round: 1
        - id: BR-4
          severity: Critical
          title: Equal timestamps are ordered by time source instead of native ID
          detail: compareNativeTime inserts TimeSource into the node comparator even though the specified tuple falls through from time directly to native_id; add an equal-time mixed-source regression and implement the exact tuple.
          family: documented-total-order
          round: 1
        - id: BR-5
          severity: Critical
          title: Filesystem enumeration admits blocking special files and discards valid partial results
          detail: ListFiles rejects symlinks but accepts FIFOs, sockets, and devices, while scannerFiles drops valid entries returned with non-listing walk errors; reject every non-regular candidate and retain partial facts with structured diagnostics.
          family: storage-boundary-regular-partial
          round: 1
        - id: BR-6
          severity: Critical
          title: Missing Claude and Codex usage objects overwrite valid token usage with zero
          detail: Value-typed nested usage structs cannot distinguish absent or null usage from a real zero; require explicit presence before accepting a record and test that missing records do not replace the last valid sample.
          family: optional-metrics-require-presence
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-28T14:05:07-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both governed repository contract tests pass and directly detect removal of the new inventory sources from their catalogs.
          round: 2
        - id: BR-2
          disposition: addressed
          note: ParentEdge and EdgeProvenance exist, are populated by BuildForest, and are asserted through model and canonical projection tests.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The exhaustive registry test pins every code and severity, detail-independent IDs, severity ordering, coalescing, and storage_absent behavior.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The mixed-source equal-time regression requires native-ID ordering and compareNativeTime now ignores TimeSource after equal timestamps.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Tests exercise a real FIFO, rejected symlink, retained regular entries, and valid facts returned alongside a generic partial-listing error.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Claude and Codex regressions place absent usage after valid usage; pointer-backed presence checks prevent the prior zero overwrite.
          round: 2
      findings:
        - id: BR-7
          severity: Critical
          title: The M1 Core Concepts inventory still names the wrong source for NativeRecordFact
          detail: 'This is the 2nd finding in family core-concepts-match-code. A sweep of all eight M1 Core Concepts rows found one contradiction: workshop/plans/000155-agent-session-tree-inventory-plan.md:21 locates NativeRecordFact in scan.go, while its declaration is in model.go:82. Do not fix only this row; state and enforce the rule that every concept name, kind, status, and path is mechanically checked against the tree, then append a plan revision recording the effective correction (ARCH-PURPOSE).'
          family: core-concepts-match-code
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-28T14:10:12-07:00"
      agent: codex
      dispose:
        - id: BR-7
          disposition: addressed
          note: The plan now locates NativeRecordFact in model.go, a bidirectional declaration contract checks every M1 concept field, and restoring the stale scan.go path makes that test fail.
          round: 3
      boundary: M1
      blocked: false
    - "n": 4
      timestamp: "2026-08-28T15:49:52-07:00"
      agent: codex
      findings:
        - id: BR-8
          severity: Critical
          title: Plain restart can resume a stale config while the current typed launch is provisional
          detail: 'ARCH-PURPOSE: cmd/internal/launcher/markers.go:87-90 falls back from the marker''s established-ledger session ID to saved.SessionID. The M2 contract requires an intentionally fresh restart while the latest typed launch is provisional; a stale compatibility cache must never restore an older root. Remove the session-ID fallback while retaining saved arguments, and add a regression where a provisional typed launch coexists with stale config. The current marker test asserting that fallback must be reversed.'
          family: established-binding-is-sole-recovery-authority
          round: 4
        - id: BR-9
          severity: Critical
          title: Unrecognized and unreadable evidence is still silently discarded
          detail: 'This is the 2nd finding in family diagnostic-registry-single-source. Do not fix only one site: state and enforce the class rule that every unrecognized versioned shape and failed evidence read produces a registry-backed diagnostic. The current sweep finds cmd/internal/sessioninventory/event.go:316-332 silently ignoring an unknown Muse run-event kind, plus cmd/internal/sessioninventory/pair_inventory.go:74-90 silently continuing after unreadable Pair logs/configs. Add tests for each adapter default and each Pair-artifact read boundary.'
          family: diagnostic-registry-single-source
          round: 4
        - id: BR-10
          severity: Critical
          title: Diagnostic ordering puts a null agent first despite schema v1 requiring null last
          detail: This is the 2nd finding in family documented-total-order. cmd/internal/sessioninventory/order.go:168-178 compares Agent as its raw empty string, which sorts before named agents; the documented tuple says agent null last. State the general nullable-comparator rule, sweep every nullable component, and add an exhaustive comparator test rather than patching this field alone.
          family: documented-total-order
          round: 4
        - id: BR-11
          severity: Critical
          title: A valid Markdown horizontal rule inside authored text makes the entire round suffix unusable
          detail: cmd/internal/sessioninventory/pairfacts.go:58-90 treats the first blank-line-delimited horizontal rule as the entry terminator. SessionLogStore permits the same bytes inside an authored body, so a prompt containing before, a Markdown horizontal rule, and after is persisted successfully but ParsePairLog rejects the remainder as a missing timestamp header. Live and offline correlation then remain provisional. Make framing round-trip valid authored Markdown and add a store-to-parser regression.
          family: authored-log-framing-round-trips
          round: 4
        - id: BR-12
          severity: Important
          title: README documentation is missing for the new public session-inventory command
          detail: The range adds pair session-inventory with user-facing flags and exit behavior, but README.md is unchanged. Document the command, its human/JSON and conformance modes, and the provisional versus established meaning.
          family: public-cli-readme
          round: 4
        - id: BR-13
          severity: Important
          title: Task 7 claims full result-matrix privacy goldens but tests cover only a subset
          detail: cmd/internal/sessioninventory/runcli_test.go:13-82 exercises six cases and cmd/internal/sessioninventory/render_test.go:52-72 contains only empty-output goldens. The plan explicitly requires byte goldens for every normal/conformance result row, including partial scans, schema drift, privacy failure with zero stdout, and serialization failure. Add the missing executable cases before keeping Task 7 checked.
          family: cli-result-matrix-is-executable
          round: 4
      boundary: M2
      blocked: true
    - "n": 5
      timestamp: "2026-08-28T16:02:09-07:00"
      agent: codex
      dispose:
        - id: BR-8
          disposition: addressed
          note: Empty-marker restart now drops stale config authority, and the regression fails against the former saved-session fallback.
          round: 5
        - id: BR-9
          disposition: addressed
          note: Unknown Muse run events become near-misses and all three Pair artifact read boundaries emit registry-backed storage_unreadable diagnostics with executable coverage.
          round: 5
        - id: BR-10
          disposition: addressed
          note: Agent, native ID, path, and source-ref nullability now use null-last projections with exhaustive equal-prefix comparisons.
          round: 5
        - id: BR-11
          disposition: addressed
          note: Byte-counted framing round-trips authored horizontal rules through SessionLogStore and ParsePairLog.
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: README prose exists, but no regression fails when the new public command documentation is removed.
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: The CLI branches are exercised, but complete byte goldens still do not exist for the promised normal and conformance result matrix.
          round: 5
      findings:
        - id: BR-14
          severity: Critical
          title: The authoritative review window contains no changes
          detail: Base and head are both 528a730, and the required stat and name-status recipes are empty. Re-run with a base preceding the M2 implementation or finding-closure commits so the anti-collusion review can inspect the actual delta.
          family: boundary-review-window-captures-delta
          round: 5
        - id: BR-15
          severity: Critical
          title: M2 carries closed project metadata before the milestone is checked
          detail: 'workshop/projects/couch.md leaves pair#155 M2 unchecked at line 173 but records actual and closed values at lines 399-400, causing TestUncheckedProjectMilestoneHasNoClosedMetadata to fail. This is the 2nd finding in family repository-contracts-stay-green. Do not fix only this instance: enforce that issue, plan, and project closure metadata changes only through the successful close-gate transaction; the current sweep measured one violation.'
          family: repository-contracts-stay-green
          round: 5
      boundary: M2
      blocked: true
---

# Gate ledger — pair#155 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T13:48:39-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `repository-contracts-stay-green` New inventory sources leave deterministic repository contract tests failing
  artifactpath coverage and the issue-149 source-set contract both fail at the pinned head; classify the new sources and update the governed catalog before closing M1.
- **BR-2** [Critical] `core-concepts-match-code` M1 omits the promised ParentEdge and EdgeProvenance entities
  model.go exposes only ParentID on Node, while the Core Concepts table requires explicit edge and provenance entities; implement and projection-test them, and reconcile the renamed Fact/Node/Forest types through a plan revision.
- **BR-3** [Critical] `diagnostic-registry-single-source` Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
  Every diagnostic is currently warning, storage_absent is omitted, IDs include free-form detail, sorting ignores the required severity order, and duplicates are retained; centralize and exhaustively test the documented registry.
- **BR-4** [Critical] `documented-total-order` Equal timestamps are ordered by time source instead of native ID
  compareNativeTime inserts TimeSource into the node comparator even though the specified tuple falls through from time directly to native_id; add an equal-time mixed-source regression and implement the exact tuple.
- **BR-5** [Critical] `storage-boundary-regular-partial` Filesystem enumeration admits blocking special files and discards valid partial results
  ListFiles rejects symlinks but accepts FIFOs, sockets, and devices, while scannerFiles drops valid entries returned with non-listing walk errors; reject every non-regular candidate and retain partial facts with structured diagnostics.
- **BR-6** [Critical] `optional-metrics-require-presence` Missing Claude and Codex usage objects overwrite valid token usage with zero
  Value-typed nested usage structs cannot distinguish absent or null usage from a real zero; require explicit presence before accepting a record and test that missing records do not replace the last valid sample.

## Round 2 — 2026-08-28T14:05:07-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Both governed repository contract tests pass and directly detect removal of the new inventory sources from their catalogs.
- BR-2 — addressed — ParentEdge and EdgeProvenance exist, are populated by BuildForest, and are asserted through model and canonical projection tests.
- BR-3 — addressed — The exhaustive registry test pins every code and severity, detail-independent IDs, severity ordering, coalescing, and storage_absent behavior.
- BR-4 — addressed — The mixed-source equal-time regression requires native-ID ordering and compareNativeTime now ignores TimeSource after equal timestamps.
- BR-5 — addressed — Tests exercise a real FIFO, rejected symlink, retained regular entries, and valid facts returned alongside a generic partial-listing error.
- BR-6 — addressed — Claude and Codex regressions place absent usage after valid usage; pointer-backed presence checks prevent the prior zero overwrite.

### Raised

- **BR-7** [Critical] `core-concepts-match-code` The M1 Core Concepts inventory still names the wrong source for NativeRecordFact
  This is the 2nd finding in family core-concepts-match-code. A sweep of all eight M1 Core Concepts rows found one contradiction: workshop/plans/000155-agent-session-tree-inventory-plan.md:21 locates NativeRecordFact in scan.go, while its declaration is in model.go:82. Do not fix only this row; state and enforce the rule that every concept name, kind, status, and path is mechanically checked against the tree, then append a plan revision recording the effective correction (ARCH-PURPOSE).

## Round 3 — 2026-08-28T14:10:12-07:00 (codex) — passed

### Disposed

- BR-7 — addressed — The plan now locates NativeRecordFact in model.go, a bidirectional declaration contract checks every M1 concept field, and restoring the stale scan.go path makes that test fail.

## Round 4 — 2026-08-28T15:49:52-07:00 (codex) — BLOCKED

### Raised

- **BR-8** [Critical] `established-binding-is-sole-recovery-authority` Plain restart can resume a stale config while the current typed launch is provisional
  ARCH-PURPOSE: cmd/internal/launcher/markers.go:87-90 falls back from the marker's established-ledger session ID to saved.SessionID. The M2 contract requires an intentionally fresh restart while the latest typed launch is provisional; a stale compatibility cache must never restore an older root. Remove the session-ID fallback while retaining saved arguments, and add a regression where a provisional typed launch coexists with stale config. The current marker test asserting that fallback must be reversed.
- **BR-9** [Critical] `diagnostic-registry-single-source` Unrecognized and unreadable evidence is still silently discarded
  This is the 2nd finding in family diagnostic-registry-single-source. Do not fix only one site: state and enforce the class rule that every unrecognized versioned shape and failed evidence read produces a registry-backed diagnostic. The current sweep finds cmd/internal/sessioninventory/event.go:316-332 silently ignoring an unknown Muse run-event kind, plus cmd/internal/sessioninventory/pair_inventory.go:74-90 silently continuing after unreadable Pair logs/configs. Add tests for each adapter default and each Pair-artifact read boundary.
- **BR-10** [Critical] `documented-total-order` Diagnostic ordering puts a null agent first despite schema v1 requiring null last
  This is the 2nd finding in family documented-total-order. cmd/internal/sessioninventory/order.go:168-178 compares Agent as its raw empty string, which sorts before named agents; the documented tuple says agent null last. State the general nullable-comparator rule, sweep every nullable component, and add an exhaustive comparator test rather than patching this field alone.
- **BR-11** [Critical] `authored-log-framing-round-trips` A valid Markdown horizontal rule inside authored text makes the entire round suffix unusable
  cmd/internal/sessioninventory/pairfacts.go:58-90 treats the first blank-line-delimited horizontal rule as the entry terminator. SessionLogStore permits the same bytes inside an authored body, so a prompt containing before, a Markdown horizontal rule, and after is persisted successfully but ParsePairLog rejects the remainder as a missing timestamp header. Live and offline correlation then remain provisional. Make framing round-trip valid authored Markdown and add a store-to-parser regression.
- **BR-12** [Important] `public-cli-readme` README documentation is missing for the new public session-inventory command
  The range adds pair session-inventory with user-facing flags and exit behavior, but README.md is unchanged. Document the command, its human/JSON and conformance modes, and the provisional versus established meaning.
- **BR-13** [Important] `cli-result-matrix-is-executable` Task 7 claims full result-matrix privacy goldens but tests cover only a subset
  cmd/internal/sessioninventory/runcli_test.go:13-82 exercises six cases and cmd/internal/sessioninventory/render_test.go:52-72 contains only empty-output goldens. The plan explicitly requires byte goldens for every normal/conformance result row, including partial scans, schema drift, privacy failure with zero stdout, and serialization failure. Add the missing executable cases before keeping Task 7 checked.

## Round 5 — 2026-08-28T16:02:09-07:00 (codex) — BLOCKED

### Disposed

- BR-8 — addressed — Empty-marker restart now drops stale config authority, and the regression fails against the former saved-session fallback.
- BR-9 — addressed — Unknown Muse run events become near-misses and all three Pair artifact read boundaries emit registry-backed storage_unreadable diagnostics with executable coverage.
- BR-10 — addressed — Agent, native ID, path, and source-ref nullability now use null-last projections with exhaustive equal-prefix comparisons.
- BR-11 — addressed — Byte-counted framing round-trips authored horizontal rules through SessionLogStore and ParsePairLog.
- BR-12 — not-addressed — README prose exists, but no regression fails when the new public command documentation is removed.
- BR-13 — not-addressed — The CLI branches are exercised, but complete byte goldens still do not exist for the promised normal and conformance result matrix.

### Raised

- **BR-14** [Critical] `boundary-review-window-captures-delta` The authoritative review window contains no changes
  Base and head are both 528a730, and the required stat and name-status recipes are empty. Re-run with a base preceding the M2 implementation or finding-closure commits so the anti-collusion review can inspect the actual delta.
- **BR-15** [Critical] `repository-contracts-stay-green` M2 carries closed project metadata before the milestone is checked
  workshop/projects/couch.md leaves pair#155 M2 unchecked at line 173 but records actual and closed values at lines 399-400, causing TestUncheckedProjectMilestoneHasNoClosedMetadata to fail. This is the 2nd finding in family repository-contracts-stay-green. Do not fix only this instance: enforce that issue, plan, and project closure metadata changes only through the successful close-gate transaction; the current sweep measured one violation.

## Open findings

- **BR-12** [Important] `public-cli-readme` README documentation is missing for the new public session-inventory command
- **BR-13** [Important] `cli-result-matrix-is-executable` Task 7 claims full result-matrix privacy goldens but tests cover only a subset
- **BR-14** [Critical] `boundary-review-window-captures-delta` The authoritative review window contains no changes
- **BR-15** [Critical] `repository-contracts-stay-green` M2 carries closed project metadata before the milestone is checked
