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
  matching the observed 1,573-file corpus. Neither path may parse unchanged
  transcript bodies. The deterministic gate is zero body-record reads plus
  O(number of directory entries) metadata operations; the one-second wall-clock
  check is a real-data smoke on the operator's observed machine, not a
  scheduler-sensitive unit-test oracle.
- A file that Pair has already categorized is not categorized again while its
  identity and parser schema remain unchanged.
- No latency-sensitive inventory path may spawn an external command per native
  file. Portable file metadata comes from the filesystem runtime directly.

### Persistent artifact catalog

Pair owns one versioned, persistent native-artifact catalog under the selected
Pair data scope. Each entry records the agent, storage root, relative path,
opaque runtime file identity, size, change token, timestamps needed by the
public contract, authorization state, scanner classification/facts, raw observed
offset, parser-complete offset, incremental scanner state, and the
scanner/parser schema version that produced it.

`FileIdentity` is an opaque runtime value built without content reads. On Unix
it includes device + inode plus the platform change token (`ctime` at nanosecond
precision); implementations on another platform must supply the equivalent
stable file ID + change token. Size and modification time remain separate
fields. If the runtime cannot obtain a stable identity/change token, continuity
is unproven: the entry is treated as replaced and cached authority is not used.
Path + size + mtime alone never proves continuity.

Catalog reconciliation is incremental and deterministic:

- unchanged identity + size + change token + timestamps + schema reuses the
  cached facts;
- an append preserves the prior authorization only while the agent-specific
  incremental validator accepts the suffix, and advances only from the prior
  parser-complete byte offset;
- a new file is categorized once and read only when it can participate in the
  current launch/recovery observation;
- replacement, truncation, or schema-version change invalidates that entry,
  not the entire catalog;
- deletion removes the entry and its projected node;
- malformed or corrupt catalog state fails locally. Startup falls back to an
  untrusted metadata-only exclusion snapshot; it does not synchronously rebuild
  the corpus or accept stale authority. Targeted use can rebuild one entry, and
  explicit diagnostic/conformance work may repair the complete catalog;
- writes serialize through one catalog-store lock. A writer locks, rereads the
  current generation, applies the pure reconciliation delta, atomically replaces
  and syncs the next generation, then unlocks. A generation mismatch retries
  from the reread state; lock/write/sync failure preserves the prior catalog and
  cannot publish new authority. Readers see either the previous complete
  generation or the next complete generation.

Each authorized entry persists the minimum scanner state needed to validate an
append without replaying history: expected native/root identity, root versus
descendant role, schema discriminator, chronology state, disputed flag, and
agent-specific first-record invariants. A suffix identity/role/schema
contradiction transitions the entry to disputed, retracts its cached root/parent
facts from projections, and emits the same #155 diagnostic class. Validation
state is versioned with the parser; an unknown state is unauthorized rather
than guessed.

The catalog is a reusable inventory seam, not a cache of rendered CLI output.
Forest/query/activity/launch/watcher consumers derive from the same catalog
facts (`ARCH-DRY`, `ARCH-PURPOSE`). Reconciliation and selection remain pure;
filesystem enumeration, targeted reads, metadata, and persistence stay behind
the injected runtime (`ARCH-PURE`). The existing stateful fake grows catalog,
append, replacement, truncation, deletion, and corruption behavior so tests use
the production boundary rather than stateless call mocks (`ARCH-MOCK`).

### Candidate discovery versus authorization

Metadata discovery produces candidates and exclusion watermarks, never scanner
authority. Scanner facts become authorized only through one of these paths:

