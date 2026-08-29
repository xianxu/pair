---
id: 000156
status: working
deps: []
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 8.96
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
opaque stable file ID, non-reusable generation token, size, mutation token,
timestamps needed by the
public contract, authorization state, scanner classification/facts, raw observed
offset, parser-complete offset, incremental scanner state, and the
scanner/parser schema version that produced it.

`StableFileID`, `GenerationToken`, and `MutationToken` are separate opaque
runtime values built without content reads. On Unix the stable ID is device +
inode. On Darwin the generation token is `stat.st_gen` (not the birth
timestamp); on Linux, whose `statx` exposes birth time but no inode generation,
Pair reports generation unavailable. The mutation token includes platform
`ctime` at nanosecond precision. Implementations on another platform must
supply an actual inode/file generation primitive or report it unavailable.
Device +
inode alone is not durable because inode reuse is possible. Size and
modification time remain separate fields. Stable-ID + generation continuity
plus size growth is the only append candidate; a stable-ID/generation change,
shrinkage, or mutation without size growth invalidates authority and requires
targeted revalidation. The changed mutation token is recorded after suffix
validation. If the runtime cannot obtain a non-reusable generation token,
persistent continuity is unproven: any mutation is treated as replacement
rather than risking cached authority. Path + size + mtime never proves
continuity.

Catalog reconciliation is incremental and deterministic:

- unchanged stable ID + generation + size + mutation token + timestamps +
  schema reuses the cached facts;
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

The typed binding record also carries a versioned `AuthorizationProof`: native
root ID, scanner schema/state, and—for every authorized artifact—stable file ID,
non-reusable generation token, mutation token, size, and parser-complete offset
at publication. This is the durable proof that survives a missing/corrupt
derived catalog and applies the same fail-closed continuity rules as a catalog
entry. On use, unchanged artifacts need no body read; append candidates validate
only suffixes from the proof offsets. A legacy
binding without a proof receives one background, targeted full validation of
its named artifact and then publishes the proof. It never triggers a corpus
scan. Until that validation succeeds, it remains durable historical evidence
but is unavailable for automatic resume/activity projection. An explicit resume
without a proof may synchronously perform the same one-artifact validation; a
contradiction anywhere in that transcript fails authorization. Tests include a
valid identifying record followed by a later Claude/Muse contradiction.

The catalog is a reusable inventory seam, not a cache of rendered CLI output.
Forest/query/activity/launch/watcher consumers derive from the same catalog
facts (`ARCH-DRY`, `ARCH-PURPOSE`). Reconciliation and selection remain pure;
filesystem enumeration, targeted reads, metadata, and persistence stay behind
the injected runtime (`ARCH-PURE`). The existing stateful fake grows catalog,
append, replacement, truncation, deletion, and corruption behavior so tests use
the production boundary rather than stateless call mocks (`ARCH-MOCK`).

### Trusted append-only storage contract

Filesystem metadata cannot prove that an earlier byte was not rewritten while
the file also grew. Pair therefore does not pretend `ctime` is an append proof.
Suffix reuse is enabled only for allowlisted native JSONL transcript stores
whose producer contract is append-only: Claude project transcripts, Codex
rollouts, Muse sessions, and Agy joined transcripts. Pair never writes those
files. For these stores, stable ID + generation continuity and size growth is
interpreted according to the provider's append-only contract; the suffix is
still fully validated before the mutation token/offset advances.

An in-place prefix rewrite plus growth violates that external storage contract
and is provider corruption, not a supported mutation Pair can distinguish
without rereading the categorized body. Explicit conformance/deep validation
detects and reports it; live conformance fixtures pin the append-only assumption
for each supported provider version. Same-size mutation, shrinkage, stable-ID or
generation change, parser-schema change, and all mutations of non-allowlisted
stores fail closed as replacement and require targeted revalidation. Agy SQLite
databases are explicitly outside the append-only set and always use their keyed
query seam after mutation. This scoped trust is the necessary tradeoff for the
operator's invariant that categorized content is not reread.

### Candidate discovery versus authorization

Metadata discovery produces candidates and exclusion watermarks, never scanner
authority. Scanner facts become authorized only through one of these paths:

