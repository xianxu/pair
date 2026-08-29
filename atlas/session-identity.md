# Session identity and storage

Pair separates identities that used to be partly conflated:

- **Repo scope** — a hidden, stable key derived from the cleaned repo root. It
  owns the scoped data directory and is not shown in user-facing labels.
- **Thread tag** — the immutable repo-local storage key. Direct Pair keeps the
  user-chosen form (`work`, `bugfix`); Couch creates opaque
  `couch-<16 lowercase hex>` tags.
- **Human thread name** — optional mutable metadata on the durable ThreadStore
  record. It is neither a filename nor a zellij socket name.
- **Public session name** — the stable zellij socket binding recorded in
  `session-names.jsonl` for one `{scope, tag}`.
- **Agent** — the resource running under a tag, such as `claude`, `codex`,
  `agy`, or `muse`. A tag can have sessions from more than one agent over time.
- **Native session id** — the agent's own resumable conversation id. Fresh
  launches expose it as recovery state only after Pair establishes a completed
  causal round; an explicit scanner-authorized resume may establish it at the
  launch boundary.

## Native-session forest inventory

`cmd/internal/sessioninventory` is the single model and scanner boundary for
native session storage (#155 M1). Its versioned Claude, Codex, Agy, and Muse
scanners emit facts into a deterministic forest: complete roots, validated
native parent/child edges, and explicit unbound orphans. Missing, conflicting,
malformed, unreadable, or unknown-schema evidence is retained as a stable coded
diagnostic rather than guessed away. Stable IDs, ordering, chronology fallback,
artifact paths, and a forest-only canonical projection are pure functions.

All native I/O crosses one injected runtime: named storage roots, bounded file
reads, read-only SQLite, and process/open-file snapshots. The sibling
`sessioninventorytest` package supplies a persistent stateful fake, while
`make test-native-session-live` checks installed native shapes without printing
paths, IDs, or transcript content. Native parentage establishes topology only;
it is not evidence that a Pair tag owns a root.

#156 makes that boundary incremental. The selected scope owns
`session-inventory-catalog.json`, a versioned catalog of scanner facts,
filesystem generation/mutation fingerprints, parser-complete offsets, and
scanner state. Launch records contain metadata-only artifact exclusion
boundaries; binding records contain a complete authorization proof. Unchanged
catalog/proof generations need no body read, trusted append-only JSONL stores
validate only their suffix, and replacement, truncation, unavailable generation,
schema drift, or corruption fail closed for the targeted entry.

`pair session-watch` observes only post-launch candidates and appended bytes.
A production `IncrementalInventory` façade reconciles metadata with the catalog;
fresh launches can inspect only `new` delta entries, while an already authorized
target advances from its proof/catalog cursor. Raw launch boundaries retain the
full continuity tuple as exclusion evidence; they never authorize reads from an
unbound preexisting append. Established and explicit targets advance from their
proof parser cursor instead of a launch-relative cursor.
A unique exact operator turn followed by assistant/tool/error progress proposes
the root; the watcher persists catalog state before appending a proof-bearing
binding while the launch ordinal is still current. Repeated matches remain
ambiguous and no timestamp, traversal order, first/newest file, or native parent
edge breaks the tie. A proofless legacy binding stays unavailable to automatic
consumers until one keyed background migration validates its named root; an
explicit resume may do that one-root validation synchronously. The watcher owns
durable background proof publication, and ledger projection selects the newest
same-root binding so the upgrade becomes visible. An unbound v1 launch has no
safe metadata boundary and therefore stops without a compatibility corpus scan.

`pair session-inventory [--agent ...] [--scope current|all] [--json]
[--conformance]` exposes the canonical forests, correlations, ambiguities, and
coded diagnostics. A dedicated public DTO keeps schema v1 exact: internal root
coordinates do not leak into evidence, and required position/fingerprint arrays
remain arrays even when empty. Conformance emits only agent/status/count/code
data. Pair's Go store and Neovim history navigation share the versioned
byte-counted log grammar while retaining legacy entries, so authored Markdown
separators round-trip.

Inventory queries remain the only native-session read authority. Context/token
usage, title activity, bounded slug text events, review scoping, launcher
recovery/resume hints, and changelog keying consume an established owner
projection by reading one ledger and its proof-named artifacts. The selected-
scope catalog is the shared persistent advancement owner: an accepted suffix is
published monotonically through `CatalogStore`, and later unchanged queries
reuse that parser cursor without rereading body bytes. Catalog loss falls back
to the durable ledger proof. Neovim's review fallback uses the bounded `--owner`
projection rather than the diagnostic whole-inventory rendering. Provisional, ambiguous, and unbound
owners remain explicit absence; only an exact inherited `PAIR_SESSION_ID` can
precede that projection. Compatibility config retains launch arguments but
cannot establish identity. Alt+X reads local sidecars and paints its confirmation
without starting inventory/activity work; age/idle enrichment is omitted from
the modal. `make test-session-inventory-conformance` runs the one-second installed
metadata budget and all four provider comparisons. Run it for #156 verification,
before any scanner/provider-contract version change, and in the monthly operator
maintenance pass (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

## Data layout

The global Pair data root is still `${XDG_DATA_HOME:-~/.local/share}/pair`.
Repo-scoped launch state lives under:

```text
<global>/repos/<scope-key>/
```

Tag sidecars keep their exact durable tag inside that scope:

```text
draft-<tag>.md
log-<tag>.md
queue-<tag>/
agent-<tag>
config-<tag>-<agent>.json
agent-default-<agent>.json
agent-ready-<tag>-<agent>.json
ledger-<tag>.jsonl
session-inventory-catalog.json
scrollback-<tag>-<agent>.raw
scrollback-<tag>-<agent>.events.jsonl
pane-<tag>-<agent>.json
```

Those names are descriptive storage vocabulary, not construction instructions.
Current code obtains them only from the `artifactpath` methods and exact
environment bindings below.

`cmd/internal/artifactpath` is the constructor authority for the complete
family list (including review, continuation, PID, parked, image, layout, and
diagnostic sidecars not repeated above). The launcher resolves the composite
address once, validates every result remains below the selected scope, and
exports exact `PAIR_*_PATH` bindings. Shell, Neovim, and KDL consumers use those
bindings directly; they do not combine `PAIR_DATA_DIR` and `PAIR_TAG`.

## Public session names

Zellij session names are globally visible, so Pair assigns a readable public
name through `session-names.jsonl` in the selected repository scope. The format
is:

```text
📁{repo}[-{residual tag tokens}]
```

Reads merge the former global index before the selected-scope index so live
pre-M5 sessions remain visible during upgrade; new rows are written only to the
selected scope. Missing files mean no bindings, while malformed or unreadable
present files fail closed before attachment, rename, restart, or orphan cleanup.

The first `pair/work` session becomes `📁pair-work`; a `pair` tag in the `pair`
repo becomes just `📁pair`, because a tag token already carried by the repo is
dropped. `parley.nvim` with tag `parley_nvim` becomes `📁parley-nvim`.

Three rules produce it:

1. **Repo** is the first alphanumeric token of the normalized display name
   (`parley.nvim` → `parley`).
2. **Residual** is the tag's tokens with a leading token matching the repo
   dropped — exactly one token, not the whole run, so tags `pair-x` and
   `pair-pair-x` stay distinct names.
3. **`📁` is the ownership prefix**, 4 bytes and needing no separator, where the
   previous `pair-` cost 5.

A second repo with the same display repo name and same tag gets a stable numeric
suffix, for example `📁pair-work-2`. The hidden scope key is stored in the index
row, not embedded in zellij names, picker rows, titles, or pane text.

### Why the prefix is load-bearing

The prefix is Pair's ownership marker in zellij's **global** namespace. It is
what keeps `delete-session --force` off a stranger's abandoned session — the
global list routinely contains foreign names. Discovery accepts **both** `📁` and
the legacy `pair-` (`isPairSessionName`); only `📁` is ever emitted.

### The budget is discovered, not assumed

A session name is a **socket filename**. On the machine this was measured on,
macOS allows **24 bytes** — and that number is the socket path's, so it varies
with username and is different on Linux (`~/.cache/zellij`). Pair therefore
treats zellij's own validator as the acceptance oracle (`ProbeSessionName`); the
numeric budget is calibrated lazily, only after a rejection, purely so the
refusal message can quote real numbers.

Three units are in play and each answers a different question — mixing them was
the original bug:

| unit | question |
|---|---|
| bytes | will zellij accept this socket name? |
| runes | where may a string be cut without splitting a character? |
| columns | how wide is this in `pair list`? (`📁` is 1 rune, 2 columns) |

### Overflow: refuse, then drop whole tokens

An overlong name is **refused at the create prompt**, quoting the real limit,
rather than silently shortened — silent truncation is what produced
`pair-parley_nv-parley_nv`, which its owner could not explain.

Where shortening still happens (non-interactive paths, which have no prompt to
refuse at), the ladder drops residual tag tokens **whole**, from the right,
before truncating anything; only once no residual is left does the repo token
shrink, by rune, to a 4-byte floor. The ladder resolves **length only** —
collisions go to the numeric suffix, because a shorter name is some other tag's
natural name.

### The name is not invertible

Rules 1 and 2 discard information, so a tag cannot be recovered from a `📁` name
by string surgery. `session-names.jsonl` is the only inverse
(`TagForSessionName`), and a `📁` name absent from it yields **no** tag rather
than a plausible-looking wrong one. The legacy `pair-<tag>` form *is* invertible
and keeps its `TrimPrefix` fallback — that is a different scheme, not a shortcut.

### Migration from `pair-`

A ledger row pins a name only once it is already `📁`-prefixed; a legacy row
falls through and re-mints. The superseded zellij record is reclaimed at the
create flow's commit point, and **only when already `EXITED`** — an attached
session is someone's live terminal and a detached one is resumable work, so
migration is never what destroys either.

Pair cannot rename a live zellij session underneath itself, so a running session
migrates by being quit and relaunched.

## Independent Pair and Couch authorities

Pair and Couch deliberately have two independent durable authorities:

- Pair owns `{repo scope, tag}` address claims, scoped artifacts, ledgers, and
  public zellij session bindings. Direct Pair establishes its own claim before
  writing artifacts. A Couch-hosted Pair changes only Couch's pre-reserved claim
  to `established`; the marker is exact registration evidence, not metadata.
- Couch owns `threadstore/manifest.json` and the addressed records under
  `threadstore/records/<scope>/<tag>.json`. ThreadStore alone owns lifecycle,
  admission, mutable human names, descriptions, working paths, and recovery.

The composed boundary preserves both owners: Pair establishes its marker before
the zellij handoff without touching Couch files; Couch observes that evidence
and then performs the creating→live transition for the exact helper identity.
Malformed, mismatched, invalid, or unreadable markers are unknown evidence and
fail closed; missing and reserved markers are absent evidence.

Standalone Pair does not open or upsert Couch's ThreadStore. `pair resume`
addresses an exact Pair tag (with Pair's own ledger permitted to invert a public
`📁...` session name), and Pair's picker uses Pair-owned live bindings and tag
history. Couch's mutable names and paths remain Couch-only resolution inputs:
they neither decorate the Pair picker nor become Pair resume addresses
(ARCH-DRY, ARCH-PURPOSE, ARCH-PURE).

### Session names are also filename components

`quit-<session>` and `restart-<session>` markers embed the name, so `📁` now
appears in filenames under `~/.cache/pair`. `artifactpath.ResolvePairCache`
owns their construction. Its session-name contract accepts Unicode basenames
while rejecting empty names, traversal, and NUL; strict ASCII validation for
thread tags remains unchanged.

## Ledger and caches

Each tag has an append-only `ledger-<tag>.jsonl` in its scope dir. Current typed
rows are a `launch`/`binding` union: physical line ordinal is the generation
key, and a binding is current only when it joins the newest exact
`{scope,tag,agent}` launch. The shared locked store owns append/fsync; malformed
lines consume their ordinal instead of being silently reused. Historical
launcher rows remain readable during migration.

Authority publication has one result vocabulary across the typed and
compatibility ledgers: an incomplete unterminated row is non-authoritative; a
complete row whose file/directory durability is uncertain is indeterminate and
is reconciled by exact physical ordinal plus encoded bytes; a cleanup failure
after durability is committed and does not roll lifecycle state back. Launcher
and watcher consumers preserve those outcomes rather than treating every error
as a missing row.

Operator-authored Pair-log entries use the same publication rule around two
atomic replacements. Each Neovim submission attempt carries a stable opaque
`append_id` in the byte-counted marker. The first replacement records
`state=prepared` before dispatch; the parser retains it for audit but excludes
it from correlation facts. A normal dispatch is followed by an exact-ID
`state=submitted` replacement, and only then can the entry match a native user
turn. An unchanged retry reuses its ID, while edited, cleared, indeterminate,
and compose-without-submit preparations remain permanently ineligible rather
than claiming an input occurred. After success, even identical later authored
text receives a new ID.

The submitted transition is also gated by the production Zellij action result,
not merely by calling the send function. Focus, write, and submit failure leave
the attempt prepared and the authored draft intact. Refocus failure after a
successful submit is a UI warning rather than a delivery rollback. If the
submitted-marker replacement then fails, Neovim retains that dispatched append
ID and performs commit-only recovery before accepting another authored send;
the original body is never dispatched twice.

Before dispatch, the editor retains a finer delivery phase. `written` means the
exact agent-facing body is already staged and retry may execute only the pending
submit/compose action; changing the authored body is blocked until that state
resolves. `indeterminate` means a failed write might have partially affected the
composer and automatic retry is refused. `composed` is a completed transfer but
never submitted evidence. These phases prevent replayed Zellij effects from
changing the native input relative to its Pair-log body.

The typed joined ledger binding is the source of truth for native recovery.
The older `agent-<tag>` and `config-<tag>-<agent>.json` files remain derived
caches and compatibility surfaces; config disagreement is diagnosed and cannot
override a current ledger generation.

### Codex root identity

A Codex rollout filename supplies only a candidate UUID; it does not prove
which conversation owns the rollout. Pair authorizes an automatic Codex
identity only when the rollout's first JSONL event is a matching
`session_meta`, its `parent_thread_id` is absent or null, and its source is the
observed root source `cli` or `exec`. Subagent, malformed, mismatched, unknown,
oversized, and incomplete first events fail closed. Candidate scans continue
past rejected rollouts so an open subagent cannot hide a later root candidate.

The rule lives in `cmd/internal/sessioninventory` and is shared by launcher,
watcher, context, slug, title, opener, and review queries. Process/open-file
evidence can corroborate a causal-round candidate but cannot select one.
Persisted config IDs are compatibility evidence only: an unavailable binding is
removed from config, its non-resume args are preserved for a fresh launch, and
the operator is warned. Explicit inherited invocation authority remains exact.

Neovim deliberately does not inspect Codex processes or rollouts. Review
target scoping uses the inherited `PAIR_SESSION_ID`, then the established
inventory projection; when neither exists it remains unscoped until the watcher
publishes a validated root binding.

`agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
has only `{agent,args}` and belongs to the repo/agent, not to a work tag or
native conversation. Fresh `pair <agent>` creates use it as the lowest-priority
argument source after explicit `-- <args>` and tag-specific config. It is written
only after the launched `pair wrap` child publishes a matching
`agent-ready-<tag>-<agent>.json` record for the launch nonce.

## Picker and list scope

Default picker/list views are current-repo scoped:

- live sessions are included only when the current scope's
  `session-names.jsonl` maps their public name to the current scope key;
- picker rows use Pair's repo/tag history and live session bindings; Couch human
  names and paths never decorate them, and selection always retains the tag;
- `pair <agent>` marks different-agent live rows unavailable and switches a
  different-agent historical tag to the requested driver, seeding from a
  matching continuation doc when present or an auto-continuation draft over
  Pair's tag files and parked scrollback when not;
- unindexed live `pair-*` sessions are treated as legacy candidates, not proof
  that they belong to the current repo;
- a legacy `pair-*` session and a new `📁` one coexist in one snapshot, both
  discoverable with the right tag.

## Legacy flat data

Flat sidecars under the global root are not silently claimed. If a flat tag is
ambiguous but matches the current repo basename family, Pair shows a manual row:

```text
legacy unscoped <tag>  (manual import)
```

Selecting it copies missing flat sidecars into the current repo scope, including
queued prompt files, preserves the flat source files, avoids overwriting scoped
files, and writes a ledger row with `legacy_import: true`.
