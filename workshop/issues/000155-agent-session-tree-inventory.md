---
id: 000155
status: working
deps: []
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
started: 2026-08-28T10:54:39-07:00
---

# deterministic agent session-tree inventory

## Problem

Pair's session probing is optimized to discover one native root session for one
new launch. It walks the current agent process tree and returns the first
authorized open transcript; fallbacks select a newly created file or the newest
birth-time candidate. After one ID is written to the tag config and ledger, the
watcher exits. Codex and Muse subagent transcripts are deliberately rejected or
ignored.

That is adequate for a happy-path resume token, but it cannot reconstruct the
world after an unclean shutdown. It does not model the agent's complete session
directory, preserve root-to-subagent relationships, explain competing
candidates, or reliably answer which native session tree belongs to which Pair
tag and in what order. Different Pair consumers consequently grow partial
point-lookups and recency heuristics around the same incomplete evidence.

## Spec

**Model native sessions as forests, not a flat list of resumable roots.** For
every supported Pair agent, one deterministic read-only inventory enumerates
all recognized native session nodes under that agent's storage roots. A node
records the native session ID, agent, transcript/artifact paths, root versus
subagent role, native parent ID when available, and stable native timestamps.
Every recognized subagent remains in the model even when only its root is
directly resumable. Missing parents, malformed metadata, unreadable paths, and
schema near-misses are explicit diagnostics rather than silently dropped rows.

Agent-specific scanners translate Claude, Codex, Agy, and Muse storage into one
shared forest model. Tree construction and ordering are pure over scanner
facts. Sibling and root order is deterministic: native chronology first, then
native ID and canonical path as total tie-breakers. Filesystem traversal order,
glob order, PID enumeration order, and `lsof` output order never affect the
result (ARCH-PURE, ARCH-PURPOSE).

**Tag correlation is a separate evidence-ranked pass.** It binds a Pair tag to
a native root only from authoritative evidence, in this order:

1. the exact Pair config or append-only ledger session ID for `{scope, tag,
   agent}`;
2. an identity-authorized live Pair PID tree holding that native root artifact
   open;
3. a unique ordered correspondence between user text observed by Pair and user
   turns parsed from the native transcript tree;
4. native parent relationships, which attach descendants to an already bound
   root.

Content correlation is deterministic and provenance-preserving, not semantic:
agent-specific transcript parsers emit ordered user turns, while Pair-owned
logs provide the exact ordered text the operator submitted under a tag.
Whitespace/framing normalization is explicit and shared; no LLM, embedding, or
fuzzy paraphrase decides identity. Generic or repeated strings are weak
evidence. A unique sufficiently discriminative turn, or an ordered sequence of
exact turns, may become high-confidence evidence under mechanically tested
rules. The output explains which turn positions/fingerprints matched without
creating another durable copy of the transcript content.

Correlation runs globally over all tags and roots rather than greedily one tag
at a time. It locks unique high-confidence assignments first, removes both the
assigned tag and root from later candidate sets, then evaluates weaker evidence
against the remainder. Conflicting high-confidence claims remain ambiguous;
scan order can never decide the winner. Already assigned descendants follow
their native root and cannot be reassigned independently.

Timestamps order otherwise plausible candidates for inspection but never
authorize a binding. Conflicting authoritative evidence and equally supported
candidates remain explicitly ambiguous and unbound. The result includes every
candidate and the evidence for and against its correlation so operators and
agents can explain the decision instead of inheriting a silent newest-file
guess.

Expose the inventory through Pair's single public binary with structured JSON
for agents and a stable human rendering for diagnosis. The structured contract
contains forests, correlations, ambiguities, and scan diagnostics; it does not
read Couch persistence. Existing session-watch, transcript lookup, recovery,
and context consumers derive their selection from this shared model rather
than maintaining independent first/newest algorithms (ARCH-DRY).

Tests use portable directory fixtures for every supported agent, including
multiple roots, nested subagents, overlapping timestamps, malformed nodes,
unreadable entries, stale Pair bindings, concurrent live evidence, and
conflicts. A live conformance probe captures each installed agent's current
directory shape without requiring an LLM response and compares it with the
scanner contract (ARCH-MOCK).

### Operational contract

The correlation subject is a **tag incarnation**, not a timeless tag. The
normative incarnation identity, legacy fallback, root-sharing, and ordering
rules are defined once in **Send transaction and identity amendment** below.
For the current-tag projection, legacy config (which has no ordinal) precedes
ledger-backed incarnations; ledger-backed incarnations order by
`(source_ordinal, incarnation_id)`, and the last valid one is current. Invalid
rows remain diagnostics and never participate in selection.

