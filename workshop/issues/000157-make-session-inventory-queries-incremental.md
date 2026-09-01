---
id: 000157
status: open
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
---

# Make session inventory queries incremental

## Problem

`pair session-inventory` reconstructs the complete native-session forest by
calling the full scanners on every invocation. It therefore rereads transcript
bodies, including hundreds of megabytes of unchanged history, even though Pair
already maintains a durable session-inventory catalog with artifact
fingerprints, parser offsets, scanner state, and extracted facts.

The watcher and targeted binding queries use that catalog incrementally, but
the public tree query bypasses it. This violates the expected operating
envelope for an inventory that may back interactive session selection
(`ARCH-CONSTRAINTS`).

## Spec

- Make the public `pair session-inventory` query reconcile native artifact
  metadata against durable catalog state before selecting content work.
- Reuse the existing catalog/scanner authority rather than creating a parallel
  cache or a second definition of transcript identity (`ARCH-DRY`).
- For an unchanged recognized artifact, reuse its authorized facts without
  reading its body.
- For a trusted append-only artifact generation, read and parse only bytes
  after the prior parser-complete offset.
- Revalidate replacements, truncations, same-size mutations, scanner-schema
  changes, and provider-contract changes; fail closed on disputed evidence.
- Publish catalog advances atomically and safely under concurrent readers and
  writers. A failed publication must not make uncommitted scanner results
  authoritative.
- Preserve the current human and JSON inventory contracts: complete recursive
  forests, correlations, ambiguities, and diagnostics. Incrementality must not
  turn the command into a targeted-owner-only view.
- Keep the warm unchanged path metadata-bound. It may enumerate/stat candidate
  files, but it must perform zero transcript-body reads. Any interactive Couch
  consumer must not block its first frame on optional refresh work.

## Done when

- Instrumented tests prove a second unchanged full-inventory query performs no
  transcript-body reads and produces the same forest as a clean scan.
- Append tests prove only the appended suffix is read and the resulting forest
  matches a clean scan for every supported append-only provider.
- Mutation, replacement, truncation, schema-change, corrupt-catalog, and
  concurrent-publication tests prove safe revalidation or fail-closed behavior.
- A representative warm benchmark covers at most 100 transcript nodes and
  reports metadata operations, bytes read, and wall latency; the unchanged
  content-read count is zero.
- `pair session-inventory`, `--json`, agent filtering, and both scope modes keep
  their documented output semantics.

## Plan

- [ ] Define catalog authority for complete-forest queries, including how
  `--scope current` and `--scope all` reuse and publish catalog generations.
- [ ] Drive reconciliation and delta validation through the existing pure
  catalog/scanner core, adding complete-inventory orchestration at the IO shell.
- [ ] Wire the CLI to load, advance, publish, and render the reconciled
  inventory without changing its output schema.
- [ ] Add equivalence, zero-body-read, append, invalidation, corruption,
  concurrency, and bounded benchmark coverage.

## Log

### 2026-08-30

Inspection found that `runCLIOptionsWithRenderers` calls
`InventoryWithRuntime`, which invokes the full scanners directly. The durable
scoped `session-inventory-catalog.json` is currently consumed by the session
watcher and targeted `QuerySession` path, but not by the public complete-tree
query. Captured as a separate issue from #151 so the hierarchical Couch menu
does not absorb a storage/scanner-authority redesign.

## Revisions

### 2026-08-30T22:03:00-07:00 — record the longer product arc

Reason: the incremental query is foundational infrastructure for a broader
cross-agent local thread inventory, not merely a Pair performance repair.

Delta: #157 should preserve an evolution path from discovery and incremental
indexing toward host-wide management of native transcript trees, including
future provider-aware archive, restore, retention, and garbage collection.
That inventory should cover locally discovered agent threads whether or not
Pair launched them. Pair is the current home and a consumer of the capability,
but lifecycle management is not part of Pair's narrow TTY-switching purpose and
may eventually warrant its own tool boundary. This records direction only;
archive and garbage-collection semantics are deliberately not designed in this
issue now.
