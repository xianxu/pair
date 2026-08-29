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
    - "n": 6
      timestamp: "2026-08-28T16:13:29-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: addressed
          note: README lines 376-392 document the public command, flags, exit behavior, privacy, and binding states; TestREADMEDocumentsSessionInventoryContract pins the required surface.
          round: 6
        - id: BR-13
          disposition: addressed
          note: The result and failure matrix tests now compare exact exit, stdout, and stderr bytes for normal, partial, conformance, usage, fatal, privacy, serialization, and writer branches against checked-in goldens.
          round: 6
        - id: BR-14
          disposition: not-addressed
          note: The range is nonempty but still excludes every M2 implementation and test commit; rerun from the previous M1 boundary 4f151b037dc2d0f20d413c5af6a2353e131cec8e.
          round: 6
        - id: BR-15
          disposition: addressed
          note: The Couch checkbox and closure detail now move together, and TestUncheckedProjectMilestoneHasNoClosedMetadata passes while directly rejecting the prior unchecked-plus-closed state.
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-28T16:24:46-07:00"
      agent: codex
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: The launcher marker instance is fixed and tested, but sessioninventory still selects config evidence when LaunchPresent is true and no round or binding exists, re-establishing stale compatibility state during a provisional typed launch.
          round: 7
        - id: BR-9
          disposition: not-addressed
          note: The adapter and CLI cases are tested, but watcher Pair-log read failures, owner-mismatched typed ledger rows, and scanner-unknown ledger/config native IDs are still silently discarded.
          round: 7
        - id: BR-10
          disposition: addressed
          note: Nullable diagnostic coordinates use the shared null-last projection, with exhaustive equal-prefix comparator coverage.
          round: 7
        - id: BR-11
          disposition: not-addressed
          note: The Go store/parser regression proves byte-counted framing, but Neovim history parsing and rewriting still use delimiter framing and cannot consume new entries or authored separator text.
          round: 7
        - id: BR-12
          disposition: addressed
          note: README documents the command, modes, lifecycle statuses, and exits, with an executable documentation regression.
          round: 7
        - id: BR-13
          disposition: addressed
          note: Checked-in result and failure matrix goldens now pin exact exit, stdout, and stderr bytes across the claimed CLI branches.
          round: 7
        - id: BR-14
          disposition: addressed
          note: The authoritative range is non-empty and contains 94 changed files with 6,422 additions and 1,668 deletions.
          round: 7
        - id: BR-15
          disposition: addressed
          note: Couch now carries the M2 checkbox, actual, date, and detail as one consistent gate-preflight state.
          round: 7
      findings:
        - id: BR-16
          severity: Critical
          title: Unrelated open files suppress portable causal-round establishment
          detail: 'ARCH-PURPOSE: cmd/internal/sessionwatch/run.go reports process evidence available whenever the Pair process tree has any open file, even when none maps to a scanner-authorized native artifact. Run then filters every otherwise unique causal round. Return available=true only for relevant native-artifact evidence, and add a stateful-fake regression with unrelated open files plus one globally unique completed round.'
          family: usable-process-evidence-only
          round: 7
        - id: BR-17
          severity: Critical
          title: Public evidence JSON does not match the documented schema-v1 contract
          detail: 'ARCH-PURPOSE: the internal Evidence type adds public root_node_id, and render.go copies it directly into the public DTO. Ledger/config evidence also renders required position and fingerprint arrays as null. Introduce an exact evidenceV1 projection with non-nil arrays and test it against an independent contract golden.'
          family: schema-v1-output-is-exact
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-28T16:44:34-07:00"
      agent: codex
      dispose:
        - id: BR-8
          disposition: addressed
          note: Provisional plain restart now drops stale config identity, and reverting the behavior makes TestPlanRestart fail.
          round: 8
        - id: BR-9
          disposition: not-addressed
          note: Named Muse and read-boundary cases are fixed, but valid v1 ledger rows with unsupported agents are still silently discarded at pair_inventory.go:69-70.
          round: 8
        - id: BR-11
          disposition: addressed
          note: Versioned byte-count framing round-trips arbitrary authored Markdown; reverting the writer to legacy framing makes the store-to-parser regression fail.
          round: 8
        - id: BR-16
          disposition: addressed
          note: Only scanner-authorized open artifacts activate process corroboration; reverting availability to any open file makes the regression hang instead of establishing the unique round.
          round: 8
        - id: BR-17
          disposition: addressed
          note: The exact public evidence projection removes internal root_node_id and forces arrays; reintroducing the field makes the independent golden fail.
          round: 8
      boundary: M2
      blocked: true
    - "n": 9
      timestamp: "2026-08-28T16:55:37-07:00"
      agent: codex
      dispose:
        - id: BR-9
          disposition: addressed
          note: Muse's unknown run-event default is pinned as near_miss and reaches a registry-backed turn_unusable diagnostic; the CLI golden separately requires diagnostics at all three Pair ledger/log/config read boundaries.
          round: 9
      boundary: M2
      blocked: false
    - "n": 10
      timestamp: "2026-08-28T17:49:56-07:00"
      agent: codex
      findings:
        - id: BR-18
          severity: Critical
          title: Missing process identity prevents portable causal-round establishment
          detail: 'This is the 2nd finding in family `usable-process-evidence-only`. A current PID file plus an unavailable identity token returns before causal matching. State the class rule: process evidence may constrain a match only when usable; its absence must never suppress a unique completed round.'
          family: usable-process-evidence-only
          round: 10
        - id: BR-19
          severity: Critical
          title: Whole-file limits make valid long transcripts unusable
          detail: Native events, token usage, and transcript consumers read the entire artifact through a 32 MiB cap although the contract bounds individual JSONL records. Four installed valid transcripts already exceed that cap, so the affected roots cannot establish or serve migrated consumers.
          family: bounded-record-streaming
          round: 10
        - id: BR-20
          severity: Critical
          title: Partial Pair-artifact enumeration discards valid evidence
          detail: 'This is the 2nd finding in family `storage-boundary-regular-partial`. `RecoverPairBindings` treats every non-absence listing error as fatal even when `ListFiles` returned valid ledger/config/log entries. Apply the class rule to every storage root: retain regular partial results and diagnose rejected entries.'
          family: storage-boundary-regular-partial
          round: 10
        - id: BR-21
          severity: Important
          title: Valid compatibility ledger rows are reported as malformed
          detail: Every launch still appends a legacy `LedgerEntry` before its typed launch row, but the inventory parser classifies every non-typed row as malformed. Mixed ledgers need one shared classification that distinguishes supported compatibility rows, typed authority, and genuinely corrupt rows.
          family: mixed-ledger-formats-are-classified
          round: 10
      blocked: true
    - "n": 11
      timestamp: "2026-08-28T18:07:18-07:00"
      agent: codex
      dispose:
        - id: BR-18
          disposition: addressed
          note: The watcher regression directly establishes a unique completed round when a current PID has no usable identity token; restoring the former early return would prevent the asserted binding.
          round: 11
        - id: BR-19
          disposition: not-addressed
          note: Event and usage paths have long-record tests, but the migrated slug transcript consumer still reconstructs and parses the whole artifact without a failing long-transcript regression; the shared helper also accepts unterminated final records contrary to the bounded-record contract.
          round: 11
        - id: BR-20
          disposition: addressed
          note: The QuerySession regression returns an established binding from valid files alongside a rejected listing entry and requires the corresponding diagnostic; the former fatal return would fail it.
          round: 11
        - id: BR-21
          disposition: addressed
          note: A valid legacy row followed by typed authority is classified without a malformed diagnostic, and the shared parser is used by both inventory and launcher.
          round: 11
      findings:
        - id: BR-22
          severity: Critical
          title: Slug remains a second native transcript parser
          detail: 'ARCH-DRY and ARCH-PURPOSE: slugcmd reads the complete transcript and maintains separate Claude, Codex, Agy, and Muse adapters, including an unknown-agent fallback to Claude. The issue explicitly requires every native parser consumer to derive from sessioninventory; expose a bounded shared text-event projection, migrate slug to it, and enforce the class in the shadow sweep.'
          family: native-record-parsing-is-single-source
          round: 11
        - id: BR-23
          severity: Important
          title: Compatibility classification admits malformed and unsupported rows
          detail: This is the 2nd finding in family `mixed-ledger-formats-are-classified`. The classifier treats any single JSON object with a nonempty agent and no v or kind as compatible, so partial, unknown-field, and unsupported-agent rows escape malformed diagnostics and can enter launcher history. State the exhaustive typed versus exact-legacy versus malformed rule and test the complete classification matrix in both consumers.
          family: mixed-ledger-formats-are-classified
          round: 11
      blocked: true
    - "n": 12
      timestamp: "2026-08-28T18:34:46-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The repository-wide Go suite again fails its declaration-disposition source contract after slug_test.go changed without entering the hand-maintained catalog.
          round: 12
        - id: BR-19
          disposition: addressed
          note: Native events, usage, and slug text projection stream arbitrary-length JSONL with per-record bounds; restoring a 32 MiB cutoff makes the long-transcript regressions fail.
          round: 12
        - id: BR-22
          disposition: addressed
          note: Slug now consumes TextEventWindowForRoot, the duplicate four-agent adapters are deleted, and the shadow-sweep regression fails if a native parser is reintroduced.
          round: 12
        - id: BR-23
          disposition: not-addressed
          note: Legacy shapes are strict, but an unsupported typed ledger row still becomes a launcher LedgerEntry and can influence history or agent inference.
          round: 12
      blocked: true
    - "n": 13
      timestamp: "2026-08-28T21:03:26-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Artifact-path and immutable issue-149 source-set contracts pass; the prior moving-HEAD implementation fails in a scratch reproduction.
          round: 13
        - id: BR-23
          disposition: not-addressed
          note: Partial, unknown-field, and unsupported-agent cases are fixed, but duplicate-key typed and compatibility rows are still accepted rather than classified malformed.
          round: 13
      blocked: false
    - "n": 14
      timestamp: "2026-08-28T21:10:31-07:00"
      agent: codex
      dispose:
        - id: BR-23
          disposition: not-addressed
          note: Duplicate, unknown, unsupported, partial, and trailing-value cases are covered, but missing or null nested event_position and explicit null legacy_import fields are still accepted instead of classified malformed.
          round: 14
      blocked: false
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