The ledger is Pair's authority. Config files are validated compatibility
caches, not a second source of truth. A config participates only when its exact
scope/tag/agent path is resolved by `artifactpath`, its body agent matches the
path agent, and its native root passes the same scanner authorization as every
other candidate. When a valid ledger segment exists, a matching config
corroborates it and a disagreement emits `binding_stale` and is ignored. A
pre-ledger config may create one legacy incarnation only when it is the sole
valid config for the tuple; otherwise it remains ambiguous. Malformed paths,
scope escapes, body/path identity mismatches, and unrecognized native roots
fail closed.

Correlation consumes this exhaustive evidence lattice:

| Rank | Evidence | Validation | Assignment rule |
|---|---|---|---|
| 1 | ledger-backed native ID | valid tuple, segment, scanner-authorized root | canonical candidate; two incarnations claiming one root remain `binding_conflict` |
| 1 | sole pre-ledger config | exact validated path/body and scanner-authorized root | legacy candidate; competing configs remain ambiguous |
| 2 | live Pair process | exact scoped PID sidecar; process identity unchanged before and after the descendant/open-file snapshot; scanner-authorized root | candidate only for that tuple; disagreement with rank 1 is diagnosed and rank 1 wins |
| 3 | exact sent-turn sequence | committed send-journal rows for one valid incarnation and the fingerprint rules below | considered only for still-unbound incarnations and roots |
| derived | native parent edge | child and parent scanner facts agree | descendants inherit the assigned root; never compete as independent roots |

Each rank is solved globally in deterministic fixed-point rounds using the
Pair-owner exclusivity algorithm defined in **Send transaction and identity
amendment**. Equal-rank conflicts stay ambiguous; lower ranks cannot break
them. Lower-rank evidence that contradicts a locked higher-rank assignment is
retained as negative evidence, never as reassignment authority.

Exact-turn evidence uses one shared `SentText` normalization owned by Pair's
send/log boundary: apply the production comment-framing removal, normalize
CRLF to LF, remove trailing horizontal whitespace per line, and trim outer
blank space; do not case-fold, collapse internal whitespace, or paraphrase.
Native parsers emit only allowlisted operator-authored user records, with
agent/system/generated/sidechain records excluded by an explicit source kind;
unknown source kinds emit `turn_unusable` and do not correlate. A match is a
contiguous sequence in filtered native operator turns against one incarnation's
committed submitted journal sequence. Native turns outside the matched window
are allowed, but gaps inside it are not; cross-incarnation window ownership is
governed by the final identity amendment.

One turn authorizes only when its normalized UTF-8 is at least 32 bytes, has at
least five Unicode word tokens, and its SHA-256 fingerprint occurs exactly once
among all remaining Pair segments and native roots. Otherwise, two consecutive
turns authorize when each is at least eight bytes, at least one has three word
tokens, and the ordered fingerprint pair is globally unique. Empty, shorter,
repeated, or non-unique text is generic evidence and is explanation-only. The
result records source and destination positions plus fingerprints, never a
second durable copy of content. Tests pin comment framing, generated prompts,
resumed roots, fresh-root rotation, repeated prompts, partial prefixes, and
single-versus-sequence thresholds.

Agent scanners recognize these versioned contracts; anything else is retained
as a coded near-miss rather than guessed (ARCH-PURPOSE):

| Agent | Storage/artifact join | ID, role, and parent authority | Chronology authority |
|---|---|---|---|
| Claude | `.claude/projects/<encoded-cwd>/<root-id>.jsonl` plus `<root-id>/subagents/agent-<child-id>.jsonl`; v1 accepts no sidecars | root/child filenames and containing root directory; JSONL may only corroborate | first accepted top-level RFC3339 timestamp, then birth time, then mtime |
| Codex | `.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` | first bounded `session_meta.payload.id`; `parent_thread_id == null` with source `cli|exec` is root, non-null parent is child | `session_meta` timestamp, then birth time, then mtime |
| Agy | `.gemini/antigravity-cli/conversations/<id>.db` joined by ID to `brain/<id>/.system_generated/logs/transcript.jsonl` | v1 DB header + exact `trajectory_meta` root rule below; child edges remain unasserted until fixture-proven | birth time, then mtime |
| Muse | `.local/share/muse/sessions/YYYY/MM/DD/<root-id>/session.jsonl` plus `<root-id>/subagent/<child-id>/session.jsonl` | directory UUID and exact path ancestry; accepted records may only corroborate | birth time, then mtime |