| agent | cold candidate discovery | targeted authorization | new-launch observation |
|---|---|---|---|
| Claude | path shape yields possible root/subagent ID and raw size | proof-bearing binding validates unchanged metadata or suffix; proofless binding/explicit resume fully validates only the named UUID transcript once | read the new file from byte zero and incrementally validate every appended record until the completed round binds |
| Codex | rollout path yields a possible artifact and raw size | proof-bearing binding validates unchanged metadata or suffix; proofless binding/explicit resume fully validates the named rollout including required first `session_meta` once | read the new rollout from byte zero, require `session_meta`, then validate appended events |
| Muse | session path yields a possible native ID and raw size | proof-bearing binding validates unchanged metadata or suffix; proofless binding/explicit resume fully validates only the named session once | read the new session from byte zero and incrementally validate appended records |
| Agy | database and joined transcript paths yield an untrusted native-ID candidate with separate metadata boundaries | proof-bearing binding validates both artifacts; proofless binding/explicit resume schema- and identity-validates the named database and fully validates only its joined transcript once | only a database absent from the baseline is schema/identity queried; its same-ID transcript must also be new, is joined explicitly, and is read from byte zero for causal events |

An established typed ledger binding was published only after prior scanner
authorization, but automatic reuse additionally requires its durable proof (or
the one-time proofless migration above). It must still resolve to artifacts
matching the agent's scanner-owned target shape; missing/replaced artifacts
report absence and never fall back to another root. Explicit resume is the only
cold foreground path that may inspect a preexisting transcript body, and it
fully validates only the named artifact—not the corpus. Unbound preexisting
candidates remain excluded until explicit diagnostic validation; they cannot
become launch bindings from metadata alone.

Agy has no historical-row cursor on a cold metadata baseline and does not
invent one. A preexisting database or transcript is excluded from fresh-launch
establishment even if it changes. A new database candidate is authorized by its
SQLite header, schema, and keyed identity row; Pair then requires the
scanner-owned same-native-ID transcript path to be new after the same launch
boundary and reads that transcript from zero. Missing or preexisting joined
transcripts cannot supply the new causal round. Established/explicit Agy paths
instead use their two-artifact authorization proof or the one-time targeted
database + joined-transcript validation.

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
whose durable binding is already known. Full-*corpus* historical content
validation is reserved for the explicit diagnostic/conformance surface and
never runs as a side effect of launch, `Alt+X`, title/context polling, or
ordinary owner lookup. The permitted exceptions are one-artifact proof
migration and explicit-resume validation: explicit resume may perform it
synchronously during launch; proofless established bindings migrate in the
background and remain unavailable to automatic resume/activity until complete.
Ordinary owner lookup reports that unavailable state and never initiates the
validation itself.

### Incremental watcher

The watcher compares the current catalog/filesystem snapshot with the durable
launch baseline. It visits only new, appended, replaced, or truncated candidate
artifacts and reads only eligible bytes after their raw launch boundary or
stored parser-complete offset. For a new candidate transcript it reads from byte
zero until it has enough evidence to classify the artifact, but it cannot
publish a binding/proof at that first match. The validator snapshots stable ID +
generation + observed EOF, consumes every complete record through that EOF, and
rejects/retracts on any later identity/role/schema contradiction in the same
snapshot. If the file grows during validation, it repeats suffix validation to
the next observed EOF before publication. The proof records the final
parser-complete offset; later appends continue from there. It does not normalize
unrelated historical transcripts.

Agy database mutation is not a byte-suffix protocol. A changed proof-bearing
database reruns only the keyed header/schema/identity query; its joined JSONL
transcript alone uses parser-complete suffix validation. A database generation
change invalidates the database proof and requires targeted reauthorization.

An artifact is eligible to contribute causal evidence only when (a) its stable
file ID/path was absent from the launch baseline, or (b) the launch explicitly
targets that root through an established/proof-bearing binding or explicit
resume and continuity/suffix validation succeeds. An append to an untrusted,
unbound preexisting candidate is excluded even if its text happens to match the
Pair round. Descendants remain eligible only through the already-authorized
root relationship defined by #155.

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
- Proofless established/explicit roots validate the complete named artifact
  once, catch late Claude/Muse contradictions, publish a durable authorization
  proof, and never scan sibling transcripts.
- A record split across the launch boundary cannot establish a round, while the
  first wholly post-launch record is retained when the old file ended in a
  newline.
- Concurrent catalog writers cannot lose a newer cursor, scanner state, or
  disputed transition.
- Inode reuse with the same device/inode cannot masquerade as append; a missing
  non-reusable generation token causes mutation to fail closed as replacement.
- Catalog loss still applies generation/mutation continuity from the durable
  proof; proof round-trip tests reject same-inode reuse.
- Prefix-rewrite-plus-growth is explicitly outside the allowlisted providers'
  append-only contract and is detected by explicit conformance; non-allowlisted
  mutations never receive suffix reuse.
- A new Claude/Muse artifact whose qualifying round precedes a later
  contradiction in the same observed EOF cannot publish a binding/proof.
- Launch baseline and watcher never normalize pre-launch historical events;
  completed-round establishment still passes the #155 exact-correlation matrix.
- Per-file metadata collection uses no external `stat` process.
- Every native-session consumer is covered by a shadow-sweep contract that
  rejects reintroduced whole-agent scans on latency-sensitive paths.
