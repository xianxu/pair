---
id: 000156
status: working
deps: []
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
started: 2026-08-29T06:33:52-07:00
---

# incremental native session inventory

## Problem

#155 made the native session inventory authoritative for launch, watcher, UI,
and recovery decisions, but its production paths rebuild that inventory by
walking and parsing the complete native transcript corpus on every query. On
the operator's Claude corpus this means 1,573 files / 358 MiB: one inventory
query takes about 13.2 seconds (9.3 seconds user, 4.4 seconds system), including
5.4 seconds spent spawning one `stat` subprocess per file.

That full scan is synchronously repeated in latency-sensitive paths:

- `Alt+X` performs three inventory queries before it paints the confirmation;
- fresh Claude launch checks established ownership, checks a minted UUID for a
  collision, captures a full launch baseline, and then starts a watcher that
  repeats the same whole-corpus scan;
- launch baseline and watcher event projection reread every root transcript to
  normalize historical events even though only post-launch bytes can establish
  the new binding.

The typed-ledger compatibility projection also discards display metadata. A
typed launch wins authority over its compatibility row but supplies no
`repo_name`, so the session picker renders an otherwise known historical row as
`?/1 claude` instead of `pair/1 claude`.

## Spec

### Performance and ordering contract

- `Alt+X` must paint an interactive confirmation before starting any inventory
  or activity enrichment. A slow, missing, or failed inventory query cannot
  delay or suppress the modal; the age/idle hint may be omitted when it is not
  already available.
- A launch with an empty cache and a launch with a warm unchanged cache must
  complete all pre-Zellij native inventory work within one second on a fixture
  matching the observed 1,573-file corpus. Neither path may read unchanged
  transcript bodies.
- A file that Pair has already categorized is not categorized again while its
  identity and parser schema remain unchanged.
- No latency-sensitive inventory path may spawn an external command per native
  file. Portable file metadata comes from the filesystem runtime directly.

### Persistent artifact catalog

Pair owns one versioned, persistent native-artifact catalog under the selected
Pair data scope. Each entry records the agent, storage root, relative path,
portable file identity/fingerprint, size, timestamps needed by the public
contract, scanner classification/facts, last complete byte offset, and the
scanner/parser schema version that produced it.

Catalog reconciliation is incremental and deterministic:

- unchanged identity + size + timestamps + schema reuses the cached facts;
- an append preserves classification and advances only from the prior complete
  byte offset;
- a new file is categorized once and read only when it can participate in the
  current launch/recovery observation;
- replacement, truncation, or schema-version change invalidates that entry,
  not the entire catalog;
- deletion removes the entry and its projected node;
- malformed or corrupt catalog state fails locally: rebuild the affected entry
  (or an unreadable catalog) without accepting stale authority;
- writes are atomic and readers see either the previous complete catalog or the
  next complete catalog.

The catalog is a reusable inventory seam, not a cache of rendered CLI output.
Forest/query/activity/launch/watcher consumers derive from the same catalog
facts (`ARCH-DRY`, `ARCH-PURPOSE`). Reconciliation and selection remain pure;
filesystem enumeration, targeted reads, metadata, and persistence stay behind
the injected runtime (`ARCH-PURE`). The existing stateful fake grows catalog,
append, replacement, truncation, deletion, and corruption behavior so tests use
the production boundary rather than stateless call mocks (`ARCH-MOCK`).

### Metadata-only launch baseline

A launch baseline snapshots the categorized root set and each root transcript's
last complete EOF/position without parsing pre-launch transcript content. It
does not need historical normalized events: by definition only bytes after the
launch boundary can establish the new causal round.

For an explicit resume or an already-established binding, Pair targets the one
ledger-named native root and checks that artifact through its scanner-owned
path/metadata validation. It does not rebuild every forest to rediscover a root
whose durable binding is already known. Full historical content validation is
reserved for the explicit diagnostic/conformance surface and never runs as a
side effect of launch, `Alt+X`, title/context polling, or ordinary owner lookup.