The implementation must obtain and check in a sanitized Agy subtrajectory
schema/fact fixture before accepting any Agy parent edge; absent that evidence,
the scanner emits the root and an explicit near-miss/orphan diagnostic instead
of inventing semantics. Duplicate artifacts with the same `{agent, native_id}`
coalesce only when role and parent agree, with sorted unique artifact paths.
Metadata/path disagreement or duplicate role/parent disagreement emits
`parent_conflict` or `duplicate_conflict`, retains the node unbound, and creates
no disputed edge.

Every emitted collection has a total order. Paths are cleaned, slash-separated
paths relative to a named native storage root; paths that escape that root or
traverse a symlink are rejected. Times carry their source and are rendered UTC
RFC3339Nano; absent/invalid time is null and sorts after present time. Forest
roots and siblings sort by `(time missing, time, native_id, storage_root,
relative_path)`. Artifacts sort by `(storage_root, relative_path)`;
incarnations, correlations, candidate edges, ambiguities, evidence, and
diagnostics each have documented tuple comparators ending in stable IDs and
paths. JSON uses schema version `1`, structs and sorted arrays rather than maps,
one trailing newline, and no HTML escaping. Fixture/property tests shuffle
filesystem, PID, `lsof`, ledger, config, turn, and injected diagnostic order and
require byte-identical JSON and human output (ARCH-PURE).

The public command is:

```text
pair session-inventory [--agent claude|codex|agy|muse] [--scope current|all] [--json] [--conformance]
```

The default inventories every agent and correlates tags in the selected current
scope. `--scope all` includes every Pair repo scope. Human output is a stable
snapshot grouped by agent forest then scope/tag incarnation; JSON stdout is a
versioned object with required `schema_version`, `forests`, `correlations`,
`ambiguities`, and `diagnostics` arrays. Optional IDs, parents, times, and paths
are explicit `null`, never omitted. Normal operation writes only the selected
rendering to stdout; fatal/usage text goes to stderr. `--conformance` uses the
same production scanners but emits only redacted versions, schema shapes,
counts, and diagnostics—never content, cwd, absolute home paths, or raw IDs.

Diagnostic codes are `storage_unreadable`, `node_malformed`,
`schema_near_miss`, `parent_missing`, `parent_conflict`, `duplicate_conflict`,
`binding_stale`, `binding_conflict`, `process_changed`, `turn_unusable`, and
`conformance_no_sample`, each with severity plus nullable agent/native-ID/path
coordinates. Exit `0` means a result was emitted, including partial results
with coded diagnostics; exit `1` is invalid usage or an unsupported requested
agent; exit `2` means no requested scanner could produce a result or rendering
failed. A missing installed-agent sample is an explicit conformance skip
diagnostic and exit `0`; scheduled conformance fails only on recognized schema
drift, unreadable representative data, or privacy leakage.

One injected IO boundary owns filesystem enumeration/reads/stats, SQLite
schema/facts, Pair artifacts, process ancestry, PID identity, and open files.
Its portable stateful fake models directory and DB state, ordered/unordered
returns, unreadable entries, PID reuse, process mutation between calls, and
concurrent open-file evidence. Production and integration tests use the same
scanner/correlation entry point; pure forest, normalization, ordering, and
matching tests require no IO fake. The opt-in no-LLM live check is
`PAIR_LIVE_NATIVE_SESSIONS=1 go test ./cmd/internal/sessioninventory -run
TestLiveNativeSessionShapeConformance -v`; it runs manually before release and
in scheduled workstation conformance, with the skip/fail rules above
(ARCH-PURE, ARCH-MOCK).

The migration shadow sweep includes `sessionwatch`; `transcript` point lookup;
`codexsid`; slug transcript parsing and live fallback; context/title lookup;
review targeting; launcher existence, live capture, restart, config picker, and
history recovery; opener/changelog keying; and Neovim review/config/session-age
lookups. Exact inherited `PAIR_SESSION_ID` remains direct launch authority, but
all native path/root validation and every filesystem/process candidate
selection derives from the shared inventory. No consumer outside the inventory
package may glob, walk, `find`, or `lsof` native session storage, choose a
first/newest candidate, or parse a native transcript independently. A
repository shadow-sweep test enforces that enumeration (ARCH-DRY,
ARCH-PURPOSE).

### Send-boundary and matching amendment