## Round 6 — 2026-08-28T16:13:29-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — addressed — README lines 376-392 document the public command, flags, exit behavior, privacy, and binding states; TestREADMEDocumentsSessionInventoryContract pins the required surface.
- BR-13 — addressed — The result and failure matrix tests now compare exact exit, stdout, and stderr bytes for normal, partial, conformance, usage, fatal, privacy, serialization, and writer branches against checked-in goldens.
- BR-14 — not-addressed — The range is nonempty but still excludes every M2 implementation and test commit; rerun from the previous M1 boundary 4f151b037dc2d0f20d413c5af6a2353e131cec8e.
- BR-15 — addressed — The Couch checkbox and closure detail now move together, and TestUncheckedProjectMilestoneHasNoClosedMetadata passes while directly rejecting the prior unchecked-plus-closed state.

## Round 7 — 2026-08-28T16:24:46-07:00 (codex) — BLOCKED

### Disposed

- BR-8 — not-addressed — The launcher marker instance is fixed and tested, but sessioninventory still selects config evidence when LaunchPresent is true and no round or binding exists, re-establishing stale compatibility state during a provisional typed launch.
- BR-9 — not-addressed — The adapter and CLI cases are tested, but watcher Pair-log read failures, owner-mismatched typed ledger rows, and scanner-unknown ledger/config native IDs are still silently discarded.
- BR-10 — addressed — Nullable diagnostic coordinates use the shared null-last projection, with exhaustive equal-prefix comparator coverage.
- BR-11 — not-addressed — The Go store/parser regression proves byte-counted framing, but Neovim history parsing and rewriting still use delimiter framing and cannot consume new entries or authored separator text.
- BR-12 — addressed — README documents the command, modes, lifecycle statuses, and exits, with an executable documentation regression.
- BR-13 — addressed — Checked-in result and failure matrix goldens now pin exact exit, stdout, and stderr bytes across the claimed CLI branches.
- BR-14 — addressed — The authoritative range is non-empty and contains 94 changed files with 6,422 additions and 1,668 deletions.
- BR-15 — addressed — Couch now carries the M2 checkbox, actual, date, and detail as one consistent gate-preflight state.