### Incremental watcher

The watcher compares the current catalog/filesystem snapshot with the durable
launch baseline. It visits only new, appended, replaced, or truncated candidate
artifacts and reads only bytes after their stored complete offset. For a new
candidate transcript it reads from byte zero only until it has enough evidence
to classify the artifact and establish or reject a completed causal round; it
does not normalize unrelated historical transcripts.

Polling frequency cannot multiply corpus work: an unchanged poll performs
metadata/catalog reconciliation only. After establishment, existing watcher
shutdown behavior remains unchanged. Partial listings, symlink rejection,
bounded records, malformed evidence diagnostics, exact completed-round gating,
and parent-edge semantics from #155 remain fail-safe.

### Query and UI projections

Owner, activity, existence, context, title, opener, review, slug, launcher, and
watcher paths use targeted catalog queries rather than whole-agent scans. A
consumer needing multiple values for one UI action requests one projection once
instead of starting parallel or repeated inventories.

`Alt+X` orders the modal before optional enrichment and never shells out to
`pair session-inventory` on the modal's critical path. Tests must block the
inventory seam indefinitely and still observe the confirmation first.

Typed ledger authority and compatibility display metadata are orthogonal. When
a typed current launch/binding wins ownership, the history projection retains
the newest compatible `repo_name`, `repo_root`, activity timestamp, and saved
args for display/launch compatibility without allowing those fields to select
the native root. Thus the observed `?/1 claude` row renders `pair/1 claude`.

## Done when

- `Alt+X` confirmation is observable before any inventory/activity work and
  remains usable when that work blocks or fails.
- Fresh and warm pre-Zellij inventory work complete within one second on a
  1,573-file unchanged-corpus regression fixture without reading transcript
  bodies.
- An unchanged artifact is categorized once across repeated inventories,
  queries, launches, and watcher polls.
- Append reads only the suffix; new, replaced, truncated, deleted, schema-stale,
  and corrupt-cache cases have deterministic regression coverage.
- Launch baseline and watcher never normalize pre-launch historical events;
  completed-round establishment still passes the #155 exact-correlation matrix.
- Per-file metadata collection uses no external `stat` process.
- Every native-session consumer is covered by a shadow-sweep contract that
  rejects reintroduced whole-agent scans on latency-sensitive paths.
- The historical picker renders the observed tag `1` as `pair/1 claude`, while
  typed ledger binding remains the sole root authority.
- Focused tests, the full Go suite with `-p 20`, Lua/shell integration tests,
  vet, and a real-data smoke all pass.

## Plan

- [ ] Define the versioned artifact catalog, fingerprints, and pure incremental
  reconciliation rules with stateful fake coverage.
- [ ] Replace whole-corpus scanner reconstruction with metadata-only discovery,
  cached facts, and targeted/suffix reads.
- [ ] Make launch baselines and watcher polling operate on catalog deltas and
  post-launch bytes only.
- [ ] Move `Alt+X` confirmation ahead of optional enrichment and migrate every
  latency-sensitive native-session consumer to targeted queries.
- [ ] Merge typed authority with compatibility display metadata and pin the
  `pair/1 claude` picker row.
- [ ] Add operation-count, one-second corpus, shadow-sweep, full-suite, and
  real-data verification; update atlas documentation.

## Log

### 2026-08-29

- Reproduced one Claude inventory at 13.16-13.29 seconds across six runs on
  1,573 files / 358 MiB; isolated 5.39 seconds to the per-file external `stat`
  process. Traced three synchronous queries before `Alt+X`, repeated full scans
  in launch/baseline/watch, and typed-ledger metadata loss behind `?/1 claude`.
- Operator approved a metadata/EOF baseline with persistent incremental
  categorization: never reread unchanged transcript bodies; inspect only new
  bytes relevant to the current round. `Alt+X` UI ordering is stricter than the
  one-second inventory budget: paint first and never block on enrichment.