Rank-3 evidence does not infer delivery from the human markdown log. Each
launch/restart mints the raw Pair `incarnation_key` defined by the final identity
amendment, records it on every ledger row for that incarnation, passes it
through the watcher, and exports it to the composer. The zero-based physical
ledger line number is retained as
`source_ordinal` even when a malformed line consumes that ordinal; parsers sort
by it before coalescing adjacent rows. Shuffled tests shuffle parsed facts with
their ordinals intact, never erase source order.

The send boundary uses the versioned `sent-<tag>.jsonl` write-ahead journal
defined in **Send transaction and identity amendment** for both normal submit
and `no_submit` composer injection. It contains no prompt text. Only valid
`prepared -> committed` events whose delivery is `submitted` are Pair-side
rank-3 inputs; `composer_only`, aborted, incomplete, and malformed events never
authorize. A record with the wrong resolved scope/tag,
body/path agent mismatch, duplicate sequence/event ID, non-monotonic sequence,
unknown delivery, invalid fingerprint, or unknown incarnation emits
`pair_record_malformed` and is excluded. Pre-v1 markdown entries and v1 rows
that cannot join uniquely to a ledger incarnation are `turn_unusable`; local
timestamp proximity never repairs them. This makes submission and incarnation
segmentation exact rather than time-inferred (ARCH-PURE, ARCH-PURPOSE).

All same-rank candidate edges are intentionally equal; evidence counts do not
vote. Multiple qualifying records for one `{incarnation, root}` coalesce into
one edge with sorted evidence. The final Pair-owner algorithm determines which
equal-rank edges lock; contradictory qualifying fingerprints diagnose
`binding_conflict` instead of becoming an undocumented score.

### Versioned native record allowlists

All filesystem candidates must be regular, non-symlink files beneath their
named root. A first metadata record is newline-terminated and at most 1 MiB;
subsequent JSONL records used for turns are at most 8 MiB each. Oversize,
truncated, unknown, or wrong-typed records emit `schema_near_miss` or
`node_malformed` and contribute no asserted fact. Versioned sanitized fixtures
under `cmd/internal/sessioninventory/testdata/native/<agent>/v1/` are the
executable allowlists; widening one requires a fixture and conformance update
in the same change.

- **Claude v1:** a root is exactly
  `.claude/projects/<encoded-cwd>/<uuid>.jsonl`; a child is exactly
  `.claude/projects/<encoded-cwd>/<root-uuid>/subagents/agent-<ascii-id>.jsonl`.
  Nested or sidecar shapes are near-misses until fixture-proven. The path owns
  root ID, child ID, role, and direct parent; JSONL may not contradict it. The
  first accepted top-level RFC3339 `timestamp` is chronology, then birth time,
  then mtime. An operator-turn record requires `type:"user"`, absent/false
  `isSidechain`, `message.role:"user"`, and `message.content` either a string or
  an array whose only consumed blocks are `{type:"text",text:string}`;
  tool-result and unknown blocks are excluded.
- **Codex v1:** the filename is
  `.codex/sessions/YYYY/MM/DD/rollout-*-<uuid>.jsonl`; the first record must be
  `type:"session_meta"` with matching `payload.id`. A root has null
  `parent_thread_id` and string source `cli` or `exec`. A child has a UUID
  `parent_thread_id`; source is an object whose sole key is `subagent`, whose
  value is either `{}` or has the sole key `thread_spawn` with an object
  containing only integer `depth >= 1`; all other key/type shapes are
  near-misses. The record's top-level RFC3339 `timestamp` is chronology,
  then birth time, then mtime. An operator turn is `type:"response_item"`,
  `payload.type:"message"`, `payload.role:"user"`, with only
  `{type:"input_text",text:string}` content consumed.
- **Agy v1:** a root candidate is exactly
  `.gemini/antigravity-cli/conversations/<uuid>.db`, has a SQLite header, and
  exposes `trajectory_meta(cascade_id, trajectory_type, source)` with exactly
  one non-empty `cascade_id` equal to the filename UUID. The transcript join is
  exactly `brain/<id>/.system_generated/logs/transcript.jsonl`; its absence is
  `parent_missing`-severity warning but does not erase the DB root. Birth time,
  then mtime, supplies chronology. An operator turn is JSONL
  `type:"USER_INPUT"` with string `content`; when a single well-formed
  `<USER_REQUEST>...</USER_REQUEST>` wrapper exists, only its interior is
  consumed, while malformed/multiple wrappers are near-misses. `steps` and
  `parent_references` assert no child edge until a sanitized populated fixture
  pins their exact columns and values.