### Raised

- **BR-16** [Critical] `usable-process-evidence-only` Unrelated open files suppress portable causal-round establishment
  ARCH-PURPOSE: cmd/internal/sessionwatch/run.go reports process evidence available whenever the Pair process tree has any open file, even when none maps to a scanner-authorized native artifact. Run then filters every otherwise unique causal round. Return available=true only for relevant native-artifact evidence, and add a stateful-fake regression with unrelated open files plus one globally unique completed round.
- **BR-17** [Critical] `schema-v1-output-is-exact` Public evidence JSON does not match the documented schema-v1 contract
  ARCH-PURPOSE: the internal Evidence type adds public root_node_id, and render.go copies it directly into the public DTO. Ledger/config evidence also renders required position and fingerprint arrays as null. Introduce an exact evidenceV1 projection with non-nil arrays and test it against an independent contract golden.

## Round 8 — 2026-08-28T16:44:34-07:00 (codex) — BLOCKED

### Disposed

- BR-8 — addressed — Provisional plain restart now drops stale config identity, and reverting the behavior makes TestPlanRestart fail.
- BR-9 — not-addressed — Named Muse and read-boundary cases are fixed, but valid v1 ledger rows with unsupported agents are still silently discarded at pair_inventory.go:69-70.
- BR-11 — addressed — Versioned byte-count framing round-trips arbitrary authored Markdown; reverting the writer to legacy framing makes the store-to-parser regression fail.
- BR-16 — addressed — Only scanner-authorized open artifacts activate process corroboration; reverting availability to any open file makes the regression hang instead of establishing the unique round.
- BR-17 — addressed — The exact public evidence projection removes internal root_node_id and forces arrays; reintroducing the field makes the independent golden fail.