| agent | cold candidate discovery | targeted authorization | new-launch observation |
|---|---|---|---|
| Claude | path shape yields possible root/subagent ID and raw size | an already-established typed binding remains durable authority; an explicit resume validates the one scanner-resolved UUID path and its bounded identifying record | read the new file from byte zero and incrementally validate every appended record until the completed round binds |
| Codex | rollout path yields a possible artifact and raw size | established binding remains authority; explicit resume targets the rollout and validates its required first `session_meta` | read the new rollout from byte zero, require `session_meta`, then validate appended events |
| Muse | session path yields a possible native ID and raw size | established binding remains authority; explicit resume validates the targeted identifying record | read the new session from byte zero and incrementally validate appended records |
| Agy | database file identity/change token yields an untrusted changed source | established binding remains authority; explicit resume performs one keyed, schema-checked SQLite query | query only rows changed after the launch watermark through the existing typed SQLite seam |

An established typed ledger binding was published only after prior scanner
authorization and is not discarded merely because a derived catalog is absent.
It must still resolve to an artifact matching the agent's scanner-owned target
shape; missing/replaced artifacts report absence and never fall back to another
root. Explicit resume is the only cold path that may synchronously inspect a
preexisting transcript body, and it inspects only the named artifact through
the bounded identifying-record seam—not the corpus. Unbound preexisting
candidates remain excluded until explicit diagnostic validation; they cannot
become launch bindings from metadata alone.

### Metadata-only launch baseline

A launch baseline snapshots an untrusted set of candidate artifact identities
and raw file sizes. Those are exclusion watermarks, not categorized roots or
scanner facts. It does not parse pre-launch transcript content or need
historical normalized events: by definition only wholly post-launch records can
establish the new causal round.

Raw launch boundary and parser-complete offset are distinct. For a preexisting
artifact the baseline stores raw size `B`. If that artifact later appends, the
watcher reads the single byte at `B-1` together with the suffix. When it is a
newline, parsing starts at `B`; otherwise the record crossing `B` began before
launch and is skipped through its terminating newline, with parsing starting at
the following byte. An empty file starts at zero. If the boundary byte or suffix
cannot be read consistently, that candidate contributes no round evidence.
Tests cover newline, incomplete-record, empty-file, truncation, and replacement
at the boundary so a wholly post-launch event is neither lost nor fabricated.

For an explicit resume or an already-established binding, Pair targets the one
ledger-named native root and checks that artifact through its scanner-owned
path/metadata validation. It does not rebuild every forest to rediscover a root
whose durable binding is already known. Full historical content validation is
reserved for the explicit diagnostic/conformance surface and never runs as a
side effect of launch, `Alt+X`, title/context polling, or ordinary owner lookup.

### Incremental watcher

The watcher compares the current catalog/filesystem snapshot with the durable
launch baseline. It visits only new, appended, replaced, or truncated candidate
artifacts and reads only eligible bytes after their raw launch boundary or
stored parser-complete offset. For a new
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
  real 1,573-file unchanged corpus without parsing transcript bodies; the
  deterministic fixture proves zero body-record reads and linear metadata work.
- An unchanged artifact is categorized once across repeated inventories,
  queries, launches, and watcher polls.
- Append reads only the suffix; new, replaced, truncated, deleted, schema-stale,
  and corrupt-cache cases have deterministic regression coverage.
- Cold-cache candidate discovery never confers authority; agent-specific
  explicit-resume, established-binding, new-file, and preexisting-exclusion
  cases pass the authorization matrix.
- A record split across the launch boundary cannot establish a round, while the
  first wholly post-launch record is retained when the old file ended in a
  newline.
- Concurrent catalog writers cannot lose a newer cursor, scanner state, or
  disputed transition.
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

## Revisions

### 2026-08-29 — cold baseline is exclusion, not authorization

Fresh-context spec review found that “categorized root set” incorrectly implied
metadata could recreate scanner authority, and that a raw EOF is not necessarily
a complete-record boundary. The spec now separates untrusted candidate/exclusion
watermarks from authorized facts, defines the agent-by-agent cold path, stores
raw and parser-complete offsets separately, handles records split across launch,
uses an opaque device/inode/change-token identity with fail-closed fallback,
persists incremental validator state, and serializes catalog generations. A
corrupt cold catalog now stays metadata-only rather than forcing a synchronous
corpus rebuild.