- **Muse v1:** a root is exactly
  `.local/share/muse/sessions/YYYY/MM/DD/<uuid>/session.jsonl`; a child is
  exactly `<root>/subagent/<uuid>/session.jsonl`. Deeper nesting is a near-miss.
  Directory UUIDs own IDs and direct parent; accepted metadata may not
  contradict them. Birth time, then mtime, supplies chronology. An operator
  turn is exactly `payload_type:"runtime.session"`, `payload.kind:"run"`,
  `payload.event.kind:"started"`, with string `payload.event.prompt`; when
  present, `payload.run_id` must equal the directory UUID.

### Schema-v1 output and result matrix

Stable IDs are lowercase `kind-` plus the first 24 hex characters of SHA-256
over the kind name and length-prefixed canonical tuple fields: nodes use
`(agent,native_id)`, incarnations use `(scope_key,tag,agent,incarnation_id)`,
evidence uses `(kind,source_ref,incarnation_id,node_id)`, and ambiguities use
`(kind,rank,sorted incarnation IDs,sorted node IDs)`. The complete JSON shape
is below; every field is required, and `*` means JSON null when unknown:

```text
InventoryV1 {
  schema_version:int,
  forests:[ForestV1], correlations:[CorrelationV1],
  ambiguities:[AmbiguityV1], diagnostics:[DiagnosticV1]
}
ForestV1 { agent:string, roots:[NodeV1], orphans:[NodeV1] }
NodeV1 {
  node_id:string, native_id:string, role:"root"|"subagent"|"orphan",
  parent_native_id:*string, resumable:bool,
  created_at:*RFC3339Nano, time_source:*"metadata"|"birth"|"mtime",
  artifacts:[ArtifactV1], children:[NodeV1]
}
ArtifactV1 { storage_root:string, relative_path:string, kind:string }
CorrelationV1 {
  incarnation_id:string, scope_key:string, tag:string, agent:string,
  source_ordinal:*uint64, root_node_id:*string,
  status:"bound"|"ambiguous"|"unbound", rank:*int,
  candidates:[CandidateV1], evidence:[EvidenceV1]
}
CandidateV1 {
  root_node_id:string, rank:int,
  outcome:"locked"|"conflict"|"rejected", evidence_ids:[string]
}
EvidenceV1 {
  evidence_id:string, kind:"ledger"|"config"|"live"|"turn"|"parent",
  rank:int, source_ref:string, positive:bool,
  source_positions:[uint64], destination_positions:[uint64],
  fingerprints:[string]
}
AmbiguityV1 {
  ambiguity_id:string, kind:"incarnation"|"root"|"evidence",
  rank:int, incarnation_ids:[string], root_node_ids:[string],
  evidence_ids:[string]
}
DiagnosticV1 {
  diagnostic_id:string,
  severity:"info"|"warning"|"error", code:string,
  agent:*string, native_id:*string, storage_root:*string,
  relative_path:*string, source_ref:*string
}
```

Comparators are exhaustive: forests by agent; node siblings/orphans by the node
tuple already specified; artifacts by `(storage_root,relative_path,kind)`;
correlations by `(scope_key,tag,agent,source_ordinal null first,incarnation_id)`;
candidates by `(rank,root_node_id,outcome,evidence_ids joined)`; evidence by
`(rank,kind,source_ref,evidence_id)`; ambiguities by
`(kind,rank,incarnation_ids joined,root_node_ids joined,ambiguity_id)`; and
diagnostics by `(severity order error/warning/info,code,agent null last,
native_id null last,storage_root null last,relative_path null last,source_ref
null last,diagnostic_id)`. Every nested ID/fingerprint/position array sorts
ascending before its owner is compared or rendered.

Diagnostic severity is fixed: `conformance_no_sample` is info;
`schema_near_miss`, `parent_missing`, `binding_stale`, `process_changed`, and
`turn_unusable` are warning; `storage_unreadable`, `node_malformed`,
`parent_conflict`, `duplicate_conflict`, `binding_conflict`,
`artifact_path_invalid`, `pair_record_malformed`, `scope_rejected`, and
`conformance_privacy_violation` are error. Invalid CLI usage and unsupported
requested agents are stderr-only usage failures, not inventory diagnostics.

| Mode/facts | Stdout | Stderr | Exit |
|---|---|---|---|
| normal, at least one requested scanner completed, including partial diagnostics | complete selected rendering | empty | 0 |
| normal, storage roots legitimately absent for every requested agent | complete empty rendering with info diagnostics | empty | 0 |
| normal, every present requested root failed before producing scanner facts | empty | stable fatal summary | 2 |
| conformance, representative sample absent | redacted rendering with `conformance_no_sample` | empty | 0 |
| conformance, recognized schema drift or unreadable representative sample | redacted rendering with diagnostics | stable failure summary | 2 |
| conformance, any privacy invariant fails | no rendering | stable privacy failure | 2 |
| either, JSON/human serialization fails | no/aborted rendering | stable render failure | 2 |
| invalid flags or unsupported explicit agent | no rendering | usage | 1 |