- The historical picker renders the observed tag `1` as `pair/1 claude`, while
  typed ledger binding remains the sole root authority.
- Focused tests, the full Go suite with `-p 20`, Lua/shell integration tests,
  vet, and a real-data smoke all pass.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method. The
existing scanner/runtime seams and `x/sys/unix` cover the library-availability
check; no greenfield filesystem or parser library is assumed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.20 impl=0.20
item: lua-neovim design=0.40 impl=0.40
item: real-api-discovery design=0.00 impl=0.12
item: real-api-discovery design=0.00 impl=0.12
item: real-api-discovery design=0.00 impl=0.12
item: real-api-discovery design=0.00 impl=0.12
item: atlas-docs design=0.10 impl=0.08
item: milestone-review design=0.08 impl=0.12
design-buffer: 0.15
total: 8.96
```

The five greenfield modules are catalog/reconciliation, transactional storage,
incremental framing/state, targeted proof/query orchestration, and watcher
delta orchestration. The seven smaller modules are platform metadata, ledger
v2, four existing scanner adaptations, and performance/conformance. The four
real-API discovery rows cover the installed Claude, Codex, Muse, and Agy
provider contracts.

## Plan

- [ ] Define the versioned artifact catalog and pure incremental reconciliation
  rules with stateful fake coverage; filesystem fingerprints and fake mutation
  operations are complete.
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
- `sdlc change-code` plan-quality cleared after two rounds. The gate corrected
  generation continuity to Darwin `stat.st_gen` with fail-closed unavailable
  fallback elsewhere, required named risky-function strategies, and added the
  live fake-versus-provider conformance cadence (`ARCH-PURE`, `ARCH-MOCK`). The
  v3.1 estimate gate accepted 8.96h with an advisory that the dense integration
  matrix may run higher.
- Added the selected-scope catalog path and a strict, flock-serialized catalog
  store. Concurrent writers recompute from the locked generation; atomic
  replacement and file/directory sync expose explicit not-authoritative,
  indeterminate, and committed outcomes. Fault-injected recovery tests cover
  write, file sync, rename, directory sync, and unlock boundaries (`ARCH-PURE`).
- Added ledger v2 while retaining strict v1 reads. V2 launch rows persist sorted
  content-free artifact boundaries; v2 binding rows persist root-matched scanner
  state and complete artifact continuity proofs. Proof bindings publish only
  under the current-launch lock and retain the ledger's byte-level authority
  outcomes.

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

### 2026-08-29 — durable proof and exact candidate eligibility

Second spec review found that a bounded identifying record could not replace
Claude/Muse whole-artifact contradiction validation, conflated stable identity
with the append-changing mutation token, left Agy's database/transcript join
without a cold boundary, and did not say whether an append to an untrusted old
file could bind. The spec now publishes scanner state and offsets as a durable
binding proof; migrates proofless bindings by validating one named artifact,
never the corpus; separates device/inode identity from mutation; defines Agy's
new database plus new joined transcript path; and restricts watcher evidence to
new artifacts or explicitly targeted, proof-validated roots.

### 2026-08-29 — non-reusable generation and proof-through-EOF

Third spec review found that device/inode can be reused, that first-match
binding could miss a later contradiction already present in the same file, and
that the diagnostic-only wording accidentally excluded targeted proof
migration. The spec now requires a non-reusable birth/generation token (or
fail-closed replacement), validates every complete record through the observed
EOF before publishing a proof, distinguishes Agy keyed database revalidation
from transcript suffix parsing, and permits only one-artifact proof migration or
explicit-resume validation outside explicit full-corpus diagnostics.

### 2026-08-29 — explicit append-only provider contract

Fourth spec review identified that metadata cannot prove a growing file's
prefix stayed unchanged and that the durable proof omitted its generation and
mutation tokens. The proof now persists the complete continuity tuple. The spec
also makes the unavoidable trust boundary explicit: suffix reuse exists only
for versioned native JSONL stores whose producers append; SQLite and every other
mutation revalidate through targeted seams. Prefix rewrite plus growth is
provider corruption covered by explicit conformance, since detecting it on
every hot-path append would require rereading the already-categorized body and
violate the issue's purpose.

### 2026-08-29 — generation primitive and live conformance

The plan-quality gate found that file birth time is not a non-reusable
generation counter. The contract now uses Darwin's actual `stat.st_gen` and
marks Linux/other platforms unavailable unless they expose an equivalent
primitive; birth time remains a timestamp only. Provider-contract conformance
will compare the stateful fake with installed Claude, Codex, Muse, Agy
transcript, and Agy SQLite behavior during #156, before every scanner/provider
contract version change, and on the monthly operator conformance run.