## Round 9 — 2026-08-28T16:55:37-07:00 (codex) — passed

### Disposed

- BR-9 — addressed — Muse's unknown run-event default is pinned as near_miss and reaches a registry-backed turn_unusable diagnostic; the CLI golden separately requires diagnostics at all three Pair ledger/log/config read boundaries.

## Round 10 — 2026-08-28T17:49:56-07:00 (codex) — BLOCKED

### Raised

- **BR-18** [Critical] `usable-process-evidence-only` Missing process identity prevents portable causal-round establishment
  This is the 2nd finding in family `usable-process-evidence-only`. A current PID file plus an unavailable identity token returns before causal matching. State the class rule: process evidence may constrain a match only when usable; its absence must never suppress a unique completed round.
- **BR-19** [Critical] `bounded-record-streaming` Whole-file limits make valid long transcripts unusable
  Native events, token usage, and transcript consumers read the entire artifact through a 32 MiB cap although the contract bounds individual JSONL records. Four installed valid transcripts already exceed that cap, so the affected roots cannot establish or serve migrated consumers.
- **BR-20** [Critical] `storage-boundary-regular-partial` Partial Pair-artifact enumeration discards valid evidence
  This is the 2nd finding in family `storage-boundary-regular-partial`. `RecoverPairBindings` treats every non-absence listing error as fatal even when `ListFiles` returned valid ledger/config/log entries. Apply the class rule to every storage root: retain regular partial results and diagnose rejected entries.