“Completed” means the scanner inspected an absent root or traversed a present
root to a finite fact/diagnostic set; opening a present root and failing before
enumeration is not completion. Fatal summaries contain codes and redacted root
labels, never raw OS errors or private paths.

### Send transaction and identity amendment

The `incarnation_key` minted before each agent launch/restart is the sole
post-v1 Pair-incarnation authority. Every ledger row for that launch carries
the same key, and the incarnation's `source_ordinal` is its minimum valid
physical ledger ordinal. Pre-v1 rows without a key form legacy incarnations
only from physically contiguous equal `{scope,tag,agent,session_id}` rows; they
cannot use rank-3 evidence. This supersedes the earlier time-boundary and
contiguous-segment definitions. Stable public IDs are distinct from raw keys:
post-v1 uses `(scope_key,tag,agent,"v1",incarnation_key)`; legacy ledger uses
`(scope_key,tag,agent,"legacy-ledger",minimum source_ordinal,session_id)`; and
sole pre-ledger config uses `(scope_key,tag,agent,"legacy-config",storage_root,
relative_path,session_id)`. Each tuple is hashed by the schema-v1 stable-ID rule.

One native root may span several launches that resume it. Root exclusivity is
therefore by **Pair owner** `{scope,tag,agent}`, not by incarnation. Within a
rank, an incarnation/root edge is lockable when it is that incarnation's only
edge and every remaining edge incident to the root belongs to the same Pair
owner. A simultaneous round locks all such edges, reserves each root to its
owner, removes the locked incarnations plus every competing-owner edge to the
reserved roots, and repeats. Multiple incarnations of the reserved owner may
then bind the same root; another owner produces `binding_conflict`. For turn
evidence, committed send sequences from same-owner incarnations must map to
unique, non-overlapping native windows whose destination positions increase by
`source_ordinal`. An ambiguous placement or a later incarnation matching only
an earlier window is `turn_unusable`, so a resumed root and a fresh-root
rotation are mechanically distinguishable.

`sent-<tag>.jsonl` is a write-ahead state journal, not a claim that files and a
PTY can commit atomically. One send owner holds an advisory lock for the exact
scope/tag across the whole protocol. Under that lock it allocates
`sequence = max(all valid durable prepared sequences for the incarnation)+1`.
Sequence consumption begins only after `prepared` append and fsync succeeds;
an append that never becomes durable allocated no observable number. Aborted
and incomplete durable preparations consume their number. Journal rows are:

```text
prepared  {v,event_id,log_entry_id,scope_key,tag,agent,incarnation_key,
           sequence,delivery,sent_at,fingerprint,byte_count,word_count}
committed {v,event_id,state:"committed"}
aborted   {v,event_id,state:"aborted",failure_code}
```

The serialized protocol is:

1. append + fsync `prepared`;
2. append + fsync the markdown log entry carrying `log_entry_id`;
3. perform the ordered zellij focus/body/submit-or-newline actions and require
   success from every action that constitutes delivery;
4. append + fsync `committed`, then release the lock.

A failure before step 3 appends `aborted` and sends nothing. A delivery-action
failure appends `aborted`, best-effort refocuses the draft, and reports that the
composer may contain a partial body; it never retries automatically. A commit
append/fsync failure after successful delivery leaves a dangling `prepared`
row, reports “delivery succeeded; identity evidence not committed,” and also
never retries. On recovery, dangling prepared rows emit `send_incomplete` and
never authorize correlation; aborted rows emit `send_aborted` and never
authorize. Thus committed evidence implies all delivery syscalls succeeded,
while a crash may conservatively lose evidence for delivered text. The system
never promotes a dangling row from transcript content, which would be circular
authority. `composer_only` uses the same transaction but is ineligible for rank
3 even when committed.

Recovery groups rows by `event_id` and applies this complete state table after
validating row shapes and physical order:

| Durable rows for one event | Result |
|---|---|
| exactly one `prepared` | incomplete; exclude and emit `send_incomplete` |
| exactly one `prepared`, then exactly one `committed` | valid terminal event |
| exactly one `prepared`, then exactly one `aborted` | aborted; exclude and emit `send_aborted` |
| terminal before/missing `prepared` | exclude and emit `pair_record_malformed` |
| duplicate `prepared` or duplicate terminal | exclude and emit `pair_record_malformed` |
| both terminal kinds in either order, including commit-after-abort or abort-after-commit | exclude and emit `pair_record_malformed` |
| malformed/partial JSON, unknown state, or fields on a terminal row beyond its allowed shape | exclude the affected event when identifiable, otherwise exclude the row; emit `pair_record_malformed` |

Every physical journal line retains an ordinal, including malformed lines.
Recovery reads the complete journal before exposing any committed evidence, so
a later conflicting row invalidates the whole event rather than leaving an
earlier commit temporarily authoritative.

Fault-injection tests cover lock contention/stale-owner recovery; concurrent
sequence allocation; every append, fsync, log, focus, body-write,
submit/newline, refocus, commit, and unlock failure; and crashes after each
step, for both `submitted` and `composer_only`. They assert no committed row
before successful delivery, no automatic duplicate delivery, stable recovery
diagnostics, monotonic sequence consumption after durable preparation, and
every invalid recovery transition in the table (ARCH-PURE, ARCH-MOCK).

Schema-v1 uses `CorrelationV1.incarnation_id` only for the derived stable ID;
the raw key is never rendered. `CorrelationV1.source_ordinal` is nullable for
pre-ledger config incarnations. Diagnostic IDs are
`diagnostic-` plus the first 24 SHA-256 hex characters of the length-prefixed
tuple `(severity,code,agent,native_id,storage_root,relative_path,source_ref)`;
identical diagnostics coalesce before rendering.

This is the single exhaustive diagnostic registry, superseding earlier lists:

- info: `storage_absent`, `conformance_no_sample`;
- warning: `schema_near_miss`, `parent_missing`, `binding_stale`,
  `process_changed`, `turn_unusable`, `send_incomplete`, `send_aborted`;
- error: `storage_unreadable`, `node_malformed`, `parent_conflict`,
  `duplicate_conflict`, `binding_conflict`, `artifact_path_invalid`,
  `pair_record_malformed`, `scope_rejected`,
  `conformance_privacy_violation`.

All rendering is completed into a buffer before any stdout write. Serialization
failure therefore produces zero stdout bytes. `storage_absent` represents a
legitimately absent requested root in normal mode. Byte-golden tests cover
every row of the normal/conformance result matrix, including absent roots and
zero-byte stdout on usage, privacy, and rendering failures.

## Done when

- One command inventories complete root/subagent session forests for every
  supported Pair agent in stable human and JSON forms.
- Native parent-child edges are preserved; subagents are never discarded merely
  because they cannot be resumed as roots.
- Pair tags bind only through explicit ranked evidence, with ambiguous or
  conflicting candidates left unbound and fully explained.
- Exact ordered user-turn matches can identify otherwise unbound roots; generic
  or duplicated strings do not silently authorize a binding, and roots already
  assigned with high confidence are excluded from later matches.
- Identical filesystem facts produce byte-stable tree order regardless of walk,
  glob, process, or `lsof` ordering.
- The current session watcher consumes the shared inventory/correlation model
  and no longer selects a root through an unreported first/newest heuristic.
- Fixtures cover all supported agents and a no-LLM live conformance probe
  detects native directory-shape drift.
- Ledger-backed tag incarnations, validated config caches, live process
  evidence, exact-turn evidence, and inherited child edges follow the pinned
  global fixed-point rules; every ambiguity and contradiction is retained.
- `pair session-inventory` implements schema-v1 JSON, stable human output,
  coded partial diagnostics, redacted conformance, and the specified exit
  statuses.
- The full migration shadow sweep has no remaining native-session glob, walk,
  `find`, `lsof`, first/newest selection, or independent native parser outside
  `cmd/internal/sessioninventory`.
- A portable stateful fake exercises filesystem, SQLite, process, PID-reuse,
  open-file, and error transitions through the production inventory entry
  point; shuffled inputs produce byte-identical complete output.

## Plan

- [ ] Define the pure session-node, forest, evidence, correlation, ambiguity,
      diagnostic, and deterministic-ordering model.
- [ ] Implement complete agent-specific filesystem scanners for Claude, Codex,
      Agy, and Muse, including native parent/subagent metadata.
- [ ] Correlate Pair configs, ledgers, and identity-authorized live process
      evidence plus exact ordered user-turn correspondence to native roots,
      locking high-confidence assignments globally without chronology-based
      assignment.