- **BR-21** [Important] `mixed-ledger-formats-are-classified` Valid compatibility ledger rows are reported as malformed
  Every launch still appends a legacy `LedgerEntry` before its typed launch row, but the inventory parser classifies every non-typed row as malformed. Mixed ledgers need one shared classification that distinguishes supported compatibility rows, typed authority, and genuinely corrupt rows.

## Round 11 — 2026-08-28T18:07:18-07:00 (codex) — BLOCKED

### Disposed

- BR-18 — addressed — The watcher regression directly establishes a unique completed round when a current PID has no usable identity token; restoring the former early return would prevent the asserted binding.
- BR-19 — not-addressed — Event and usage paths have long-record tests, but the migrated slug transcript consumer still reconstructs and parses the whole artifact without a failing long-transcript regression; the shared helper also accepts unterminated final records contrary to the bounded-record contract.
- BR-20 — addressed — The QuerySession regression returns an established binding from valid files alongside a rejected listing entry and requires the corresponding diagnostic; the former fatal return would fail it.
- BR-21 — addressed — A valid legacy row followed by typed authority is classified without a malformed diagnostic, and the shared parser is used by both inventory and launcher.

### Raised

- **BR-22** [Critical] `native-record-parsing-is-single-source` Slug remains a second native transcript parser
  ARCH-DRY and ARCH-PURPOSE: slugcmd reads the complete transcript and maintains separate Claude, Codex, Agy, and Muse adapters, including an unknown-agent fallback to Claude. The issue explicitly requires every native parser consumer to derive from sessioninventory; expose a bounded shared text-event projection, migrate slug to it, and enforce the class in the shadow sweep.
- **BR-23** [Important] `mixed-ledger-formats-are-classified` Compatibility classification admits malformed and unsupported rows
  This is the 2nd finding in family `mixed-ledger-formats-are-classified`. The classifier treats any single JSON object with a nonempty agent and no v or kind as compatible, so partial, unknown-field, and unsupported-agent rows escape malformed diagnostics and can enter launcher history. State the exhaustive typed versus exact-legacy versus malformed rule and test the complete classification matrix in both consumers.

## Round 12 — 2026-08-28T18:34:46-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The repository-wide Go suite again fails its declaration-disposition source contract after slug_test.go changed without entering the hand-maintained catalog.
- BR-19 — addressed — Native events, usage, and slug text projection stream arbitrary-length JSONL with per-record bounds; restoring a 32 MiB cutoff makes the long-transcript regressions fail.
- BR-22 — addressed — Slug now consumes TextEventWindowForRoot, the duplicate four-agent adapters are deleted, and the shadow-sweep regression fails if a native parser is reintroduced.
- BR-23 — not-addressed — Legacy shapes are strict, but an unsupported typed ledger row still becomes a launcher LedgerEntry and can influence history or agent inference.

## Round 13 — 2026-08-28T21:03:26-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Artifact-path and immutable issue-149 source-set contracts pass; the prior moving-HEAD implementation fails in a scratch reproduction.
- BR-23 — not-addressed — Partial, unknown-field, and unsupported-agent cases are fixed, but duplicate-key typed and compatibility rows are still accepted rather than classified malformed.

## Round 14 — 2026-08-28T21:10:31-07:00 (codex) — passed

### Disposed

- BR-23 — not-addressed — Duplicate, unknown, unsupported, partial, and trailing-value cases are covered, but missing or null nested event_position and explicit null legacy_import fields are still accepted instead of classified malformed.

## Open findings

- **BR-23** [Important] `mixed-ledger-formats-are-classified` Compatibility classification admits malformed and unsupported rows