- [ ] Expose structured and human inventory output through the Pair binary.
- [ ] Replace session-watch and transcript point-selection heuristics with the
      shared model and add portable fixtures plus live conformance.
- [ ] Pin sanitized versioned fixtures and scanner contracts for Claude, Codex,
      Agy, and Muse; do not accept an Agy parent edge before fixture proof.
- [ ] Extract the shared sent-text/native-turn parser, implement deterministic
      tag-incarnation segmentation and global fixed-point correlation, and test
      every evidence/conflict class.
- [ ] Define schema-v1 JSON, stable human rendering, coded diagnostics, the
      stateful IO fake, and redacted live conformance behind one entry point.
- [ ] Migrate and enforce the complete shadow-consumer enumeration across Go,
      shell, and Neovim rather than stopping after session-watch.
- [ ] Add the launcher-minted incarnation key and source-ordinal-preserving
      ledger parser, including post-v1 and fail-closed legacy behavior.
- [ ] Implement the serialized send journal protocol and recovery diagnostics;
      fault-inject concurrent allocation plus every write, fsync, delivery,
      commit, crash, `composer_only`, and invalid journal-transition boundary.
- [ ] Pin byte-golden schema-v1 output for every result-matrix row, including
      malformed/duplicate/non-monotonic records, absent storage, and buffered
      serialization failure.

## Log

### 2026-08-28

Split from #152 design. The operator identified that durable repository state
plus the native transcript tree is sufficient for recovery; the missing
foundation is deterministic, explainable reconstruction of every supported
agent's full root/subagent forest and its Pair-tag bindings.

## Revisions

### 2026-08-28 — correlate exact user turns before weaker candidates

**Reason:** Pair observes what the operator submitted, and native transcripts
record user turns. Their unique ordered text correspondence is deterministic
evidence that can recover bindings missed by launch-time PID/open-file probes.

**Delta:** add exact user-turn correspondence to the evidence model. Resolve
assignments globally: lock unique high-confidence tag/root pairs first, exclude
them from later candidate pools, inherit subagents through native parent edges,
and leave conflicts ambiguous. Chronology and generic repeated strings remain
ordering/debug evidence only (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-28 — make the inventory contract executable

**Reason:** fresh-context spec review found that evidence confidence, native
format coverage, exact-turn segmentation, total ordering, CLI behavior,
consumer migration, and the external fake were described directionally but not
mechanically enough to implement or verify.

**Delta:** define tag incarnations and ledger authority; pin the ranked global
fixed-point matcher and exact fingerprint thresholds; enumerate every supported
native shape and shadow consumer; specify total ordering, schema-v1 output,
diagnostic/exit behavior, the stateful IO fake, and live conformance. Agy parent
edges now fail closed until a sanitized native fixture proves their schema
semantics (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-28 — make sent evidence and schema-v1 exact

**Reason:** the second fresh-context review found that the markdown log cannot
prove submission or precise incarnation ownership, equal-rank evidence still
lacked conflict semantics, scanner record allowlists contained placeholders,
and schema-v1 could not yet support byte-golden tests.

**Delta:** add the content-free `sent-<tag>.jsonl` send-boundary record and
launcher-minted incarnation identity; retain physical ledger ordinals; make
same-rank edges equal and degree-one/degree-one fixed-point locking explicit;
pin bounded v1 native records for all four agents; and publish stable-ID,
nested-schema, comparator, diagnostic-severity, and exit/result contracts
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-28 — make the send boundary recoverable

**Reason:** the third fresh-context review caught an impossible atomicity claim,
conflicting incarnation definitions, native-allowlist contradictions, and
schema fields that could not represent legacy or failure results.

**Delta:** define a serialized write-ahead send journal with conservative crash
recovery; make launcher-minted keys the post-v1 incarnation authority; allow
same-owner incarnations to resume one root while excluding competing tags;
consolidate native v1 allowlists; distinguish raw/stable IDs and nullable
ordinals; unify the diagnostic registry; and require buffered rendering plus
fault/result-matrix goldens (ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-28 — remove superseded identity ambiguity

**Reason:** the fourth fresh-context review found that earlier prose remained
normative beside the final amendment, failed prepare writes could not consume a
recoverable sequence, legacy public IDs were underspecified, and journal
recovery omitted conflicting terminal histories.

**Delta:** replace the earlier incarnation, rank-3, single-row journal, and
degree-one clauses with final-contract references; consume sequences only after
durable preparation; define post-v1, legacy-ledger, and legacy-config ID tuples;
make source ordinals nullable/null-first; and add the exhaustive journal state
table plus recovery/fault tests (ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).
